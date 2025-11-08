package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// CallbackResult holds the result from OAuth callback
type CallbackResult struct {
	Code  string
	State string
	Error string
}

// CallbackServer handles OAuth callback HTTP requests
type CallbackServer struct {
	server   *http.Server
	listener net.Listener
	result   chan CallbackResult
	once     sync.Once
}

// NewCallbackServer creates a new OAuth callback server
func NewCallbackServer(port int) (*CallbackServer, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	cs := &CallbackServer{
		listener: listener,
		result:   make(chan CallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", cs.handleCallback)
	mux.HandleFunc("/", cs.handleRoot)

	cs.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return cs, nil
}

// Start starts the callback server
func (cs *CallbackServer) Start() error {
	go func() {
		if err := cs.server.Serve(cs.listener); err != nil && err != http.ErrServerClosed {
			cs.result <- CallbackResult{Error: err.Error()}
		}
	}()
	return nil
}

// WaitForCallback waits for OAuth callback with timeout
func (cs *CallbackServer) WaitForCallback(timeout time.Duration) (CallbackResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case result := <-cs.result:
		if result.Error != "" {
			return result, fmt.Errorf("callback error: %s", result.Error)
		}
		return result, nil
	case <-ctx.Done():
		return CallbackResult{}, fmt.Errorf("timeout waiting for OAuth callback")
	}
}

// Stop stops the callback server
func (cs *CallbackServer) Stop() error {
	var err error
	cs.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = cs.server.Shutdown(ctx)
		close(cs.result)
	})
	return err
}

// GetRedirectURL returns the callback URL for this server
func (cs *CallbackServer) GetRedirectURL() string {
	return fmt.Sprintf("http://%s/callback", cs.listener.Addr().String())
}

// handleCallback handles the OAuth callback request
func (cs *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	if errorParam != "" {
		cs.result <- CallbackResult{
			Error: fmt.Sprintf("%s: %s", errorParam, errorDesc),
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
	<h1>❌ Authentication Failed</h1>
	<p>Error: %s</p>
	<p>%s</p>
	<p>You can close this window.</p>
</body>
</html>
`, errorParam, errorDesc)
		return
	}

	if code == "" {
		cs.result <- CallbackResult{Error: "missing authorization code"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
	<h1>❌ Authentication Failed</h1>
	<p>Missing authorization code in callback.</p>
	<p>You can close this window.</p>
</body>
</html>
`)
		return
	}

	cs.result <- CallbackResult{
		Code:  code,
		State: state,
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body>
	<h1>✅ Authentication Successful!</h1>
	<p>You have successfully authenticated with the provider.</p>
	<p>You can close this window and return to the terminal.</p>
	<script>
		setTimeout(function() {
			window.close();
		}, 3000);
	</script>
</body>
</html>
`)
}

// handleRoot handles requests to the root path
func (cs *CallbackServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head><title>OAuth Callback Server</title></head>
<body>
	<h1>OAuth Callback Server</h1>
	<p>Waiting for OAuth callback...</p>
	<p>Please complete the authentication in your browser.</p>
</body>
</html>
`)
}
