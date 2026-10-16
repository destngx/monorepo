package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const (
	accountsSheetRange      = "Accounts!A2:I"
	accountsAnchorRange     = "Accounts!A2:A"
	accountsFirstRow        = 2
	transactionsSheetRange  = "Transactions!A2:J"
	transactionsAnchorRange = "Transactions!A2:A"
	transactionsFirstRow    = 2
	categoriesSheetRange    = "Categories!A1:B200"
	cryptoSheetRange        = "Crypto!A1:Z100"
	fundsSheetRange         = "InvestmentFundCertificate!A1:Z100"
	promptsSheetRange       = "Prompts!A2:C"
	knowledgeSheetRange     = "Knowledge!A2:B"

	// DefaultBudgetSheetTemplate resolves the budget tab per year (WM-DATA-2).
	DefaultBudgetSheetTemplate = "Budget_{year}"
	budgetSheetColumns         = "A1:Z200"

	sheetDataTTL      = 300  // WM-RULE-10
	promptsTTL        = 3600 // WM-RULE-10
	batchReplayTTL    = 7 * 24 * 3600
	maxLocalReplays   = 256
	cacheKeyAccounts  = "accounts:all"
	cacheKeyTxs       = "transactions:all"
	cacheKeyCats      = "categories:all"
	cacheKeyInvest    = "investments:assets"
	cacheKeyAIContent = "sheets:ai-content"
)

// sheetCachePatterns lists every sheet-derived cache key (WM-OPS-3).
var sheetCachePatterns = []string{"accounts:*", "transactions:*", "budget:*", "categories:*", cacheKeyGoalPattern, cacheKeyAIContent, cacheKeyInvest, cacheKeyCaptureRules, cacheKeyCards, cacheKeyPreferences, cacheKeyRecurringPattern}

// DatabaseServiceOptions tunes the sheet service; zero values use defaults.
type DatabaseServiceOptions struct {
	// BudgetSheetTemplate names the budget tab; "{year}" is replaced.
	BudgetSheetTemplate string
	// Now is the clock, for tests.
	Now func() time.Time
	// SpreadsheetID is reported by SheetInfo.
	SpreadsheetID string
}

type dbService struct {
	client         port.DatabaseClient
	cache          port.CacheService
	log            *logger.Logger
	budgetTemplate string
	now            func() time.Time
	spreadsheetID  string

	// writeMu serialises read-compare-write operations (WM-DATA-4); the
	// client separately serialises its own first-empty-row appends.
	writeMu       sync.Mutex
	localReplays  map[string]domain.TransactionBatchResult
	replayHistory []string
	stale         *staleTracker

	// lastSyncAt is the last successful SyncCache, zero until one (WM-SET-2).
	syncMu     sync.Mutex
	lastSyncAt time.Time
}

func NewDatabaseService(client port.DatabaseClient, cache port.CacheService, log *logger.Logger, options DatabaseServiceOptions) port.DatabaseService {
	log.LogApplicationEvent(context.Background(), "initializing database service")
	if options.BudgetSheetTemplate == "" {
		options.BudgetSheetTemplate = DefaultBudgetSheetTemplate
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &dbService{
		client:         client,
		cache:          cache,
		log:            log,
		budgetTemplate: options.BudgetSheetTemplate,
		now:            options.Now,
		spreadsheetID:  options.SpreadsheetID,
		localReplays:   map[string]domain.TransactionBatchResult{},
		stale:          newStaleTracker(),
	}
}

func (s *dbService) today() time.Time {
	return domain.Today(s.now())
}

// cachedRead returns the cached value for key unless force is set, otherwise
// loads it and caches the result. Cache failures are logged and degrade to a
// direct load; they never fail the request (WM-OPS-2). A failed load serves
// the last good copy when there is one (WM-NFR-4).
func cachedRead[T any](s *dbService, key string, ttlSeconds int, force bool, load func() (T, error)) (T, error) {
	if !force {
		if value, ok := readCached[T](s, key); ok {
			return value, nil
		}
	}
	value, err := load()
	if err != nil {
		return serveLastGood[T](s, key, err)
	}
	s.storeFresh(key, value, ttlSeconds)
	return value, nil
}

// readRanges reads several ranges in one Sheets request (WM-NFR-3); rows
// come back in the order of ranges.
func (s *dbService) readRanges(ranges ...string) ([][][]string, error) {
	rows, err := s.client.ReadSheets(ranges)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", strings.Join(ranges, ", "), err)
	}
	if len(rows) != len(ranges) {
		return nil, fmt.Errorf("read %s: the sheet returned %d ranges for %d requested", strings.Join(ranges, ", "), len(rows), len(ranges))
	}
	return rows, nil
}

