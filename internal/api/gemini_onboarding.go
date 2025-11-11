package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

const (
	geminiBaseURL       = "https://cloudcode-pa.googleapis.com/v1internal"
	loadCodeAssistRoute = ":loadCodeAssist"
	onboardUserRoute    = ":onboardUser"
	pollInterval        = 5 * time.Second
)

// ProjectIDRequiredError is returned when a project ID is needed but not provided.
// This typically happens with workspace/standard tier accounts that require a user-defined project.
type ProjectIDRequiredError struct{}

// Error returns the error message.
func (e *ProjectIDRequiredError) Error() string {
	return "This account requires setting the GOOGLE_CLOUD_PROJECT env var. Please set GOOGLE_CLOUD_PROJECT before calling setup."
}

// IsProjectIDRequired checks if an error is a ProjectIDRequiredError.
func IsProjectIDRequired(err error) bool {
	_, ok := err.(*ProjectIDRequiredError)
	return ok
}

// SetupUserProject performs the full onboarding flow and returns the Google Cloud project ID that
// the user belongs to. It will create a new project for the user if necessary and poll
// for the long‑running onboard operation to finish.
//
// The method is analogous to the TypeScript implementation in
// llxprt-code/packages/core/src/code_assist/setup.ts.
func (c *GeminiClient) SetupUserProject(ctx context.Context) (string, error) {
	// Fetch project ID from env if present.
	var projectID *string
	if id := os.Getenv("GOOGLE_CLOUD_PROJECT"); id != "" {
		projectID = &id
	}

	metadata := ClientMetadata{
		IDEType:    IDETypeUnspecified,
		Platform:   PlatformUnspecified,
		PluginType: PluginTypeGemini,
	}

	// Load current state.
	loadRes, err := c.loadCodeAssist(ctx, projectID, metadata)
	if err != nil {
		return "", fmt.Errorf("loadCodeAssist failed: %w", err)
	}

	// Debug: Log the full loadCodeAssist response
	logger.Debugf("Gemini: loadCodeAssist response - CurrentTier: %v, CloudaicompanionProject: %v, AllowedTiers count: %d",
		loadRes.CurrentTier != nil, loadRes.CloudaicompanionProject, len(loadRes.AllowedTiers))
	if loadRes.CurrentTier != nil {
		logger.Debugf("Gemini: Current tier ID: %s", loadRes.CurrentTier.ID)
	}
	for i, tier := range loadRes.AllowedTiers {
		isDefault := "nil"
		if tier.IsDefault != nil {
			isDefault = fmt.Sprintf("%v", *tier.IsDefault)
		}
		userDefined := "nil"
		if tier.UserDefinedCloudaicompanionProject != nil {
			userDefined = fmt.Sprintf("%v", *tier.UserDefinedCloudaicompanionProject)
		}
		logger.Debugf("Gemini: AllowedTier[%d] - ID: %s, Name: %s, IsDefault: %s, UserDefinedProject: %s",
			i, tier.ID, tier.Name, isDefault, userDefined)
	}

	// If user already has tier and project, return it.
	if loadRes.CurrentTier != nil {
		// Project from response, if any.
		if loadRes.CloudaicompanionProject != nil && *loadRes.CloudaicompanionProject != "" {
			logger.Debugf("Gemini: User has currentTier, returning project from response: %s", *loadRes.CloudaicompanionProject)
			return *loadRes.CloudaicompanionProject, nil
		}
		// Fallback to env project ID if provided.
		if projectID != nil && *projectID != "" {
			logger.Debugf("Gemini: User has currentTier but no project in response, using env project ID: %s", *projectID)
			return *projectID, nil
		}
		logger.Debugf("Gemini: User has currentTier but no project available")
		return "", &ProjectIDRequiredError{}
	}

	// No current tier, determine which tier to onboard.
	tier := getOnboardTier(loadRes)
	if tier == nil {
		return "", fmt.Errorf("no onboard tier found")
	}

	if tier.UserDefinedCloudaicompanionProject != nil && *tier.UserDefinedCloudaicompanionProject && projectID == nil {
		return "", &ProjectIDRequiredError{}
	}

	// Prepare onboard request.
	onboardReq := OnboardUserRequest{
		TierID: &tier.ID,
		Metadata: &metadata,
	}
	if tier.ID == UserTierIDFree {
		// Free tier uses managed project; skip explicit project ID.
		onboardReq.CloudaicompanionProject = nil
	} else {
		onboardReq.CloudaicompanionProject = projectID
		// Include duetProject in metadata for non‑free tiers.
		metadata.DuetProject = *projectID
		onboardReq.Metadata = &metadata
	}

	// Call onboardUser and poll until done.
	lro, err := c.onboardUser(ctx, onboardReq)
	if err != nil {
		return "", fmt.Errorf("onboardUser failed: %w", err)
	}
	for !lro.Done {
		logger.Debugf("Gemini: onboardUser LRO not done, sleeping %s", pollInterval)
		time.Sleep(pollInterval)
		lro, err = c.onboardUser(ctx, onboardReq)
		if err != nil {
			return "", fmt.Errorf("while polling onboardUser: %w", err)
		}
	}

	// Inspect response for the project ID.
	if lro.Response != nil && lro.Response.CloudaicompanionProject != nil && lro.Response.CloudaicompanionProject.ID != "" {
		return lro.Response.CloudaicompanionProject.ID, nil
	}

	if projectID != nil && *projectID != "" {
		return *projectID, nil
	}

	return "", &ProjectIdRequiredError{}
}

