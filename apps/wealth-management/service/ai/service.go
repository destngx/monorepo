// Package ai implements the AI use cases (§10): one function per feature,
// each with its prompt keys, deterministic input builder, output contract
// and cache policy, on top of the provider-agnostic port.AIClient.
package ai

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
)

const (
	// resultTTL caches AI results by request hash (WM-BUD-5); the hash covers
	// the data, so a sheet change produces a new key.
	resultTTL      = 30 * 60
	cachePrefix    = "ai:"
	chatTimeout    = 5 * time.Minute
	maxChatSteps   = 6
	maxChatHistory = 20
)

// RequiredPrompts are the sheet keys the features need (WM-AI-1).
var RequiredPrompts = []string{
	keyIdentity,
	"tasks/account-review", "actions/account-review",
	"tasks/credit-summary", "actions/credit-summary",
	"tasks/transaction-review", "actions/transaction-review",
	"tasks/year-review", "actions/year-review",
	"tasks/intelligence-briefing", "actions/intelligence-briefing",
	keyAdvisorTask, "actions/budget-advisor",
	keyHealthTask, "actions/financial-health",
	keyChartTask, "actions/chart-insight",
	"tasks/cashback-explain", "actions/cashback-explain",
	"tasks/suggest-category", "actions/suggest-category",
}

const (
	keyIdentity    = "system/identity"
	keyAdvisorTask = "budget/advisor"
	keyHealthTask  = "finance/health"
	keyChartTask   = "dashboard/insight"
	keyChatTask    = "tasks/chat"
)

type Options struct {
	// Now is the clock, for tests.
	Now func() time.Time
	// CaptureLimits bound Smart Capture batches; zero uses the defaults.
	CaptureLimits domain.CaptureLimits
	// Search, Market and Portfolio enable the market features (WM-AI-9/14,
	// WM-INV-17/18); each is optional and its features fail as unavailable.
	Search    port.SearchService
	Market    port.MarketAnalysisService
	Portfolio port.PortfolioService
	// Recommendations, when set, receives what the reviews and the briefing
	// recommend and gives them the earlier items to check (WM-REC-2, 4).
	Recommendations port.RecommendationService
	// Close gives the briefing the net worth history (ROADMAP P3); optional.
	Close port.MonthlyCloseService
}

type Service struct {
	client    port.AIClient
	db        port.DatabaseService
	analytics port.AnalyticsService
	cache     port.CacheService
	config    domain.AIConfig
	masker    domain.PIIMasker
	log       *logger.Logger
	now       func() time.Time

	captureLimits domain.CaptureLimits
	search        port.SearchService
	market        port.MarketAnalysisService
	portfolio     port.PortfolioService
	recommend     port.RecommendationService
	close         port.MonthlyCloseService
}

var _ port.AIService = (*Service)(nil)

// New returns the AI service. cache may be nil (results are then not cached).
func New(client port.AIClient, db port.DatabaseService, analytics port.AnalyticsService, cache port.CacheService, config domain.AIConfig, log *logger.Logger, options Options) *Service {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.CaptureLimits == (domain.CaptureLimits{}) {
		options.CaptureLimits = domain.DefaultCaptureLimits()
	}
	return &Service{
		client:    client,
		db:        db,
		analytics: analytics,
		cache:     cache,
		config:    config,
		masker:    domain.NewPIIMasker(config.PIINames),
		log:       log,
		now:       options.Now,

		captureLimits: options.CaptureLimits,
		search:        options.Search,
		market:        options.Market,
		portfolio:     options.Portfolio,
		recommend:     options.Recommendations,
		close:         options.Close,
	}
}

func (s *Service) today() time.Time {
	return domain.Today(s.now())
}