func readCached[T any](s *dbService, key string) (T, bool) {
	var value T
	if s.cache == nil {
		return value, false
	}
	entry, err := s.cache.Get(context.Background(), key)
	if err != nil {
		s.log.LogError(context.Background(), "cache read failed", err, slog.String("key", key))
		return value, false
	}
	if !entry.Found || strings.TrimSpace(entry.Value) == "" {
		return value, false
	}
	if err := json.Unmarshal([]byte(entry.Value), &value); err != nil {
		s.log.LogError(context.Background(), "cached value is not valid JSON, ignoring it", err, slog.String("key", key))
		var zero T
		return zero, false
	}
	return value, true
}

func (s *dbService) writeCache(key string, value any, ttlSeconds int) {
	if s.cache == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		s.log.LogError(context.Background(), "cannot encode value for cache", err, slog.String("key", key))
		return
	}
	if err := s.cache.Set(context.Background(), key, string(payload), ttlSeconds); err != nil {
		s.log.LogError(context.Background(), "cache write failed", err, slog.String("key", key))
	}
}

// invalidate drops cache keys after a successful write. A failure is logged
// rather than returned: the write already happened, and failing the request
// would invite a retry that writes twice.
func (s *dbService) invalidate(patterns ...string) {
	if s.cache == nil {
		return
	}
	for _, pattern := range patterns {
		if _, err := s.cache.Invalidate(context.Background(), pattern); err != nil {
			s.log.LogError(context.Background(), "cache invalidation failed after a sheet write; data may be stale until the TTL expires", err, slog.String("pattern", pattern))
		}
	}
}

func (s *dbService) SyncCache() (domain.SyncResult, error) {
	result := domain.SyncResult{Success: true, Message: "Cache cleared successfully"}
	if s.cache == nil {
		result.Message = "No cache configured"
		s.markSynced()
		return result, nil
	}
	for _, pattern := range sheetCachePatterns {
		deleted, err := s.cache.Invalidate(context.Background(), pattern)
		if err != nil {
			return domain.SyncResult{}, err
		}
		result.Deleted += deleted
	}
	s.markSynced()
	return result, nil
}

func (s *dbService) WarmAIContent(forceFresh bool) (domain.InitResult, error) {
	content, err := s.LoadAIContent(forceFresh)
	if err != nil {
		return domain.InitResult{}, err
	}
	result := domain.InitResult{Ready: true}
	result.Loaded.Prompts = len(content.Prompts)
	result.Loaded.Knowledge = len(content.Knowledge)
	return result, nil
}

// LoadAIContent reads the prompt registry (domain/name → content) and the
// knowledge base (WM-AI-1). Rows missing a key or content are skipped.
func (s *dbService) LoadAIContent(forceFresh bool) (domain.AIContent, error) {
	return cachedRead(s, cacheKeyAIContent, promptsTTL, forceFresh, func() (domain.AIContent, error) {
		tabs, err := s.readRanges(promptsSheetRange, knowledgeSheetRange)
		if err != nil {
			return domain.AIContent{}, err
		}
		prompts, knowledge := tabs[0], tabs[1]
		content := domain.AIContent{Prompts: map[string]string{}, Knowledge: map[string]string{}}
		for _, row := range prompts {
			domainName, name, text := strings.TrimSpace(cellAt(row, 0)), strings.TrimSpace(cellAt(row, 1)), strings.TrimSpace(cellAt(row, 2))
			if domainName != "" && name != "" && text != "" {
				content.Prompts[domainName+"/"+name] = text
			}
		}
		for _, row := range knowledge {
			if name, text := strings.TrimSpace(cellAt(row, 0)), strings.TrimSpace(cellAt(row, 1)); name != "" && text != "" {
				content.Knowledge[name] = text
			}
		}
		return content, nil
	})
}

// Ping checks that the spreadsheet is reachable (WM-OPS-6).
func (s *dbService) Ping() error {
	_, err := s.client.SheetTitles()
	return err
}
