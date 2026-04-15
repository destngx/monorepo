package domain

type ContextKey string

const (
	RequestIDKey  ContextKey = "requestID"
	LogMappingKey ContextKey = "logMapping"
	LogModelKey   ContextKey = "logModel"
)
