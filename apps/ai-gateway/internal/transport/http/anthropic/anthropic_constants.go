package anthropic

const (
	typeAnthroAPIError = "api_error"

	stopReasonEndTurn = "end_turn"

	eventMessageStart          = "message_start"
	eventMessageDelta          = "message_delta"
	eventMessageStop           = "message_stop"
	eventContentBlockStart     = "content_block_start"
	eventContentBlockDelta     = "content_block_delta"
	eventContentBlockStop      = "content_block_stop"
	eventError                 = "error"
	typeWebSearchResultError   = "web_search_tool_result_error"
	webSearchErrorUnavailable  = "unavailable"
	fallbackAnthropicMessageID = "msg_ai_gateway"

	logFormatAnthroRequest       = "[ID:%s] [VERBOSE 1] Received Anthropic Request: %s"
	logMsgAnthroFinished         = "[ID:%s] [VERBOSE 2] Finished decoding Anthropic request"
	logMsgAnthroEnteringSync     = "[ID:%s] [VERBOSE 2] Entering Anthropic handleSync"
	logMsgAnthroEnteringStream   = "[ID:%s] [VERBOSE 2] Entering Anthropic handleStream"
	logFormatAnthroProviderError = "[ID:%s] [VERBOSE 1] Provider returned error: %v"
	logFormatAnthroResponse      = "[ID:%s] [VERBOSE 1] Provider Response (Anthropic format): %s"
	logFormatAnthroStreamError   = "[ID:%s] STREAM ERROR: %v"
	logFormatStreamConvErr       = "[ID:%s] STREAM CONVERT ERROR: %v"

	errMsgInvalidAnthroBody = "invalid anthropic request body: "
)
