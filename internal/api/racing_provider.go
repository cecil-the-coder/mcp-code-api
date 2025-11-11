package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

type raceResult struct {
	code          string
	usage         *types.Usage
	providerModel string
	duration      time.Duration
}

type RacingProvider struct {
	config          *config.RacingConfig
	configRef       *config.Config
	lastWinner      string
	lastCompletions map[string]time.Duration
	mu              sync.RWMutex
}

func NewRacingProvider(cfg *config.RacingConfig, configRef *config.Config) *RacingProvider {
	return &RacingProvider{
		config:          cfg,
		configRef:       configRef,
		lastCompletions: make(map[string]time.Duration),
	}
}

func (r *RacingProvider) parseProviderModel(s string) (providerName, modelName string, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid provider:model format: %q", s)
	}

	providerNameOrAlias := strings.TrimSpace(parts[0])
	modelName = strings.TrimSpace(parts[1])

	actualProvider := r.resolveProviderName(providerNameOrAlias)
	return actualProvider, modelName, nil
}

func (r *RacingProvider) resolveProviderName(nameOrAlias string) string {
	if r.configRef.Providers.Anthropic != nil && r.configRef.Providers.Anthropic.DisplayName == nameOrAlias {
		return "anthropic"
	}
	if r.configRef.Providers.Cerebras != nil && r.configRef.Providers.Cerebras.DisplayName == nameOrAlias {
		return "cerebras"
	}
	return nameOrAlias
}

func (r *RacingProvider) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (*types.CodeGenerationResult, error) {
	models := r.config.Models
	if len(models) == 0 {
		return nil, fmt.Errorf("no models configured for racing")
	}
	numRacers := r.config.NumRacers
	if numRacers > 0 && int(numRacers) < len(models) {
		models = models[:numRacers]
	}
	logger.Infof("Racing %d models: %v", len(models), models)
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultChan := make(chan raceResult, 1)
	errChan := make(chan error, len(models))
	r.mu.RLock()
	gracePeriod := time.Duration(r.config.GracePeriodMS) * time.Millisecond
	if gracePeriod <= 0 {
		gracePeriod = 500 * time.Millisecond
	}
	r.mu.RUnlock()
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(len(models))
	for _, providerModel := range models {
		go func(pm string) {
			defer wg.Done()
			providerName, modelName, err := r.parseProviderModel(pm)
			if err != nil {
				logger.Errorf("[%s] parse error: %v", pm, err)
				select {
				case errChan <- fmt.Errorf("[%s] parse error: %w", pm, err):
				case <-cancelCtx.Done():
				}
				return
			}
			var code string
			var usage *types.Usage
			var clientErr error
			switch providerName {
			case "anthropic":
				if r.configRef.Providers.Anthropic == nil {
					clientErr = fmt.Errorf("anthropic provider config not found")
				} else {
					var result *types.CodeGenerationResult
					result, clientErr = NewAnthropicClient(*r.configRef.Providers.Anthropic).GenerateCode(cancelCtx, prompt, contextStr, outputFile, language, contextFiles)
					if clientErr == nil {
						code = result.Code
						usage = result.Usage
					}
				}
			case "cerebras":
				if r.configRef.Providers.Cerebras == nil {
					clientErr = fmt.Errorf("cerebras provider config not found")
				} else {
					var result *types.CodeGenerationResult
					result, clientErr = NewCerebrasClient(*r.configRef.Providers.Cerebras).GenerateCode(cancelCtx, prompt, contextStr, outputFile, language, contextFiles)
					if clientErr == nil {
						code = result.Code
						usage = result.Usage
					}
				}
			case "openrouter":
				if r.configRef.Providers.OpenRouter == nil {
					clientErr = fmt.Errorf("openrouter provider config not found")
				} else {
					orcCopy := *r.configRef.Providers.OpenRouter
					orcCopy.Model = modelName
					orcCopy.Models = nil
					var result *types.CodeGenerationResult
					result, clientErr = NewOpenRouterClient(orcCopy).GenerateCode(cancelCtx, prompt, contextStr, outputFile, language, contextFiles)
					if clientErr == nil {
						code = result.Code
						usage = result.Usage
					}
				}
			case "gemini":
				if r.configRef.Providers.Gemini == nil {
					clientErr = fmt.Errorf("gemini provider config not found")
				} else {
					var result *types.CodeGenerationResult
					result, clientErr = NewGeminiClient(*r.configRef.Providers.Gemini).GenerateCode(cancelCtx, prompt, contextStr, outputFile, language, contextFiles)
					if clientErr == nil {
						code = result.Code
						usage = result.Usage
					}
				}
			default:
				clientErr = fmt.Errorf("unknown provider: %s", providerName)
			}
			if clientErr != nil {
				if !errors.Is(clientErr, context.Canceled) && !strings.Contains(clientErr.Error(), "context canceled") {
					logger.Errorf("[%s] error: %v", pm, clientErr)
					select {
					case errChan <- fmt.Errorf("[%s] error: %w", pm, clientErr):
					case <-cancelCtx.Done():
					}
				} else {
					logger.Debugf("[%s] canceled (another model won the race)", pm)
				}
				return
			}
			duration := time.Since(start)
			logger.Infof("[%s] completed in %v", pm, duration)
			select {
			case resultChan <- raceResult{code: code, usage: usage, providerModel: pm, duration: duration}:
			case <-cancelCtx.Done():
			}
		}(providerModel)
	}
	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()
	select {
	case result := <-resultChan:
		winnerResult := result.code
		winnerUsage := result.usage
		winnerTime := result.duration
		if winnerUsage != nil {
			logger.Infof("🏆 WINNER: %s in %v (tokens: %d, tokens/sec: %.1f)",
				result.providerModel, winnerTime, winnerUsage.TotalTokens,
				float64(winnerUsage.TotalTokens)/(winnerTime.Seconds()))
		} else {
			logger.Infof("🏆 WINNER: %s in %v (no token data)", result.providerModel, winnerTime)
		}

		r.mu.Lock()
		r.lastWinner = result.providerModel
		r.lastCompletions = make(map[string]time.Duration)
		r.lastCompletions[result.providerModel] = result.duration
		r.mu.Unlock()

		graceTimer := time.NewTimer(gracePeriod)
		defer graceTimer.Stop()
		for {
			select {
			case <-graceTimer.C:
				cancel()
			case res := <-resultChan:
				logger.Infof("[late completion] %s also completed during grace period in %v", res.providerModel, res.duration)
				r.mu.Lock()
				r.lastCompletions[res.providerModel] = res.duration
				r.mu.Unlock()
			case <-doneChan:
				return &types.CodeGenerationResult{Code: winnerResult, Usage: winnerUsage}, nil
			case <-ctx.Done():
				cancel()
				return nil, fmt.Errorf("race canceled: %w", ctx.Err())
			}
		}
	case <-doneChan:
		cancel()
		var errors []string
		close(errChan)
		for err := range errChan {
			errors = append(errors, err.Error())
		}
		return nil, fmt.Errorf("all %d racers failed: %s", len(models), strings.Join(errors, "; "))
	case <-ctx.Done():
		cancel()
		return nil, fmt.Errorf("race canceled: %w", ctx.Err())
	}
}

func (r *RacingProvider) GetLastWinner() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastWinner
}

func (r *RacingProvider) GetLastCompletions() map[string]time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	completions := make(map[string]time.Duration)
	for k, v := range r.lastCompletions {
		completions[k] = v
	}
	return completions
}