package domain

type ContextKey string

type RequestLogMeta struct {
	Provider        string
	Mapping         string
	Model           string
	ReasoningEffort string
	Failover        *FailoverLogMeta
}

// Failover reasons recorded on FailoverLogMeta.Reason.
const (
	// FailoverReasonPrimaryLimit: the primary returned a limit error on this
	// request and the request was retried on the fallback.
	FailoverReasonPrimaryLimit = "primary_limit"
	// FailoverReasonPrimaryCooldown: an earlier request put the primary in
	// cooldown, so this request went straight to the fallback.
	FailoverReasonPrimaryCooldown = "primary_cooldown"
	// FailoverReasonStreamStarted: the primary hit a limit after streaming
	// output, so the request could not be retried on the fallback.
	FailoverReasonStreamStarted = "stream_started"
)

// FailoverLogMeta records why and how a request was rerouted from its
// primary provider to a fallback provider.
type FailoverLogMeta struct {
	Primary       string
	Fallback      string
	Reason        string
	TriggerModel  string
	TriggerError  string
	FallbackError string
}

// ServedByFallback reports whether the fallback provider handled the request.
func (f *FailoverLogMeta) ServedByFallback() bool {
	return f != nil && f.Reason != FailoverReasonStreamStarted
}

// MetricsRecord returns the failover summary kept with request metrics.
func (f *FailoverLogMeta) MetricsRecord() *FailoverRecord {
	if f == nil {
		return nil
	}
	return &FailoverRecord{Primary: f.Primary, Fallback: f.Fallback, Reason: f.Reason}
}

// LogAttrs returns key/value pairs describing the failover for access logs.
func (f *FailoverLogMeta) LogAttrs() []any {
	if f == nil {
		return nil
	}
	attrs := []any{
		"failover", f.Reason,
		"failover_primary", f.Primary,
		"failover_fallback", f.Fallback,
		"failover_trigger_model", f.TriggerModel,
		"failover_trigger_error", f.TriggerError,
	}
	if f.FallbackError != "" {
		attrs = append(attrs, "failover_fallback_error", f.FallbackError)
	}
	return attrs
}

// ServingProvider returns the provider that actually handled the request,
// which differs from Provider when the request failed over.
func (m *RequestLogMeta) ServingProvider() string {
	if m == nil {
		return ""
	}
	if m.Failover.ServedByFallback() {
		return m.Failover.Fallback
	}
	return m.Provider
}

const (
	RequestIDKey      ContextKey = "requestID"
	LogMetaKey        ContextKey = "logMeta"
	MetricsPayloadKey ContextKey = "metricsPayload"
)
