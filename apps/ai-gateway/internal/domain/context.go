package domain

type ContextKey string

type RequestLogMeta struct {
	Provider        string
	Mapping         string
	Model           string
	ReasoningEffort string
}

const (
	RequestIDKey      ContextKey = "requestID"
	LogMetaKey        ContextKey = "logMeta"
	MetricsPayloadKey ContextKey = "metricsPayload"
)