// ProviderUsage reports the AI provider's quota through the gateway (WM-SET-4).
// It needs only the endpoint, so it works before any model is chosen.
func (s *Service) ProviderUsage(ctx context.Context) (domain.AIProviderUsage, error) {
	if strings.TrimSpace(s.config.BaseURL) == "" {
		return domain.AIProviderUsage{}, fmt.Errorf("%w: set AI_BASE_URL", domain.ErrAIUnavailable)
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	return s.client.Usage(ctx)
}

// ProviderModels lists the endpoint's models so Settings can offer them. Like
// ProviderUsage it needs only the endpoint.
func (s *Service) ProviderModels(ctx context.Context) ([]string, error) {
	if strings.TrimSpace(s.config.BaseURL) == "" {
		return nil, fmt.Errorf("%w: set AI_BASE_URL", domain.ErrAIUnavailable)
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	return s.client.Models(ctx)
}

func (s *Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.config.Timeout > 0 {
		return context.WithTimeout(ctx, s.config.Timeout)
	}
	return ctx, func() {}
}

func (s *Service) Status(ctx context.Context) domain.AIStatus {
	status := domain.AIStatus{
		Configured:    s.config.Configured(),
		Provider:      domain.AIProvider,
		Endpoint:      endpointHost(s.config.BaseURL),
		Models:        s.config.Models,
		FallbackModel: s.config.FallbackModel,
		MissingKeys:   []string{},
		Capture:       s.captureLimits,
	}
	content, err := s.db.LoadAIContent(false)
	if err != nil {
		status.PromptsError = err.Error()
		return status
	}
	status.Prompts = len(content.Prompts)
	keys := append(append([]string{}, RequiredPrompts...), capturePrompts...)
	if s.market != nil {
		keys = append(keys, marketPrompts...)
	}
	for _, key := range keys {
		if strings.TrimSpace(content.Prompts[key]) == "" {
			status.MissingKeys = append(status.MissingKeys, key)
		}
	}
	return status
}

// endpointHost reports scheme://host only, never a path or credentials.
func endpointHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

// call is one model invocation with its observability metadata (WM-AI-15).
type call struct {
	feature  string
	useCase  domain.AIUseCase
	model    string
	prompts  []string
	messages []domain.AIMessage
	json     bool
	// validate checks the reply; a failure retries with the fallback model.
	validate func(content string) error
}

func (s *Service) models(useCase domain.AIUseCase, override string) []string {
	primary := strings.TrimSpace(override)
	if primary == "" {
		primary = s.config.Models[useCase]
	}
	models := []string{primary}
	if fallback := s.config.FallbackModel; fallback != "" && fallback != primary {
		models = append(models, fallback)
	}
	return models
}

func (s *Service) requireConfigured() error {
	if !s.config.Configured() {
		return fmt.Errorf("%w: set AI_BASE_URL and AI_MODEL", domain.ErrAIUnavailable)
	}
	return nil
}

// complete runs a call, retrying once on the fallback model when the
// provider fails or the reply does not validate.
func (s *Service) complete(ctx context.Context, c call) (domain.AIResponse, error) {
	if err := s.requireConfigured(); err != nil {
		return domain.AIResponse{}, err
	}
	var lastErr error
	for _, model := range s.models(c.useCase, c.model) {
		attemptCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		started := time.Now()
		response, err := s.client.Complete(attemptCtx, domain.AIRequest{Model: model, Messages: c.messages, JSON: c.json})
		cancel()
		if err == nil && c.validate != nil {
			err = c.validate(response.Content)
		}
		s.logCall(ctx, c.feature, model, c.prompts, time.Since(started), response.Usage, err)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	return domain.AIResponse{}, lastErr
}

// logCall records model, prompt keys and latency, never content (WM-AI-15).
func (s *Service) logCall(ctx context.Context, feature string, model string, promptKeys []string, latency time.Duration, usage domain.AIUsage, err error) {
	attrs := []slog.Attr{
		slog.String("feature", feature),
		slog.String("model", model),
		slog.String("prompts", strings.Join(promptKeys, ",")),
		slog.Int64("latency_ms", latency.Milliseconds()),
		slog.Int("prompt_tokens", usage.PromptTokens),
		slog.Int("completion_tokens", usage.CompletionTokens),
	}
	if err != nil {
		s.log.LogError(ctx, "ai call failed", err, attrs...)
		return
	}
	s.log.LogApplicationEvent(ctx, "ai call", attrs...)
}

// cached returns a stored result for the request, or runs compute and stores it.
func cached[T any](ctx context.Context, s *Service, feature string, force bool, key any, compute func() (T, error)) (T, error) {
	cacheKey := cachePrefix + feature + ":" + hashOf(key)
	if s.cache != nil && !force {
		entry, err := s.cache.Get(ctx, cacheKey)
		if err != nil {
			s.log.LogError(ctx, "ai cache read failed", err, slog.String("feature", feature))
		} else if entry.Found {
			var value T
			if json.Unmarshal([]byte(entry.Value), &value) == nil {
				return value, nil
			}
		}
	}
	value, err := compute()
	if err != nil {
		return value, err
	}
	if s.cache != nil {
		if payload, err := json.Marshal(value); err == nil {
			if err := s.cache.Set(ctx, cacheKey, string(payload), resultTTL); err != nil {
				s.log.LogError(ctx, "ai cache write failed", err, slog.String("feature", feature))
			}
		}
	}
	return value, nil
}

func hashOf(value any) string {
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:12])
}

// IsUserFacing reports errors whose message is safe and useful to show.
func IsUserFacing(err error) bool {
	return errors.Is(err, domain.ErrAIUnavailable) || errors.Is(err, domain.ErrAIOutput) || errors.Is(err, domain.ErrPromptMissing) || errors.Is(err, domain.ErrValidation)
}

func toJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(payload)
}
