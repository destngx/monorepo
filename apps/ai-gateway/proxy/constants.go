package proxy

const (
	HeaderAIProvider      = "X-AI-Provider"
	HeaderAuthorization   = "Authorization"
	HeaderContentType     = "Content-Type"
	HeaderAllowOrigin     = "Access-Control-Allow-Origin"
	HeaderAllowMethods    = "Access-Control-Allow-Methods"
	HeaderAllowHeaders    = "Access-Control-Allow-Headers"
	HeaderEventID         = "X-Event-ID"
	HeaderCacheControl    = "Cache-Control"
	HeaderConnection      = "Connection"
	HeaderXAccelBuffering = "X-Accel-Buffering"

	ContentTypeJSON        = "application/json"
	ContentTypeEventStream = "text/event-stream"

	ProviderGitHub        = "github"
	ProviderGitHubCopilot = "github-copilot"

	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"

	TypeMessage    = "message"
	TypeText       = "text"
	TypeToolUse    = "tool_use"
	TypeToolResult = "tool_result"
	TypeTextDelta  = "text_delta"
	TypeInputJSON  = "input_json_delta"
	TypeAPIError   = "error"

	StopReasonMaxTokens = "max_tokens"
	StopReasonToolUse   = "tool_use"

	FinishReasonLength    = "length"
	FinishReasonToolCalls = "tool_calls"
	FinishReasonStop      = "stop"

	ValueNoCache         = "no-cache"
	ValueKeepAlive       = "keep-alive"
	ValueNo              = "no"
	ValueAllowAll        = "*"
	ValueMethodsStandard = "GET, POST, OPTIONS"
	ValueHeadersStandard = "Content-Type, X-AI-Provider, Authorization"

	ErrMsgMethodNotAllowed = "method not allowed"
	ErrMsgRoutingFailed    = "routing failed: "
	ErrMsgStreamNotSupp    = "streaming not supported by response writer"
	ErrMsgInvalidBody      = "invalid request body: "

	SSEDataPrefix = "data: "
	SSEDone       = "[DONE]"

	LogFormatError       = "[ID:%s] ERROR: code=%d msg=%s"
	LogFormatStreamChunk = "[ID:%s] [VERBOSE 1] Stream chunk: %s"
)
