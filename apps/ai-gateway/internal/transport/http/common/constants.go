// Package common contains transport primitives shared by provider-specific HTTP handlers.
package common

const (
	HeaderContentType     = "Content-Type"
	HeaderCacheControl    = "Cache-Control"
	HeaderConnection      = "Connection"
	HeaderXAccelBuffering = "X-Accel-Buffering"
	HeaderAIProvider      = "X-AI-Provider"

	ContentTypeJSON        = "application/json"
	ContentTypeEventStream = "text/event-stream"

	ValueNoCache   = "no-cache"
	ValueKeepAlive = "keep-alive"
	ValueNo        = "no"

	SSEDataPrefix = "data: "

	ErrMsgMethodNotAllowed = "method not allowed"
	ErrMsgRoutingFailed    = "routing failed: "
	ErrMsgStreamNotSupp    = "streaming not supported by response writer"
	ErrMsgInvalidBody      = "invalid request body: "
)
