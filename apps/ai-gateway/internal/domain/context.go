package domain

type ContextKey string

type RequestLogMeta struct {
	Mapping string
	Model   string
}

const (
	RequestIDKey ContextKey = "requestID"
	LogMetaKey   ContextKey = "logMeta"
)