// loadCodeAssist calls the loadCodeAssist endpoint and returns the response.
func (c *GeminiClient) loadCodeAssist(ctx context.Context, projectID *string, metadata ClientMetadata) (*LoadCodeAssistResponse, error) {
	logger.Debugf("Gemini: Calling loadCodeAssist")
	reqBody := LoadCodeAssistRequest{
		CloudaicompanionProject: projectID,
		Metadata:                metadata,
	}
	resp, err := c.doRequest(ctx, "POST", loadCodeAssistRoute, reqBody)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loadCodeAssist returned %d: %s", resp.StatusCode, string(body))
	}

	var loadRes LoadCodeAssistResponse
	if err := json.NewDecoder(resp.Body).Decode(&loadRes); err != nil {
		return nil, fmt.Errorf("failed to decode loadCodeAssist response: %w", err)
	}
	return &loadRes, nil
}

// onboardUser calls the onboardUser endpoint and returns the LRO response.
func (c *GeminiClient) onboardUser(ctx context.Context, req OnboardUserRequest) (*LongRunningOperationResponse, error) {
	logger.Debugf("Gemini: Calling onboardUser")
	resp, err := c.doRequest(ctx, "POST", onboardUserRoute, req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("onboardUser returned %d: %s", resp.StatusCode, string(body))
	}

	var lroResp LongRunningOperationResponse
	if err := json.NewDecoder(resp.Body).Decode(&lroResp); err != nil {
		return nil, fmt.Errorf("failed to decode onboardUser response: %w", err)
	}
	return &lroResp, nil
}

// getOnboardTier chooses the default tier from the loadCodeAssist response or
// returns a minimal fallback if none is marked default.
func getOnboardTier(res *LoadCodeAssistResponse) *GeminiUserTier {
	if res == nil {
		return nil
	}
	for i := range res.AllowedTiers {
		tier := &res.AllowedTiers[i]
		if tier.IsDefault != nil && *tier.IsDefault {
			return tier
		}
	}
	// Fallback: return legacy tier with userDefinedCloudaicompanionProject true.
	return &GeminiUserTier{
		ID:                         UserTierIDLegacy,
		Name:                       "",
		Description:                "",
		UserDefinedCloudaicompanionProject: boolPtr(true),
		IsDefault:                  boolPtr(false),
	}
}

func boolPtr(b bool) *bool { return &b }