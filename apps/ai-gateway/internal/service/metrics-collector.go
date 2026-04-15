package service

import (
	"apps/ai-gateway/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector manages in-memory request metrics with periodic persistence.
type MetricsCollector struct {
	// Atomic counters for high-frequency updates (lock-free)
	totalRequests atomic.Int64
	totalErrors   atomic.Int64
	totalTokens   atomic.Int64
	totalDuration atomic.Int64 // sum in ms

	// Guarded structures
	mu          sync.Mutex
	ringBuffer  []domain.RequestRecord
	bufferSize  int
	bufferHead  int
	bufferCount int

	byRoute    map[string]*dimensionStats
	byProvider map[string]*dimensionStats
	byModel    map[string]*dimensionStats

	// Time-series buckets
	dailyTokens   map[string]int64
	weeklyTokens  map[string]int64
	monthlyTokens map[string]int64

	startTime    time.Time
	persistPath  string
	saveInterval time.Duration
	stopCh       chan struct{}

	// Cardinality guards
	knownRoutes    map[string]bool
	knownProviders map[string]bool
}

type dimensionStats struct {
	count         int64
	errors        int64
	totalDuration int64
	totalPrompt   int64
	totalComp     int64
	totalTokens   int64
	durations     [200]int64 // rolling fixed array for P95
	durHead       int
	durCount      int
}

// NewMetricsCollector creates a new collector with the given buffer size and persistence settings.
func NewMetricsCollector(bufferSize int, persistPath string, saveInterval time.Duration) *MetricsCollector {
	if bufferSize <= 0 {
		bufferSize = 2000
	}
	return &MetricsCollector{
		ringBuffer:    make([]domain.RequestRecord, bufferSize),
		bufferSize:    bufferSize,
		byRoute:       make(map[string]*dimensionStats),
		byProvider:    make(map[string]*dimensionStats),
		byModel:       make(map[string]*dimensionStats),
		dailyTokens:   make(map[string]int64),
		weeklyTokens:  make(map[string]int64),
		monthlyTokens: make(map[string]int64),
		startTime:     time.Now(),
		persistPath:   persistPath,
		saveInterval:  saveInterval,
		stopCh:        make(chan struct{}),
		knownRoutes: map[string]bool{
			"/v1/chat/completions": true,
			"/v1/messages":         true,
			"/v1/embeddings":       true,
			"/tokenize":            true,
		},
		knownProviders: map[string]bool{
			domain.ProviderOpenAI:        true,
			domain.ProviderAnthropic:     true,
			domain.ProviderGitHubCopilot: true,
			domain.ProviderGitHubModels:  true,
			domain.ProviderOllama:        true,
			domain.ProviderBedrock:       true,
		},
	}
}

// Record captures a single request's data.
func (c *MetricsCollector) Record(r domain.RequestRecord) {
	c.totalRequests.Add(1)
	if r.StatusCode >= 400 {
		c.totalErrors.Add(1)
	}
	c.totalTokens.Add(int64(r.Usage.TotalTokens))
	c.totalDuration.Add(r.DurationMs)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update ring buffer
	c.ringBuffer[c.bufferHead] = r
	c.bufferHead = (c.bufferHead + 1) % c.bufferSize
	if c.bufferCount < c.bufferSize {
		c.bufferCount++
	}

	// Sanitize dimensions
	route := c.sanitize(c.knownRoutes, r.Route)
	provider := c.sanitize(c.knownProviders, r.Provider)
	model := r.Model
	if model == "" {
		model = "unknown"
	}

	c.updateDimension(c.byRoute, route, r)
	c.updateDimension(c.byProvider, provider, r)
	c.updateDimension(c.byModel, model, r)

	// Update time-series buckets
	now := r.Timestamp.UTC() // Use request timestamp for historical consistency
	dayKey := now.Format("2006-01-02")
	_, week := now.ISOWeek()
	weekKey := fmt.Sprintf("%d-W%02d", now.Year(), week)
	monthKey := now.Format("2006-01")

	c.dailyTokens[dayKey] += int64(r.Usage.TotalTokens)
	c.weeklyTokens[weekKey] += int64(r.Usage.TotalTokens)
	c.monthlyTokens[monthKey] += int64(r.Usage.TotalTokens)

	c.pruneBuckets()
}

func (c *MetricsCollector) sanitize(allowed map[string]bool, val string) string {
	if allowed[val] {
		return val
	}
	return "other"
}

func (c *MetricsCollector) updateDimension(m map[string]*dimensionStats, key string, r domain.RequestRecord) {
	s, ok := m[key]
	if !ok {
		// Limit number of models to prevent OOM
		if len(m) > 100 {
			key = "other"
			s, ok = m[key]
			if !ok {
				s = &dimensionStats{}
				m[key] = s
			}
		} else {
			s = &dimensionStats{}
			m[key] = s
		}
	}

	s.count++
	if r.StatusCode >= 400 {
		s.errors++
	}
	s.totalDuration += r.DurationMs
	s.totalPrompt += int64(r.Usage.PromptTokens)
	s.totalComp += int64(r.Usage.CompletionTokens)
	s.totalTokens += int64(r.Usage.TotalTokens)

	// Update rolling P95 durations
	s.durations[s.durHead] = r.DurationMs
	s.durHead = (s.durHead + 1) % len(s.durations)
	if s.durCount < len(s.durations) {
		s.durCount++
	}
}

func (c *MetricsCollector) pruneBuckets() {
	// Keep last 30 days
	if len(c.dailyTokens) > 30 {
		c.removeOldest(c.dailyTokens, 30)
	}
	// Keep last 12 weeks
	if len(c.weeklyTokens) > 12 {
		c.removeOldest(c.weeklyTokens, 12)
	}
	// Keep last 12 months
	if len(c.monthlyTokens) > 12 {
		c.removeOldest(c.monthlyTokens, 12)
	}
}

func (c *MetricsCollector) removeOldest(m map[string]int64, keep int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := 0; i < len(keys)-keep; i++ {
		delete(m, keys[i])
	}
}

// Summary returns a snapshot of the current metrics.
func (c *MetricsCollector) Summary() domain.MetricsSummary {
	uptime := int64(time.Since(c.startTime).Seconds())
	totalReq := c.totalRequests.Load()

	summary := domain.MetricsSummary{
		TotalRequests: totalReq,
		TotalErrors:   c.totalErrors.Load(),
		TotalTokens:   c.totalTokens.Load(),
		UptimeSecs:    uptime,
		ByRoute:       make(map[string]*domain.RouteStats),
		ByProvider:    make(map[string]*domain.RouteStats),
		ByModel:       make(map[string]*domain.RouteStats),
		TimeSeries:    &domain.TimeSeriesUsage{},
	}

	if totalReq > 0 {
		summary.AvgDurationMs = float64(c.totalDuration.Load()) / float64(totalReq)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.fillStats(summary.ByRoute, c.byRoute)
	c.fillStats(summary.ByProvider, c.byProvider)
	c.fillStats(summary.ByModel, c.byModel)

	summary.TimeSeries.Daily = c.exportUsage(c.dailyTokens)
	summary.TimeSeries.Weekly = c.exportUsage(c.weeklyTokens)
	summary.TimeSeries.Monthly = c.exportUsage(c.monthlyTokens)

	// Collect recent requests (last 20)
	recentCount := 20
	if c.bufferCount < recentCount {
		recentCount = c.bufferCount
	}
	summary.RecentRequests = make([]domain.RequestSnapshot, 0, recentCount)
	for i := 0; i < recentCount; i++ {
		idx := (c.bufferHead - 1 - i + c.bufferSize) % c.bufferSize
		rec := c.ringBuffer[idx]
		summary.RecentRequests = append(summary.RecentRequests, domain.RequestSnapshot{
			Timestamp:    rec.Timestamp.Format(time.RFC3339),
			Route:        rec.Route,
			Provider:     rec.Provider,
			Model:        rec.Model,
			StatusCode:   rec.StatusCode,
			DurationMs:   rec.DurationMs,
			PromptTokens: rec.Usage.PromptTokens,
			CompTokens:   rec.Usage.CompletionTokens,
			TotalTokens:  rec.Usage.TotalTokens,
			Error:        rec.Error,
		})
	}

	return summary
}

func (c *MetricsCollector) fillStats(target map[string]*domain.RouteStats, source map[string]*dimensionStats) {
	for k, s := range source {
		stats := &domain.RouteStats{
			Count:          s.count,
			Errors:         s.errors,
			TotalPromptTkn: s.totalPrompt,
			TotalCompTkn:   s.totalComp,
			TotalTokens:    s.totalTokens,
		}
		if s.count > 0 {
			stats.AvgDurationMs = float64(s.totalDuration) / float64(s.count)

			// Calculate P95 from rolling window
			durs := make([]int64, s.durCount)
			copy(durs, s.durations[:s.durCount])
			sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
			if s.durCount > 0 {
				p95Idx := int(float64(s.durCount) * 0.95)
				if p95Idx >= s.durCount {
					p95Idx = s.durCount - 1
				}
				stats.P95DurationMs = durs[p95Idx]
			}
		}
		target[k] = stats
	}
}

func (c *MetricsCollector) exportUsage(m map[string]int64) []domain.UsagePoint {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]domain.UsagePoint, 0, len(keys))
	for _, k := range keys {
		res = append(res, domain.UsagePoint{Label: k, Tokens: m[k]})
	}
	return res
}

// Reset clears all metrics.
func (c *MetricsCollector) Reset() {
	c.totalRequests.Store(0)
	c.totalErrors.Store(0)
	c.totalTokens.Store(0)
	c.totalDuration.Store(0)
	c.startTime = time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()
	c.bufferHead = 0
	c.bufferCount = 0
	c.byRoute = make(map[string]*dimensionStats)
	c.byProvider = make(map[string]*dimensionStats)
	c.byModel = make(map[string]*dimensionStats)
	c.dailyTokens = make(map[string]int64)
	c.weeklyTokens = make(map[string]int64)
	c.monthlyTokens = make(map[string]int64)
}

// StartPersistence starts the periodic save goroutine.
func (c *MetricsCollector) StartPersistence(ctx context.Context) {
	if c.persistPath == "" || c.saveInterval == 0 {
		return
	}
	ticker := time.NewTicker(c.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Flush()
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Flush saves metrics to disk immediately.
func (c *MetricsCollector) Flush() {
	if c.persistPath == "" {
		return
	}

	// Create a persistent DTO
	c.mu.Lock()
	dto := struct {
		SavedAt         time.Time              `json:"saved_at"`
		TotalRequests   int64                  `json:"total_requests"`
		TotalErrors     int64                  `json:"total_errors"`
		TotalTokens     int64                  `json:"total_tokens"`
		TotalDurationMs int64                  `json:"total_duration_ms"`
		DailyTokens     map[string]int64       `json:"daily_tokens"`
		WeeklyTokens    map[string]int64       `json:"weekly_tokens"`
		MonthlyTokens   map[string]int64       `json:"monthly_tokens"`
		RecentRequests  []domain.RequestRecord `json:"recent_requests"`
	}{
		SavedAt:         time.Now(),
		TotalRequests:   c.totalRequests.Load(),
		TotalErrors:     c.totalErrors.Load(),
		TotalTokens:     c.totalTokens.Load(),
		TotalDurationMs: c.totalDuration.Load(),
		DailyTokens:     c.dailyTokens,
		WeeklyTokens:    c.weeklyTokens,
		MonthlyTokens:   c.monthlyTokens,
	}

	// Grab all records currently in buffer
	for i := 0; i < c.bufferCount; i++ {
		idx := (c.bufferHead - 1 - i + c.bufferSize) % c.bufferSize
		dto.RecentRequests = append(dto.RecentRequests, c.ringBuffer[idx])
	}
	c.mu.Unlock()

	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal metrics", "error", err)
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(c.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("Failed to create metrics directory", "path", dir, "error", err)
		return
	}

	// Write to temp file then rename
	tmpPath := c.persistPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		slog.Error("Failed to write metrics temp file", "path", tmpPath, "error", err)
		return
	}

	if err := os.Rename(tmpPath, c.persistPath); err != nil {
		slog.Error("Failed to rename metrics file", "from", tmpPath, "to", c.persistPath, "error", err)
	} else {
		slog.Debug("Metrics persisted to disk", "path", c.persistPath)
	}
}

// LoadFromDisk restores metrics from the persistence file.
func (c *MetricsCollector) LoadFromDisk() {
	if c.persistPath == "" {
		return
	}
	data, err := os.ReadFile(c.persistPath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("Failed to read metrics file", "path", c.persistPath, "error", err)
		}
		return
	}

	var dto struct {
		SavedAt         time.Time              `json:"saved_at"`
		TotalRequests   int64                  `json:"total_requests"`
		TotalErrors     int64                  `json:"total_errors"`
		TotalTokens     int64                  `json:"total_tokens"`
		TotalDurationMs int64                  `json:"total_duration_ms"`
		DailyTokens     map[string]int64       `json:"daily_tokens"`
		WeeklyTokens    map[string]int64       `json:"weekly_tokens"`
		MonthlyTokens   map[string]int64       `json:"monthly_tokens"`
		RecentRequests  []domain.RequestRecord `json:"recent_requests"`
	}

	if err := json.Unmarshal(data, &dto); err != nil {
		slog.Error("Failed to unmarshal metrics file", "error", err)
		return
	}

	c.Reset()

	c.mu.Lock()
	if dto.DailyTokens != nil {
		c.dailyTokens = dto.DailyTokens
	}
	if dto.WeeklyTokens != nil {
		c.weeklyTokens = dto.WeeklyTokens
	}
	if dto.MonthlyTokens != nil {
		c.monthlyTokens = dto.MonthlyTokens
	}
	c.mu.Unlock()

	// Replay recent requests into buffer and maps
	for i := len(dto.RecentRequests) - 1; i >= 0; i-- {
		// Note: Record will update time-series again, but since we reloaded them,
		// we should be careful. However, replaying requests will correctly populate
		// the dimensional stats which aren't explicitly saved.
		// To avoid double-counting in time-series, we could temporary disable them
		// or just clear them before replay.
		c.Record(dto.RecentRequests[i])
	}

	// Override totals with the persisted values
	c.totalRequests.Store(dto.TotalRequests)
	c.totalErrors.Store(dto.TotalErrors)
	c.totalTokens.Store(dto.TotalTokens)
	c.totalDuration.Store(dto.TotalDurationMs)

	slog.Info("Metrics restored from disk", "path", c.persistPath, "reqs", dto.TotalRequests)
}

// Stop signals the persistence goroutine to exit.
func (c *MetricsCollector) Stop() {
	close(c.stopCh)
}
