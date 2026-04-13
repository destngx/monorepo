package providers

const (
	HeaderAuthorization   = "Authorization"
	HeaderAccept          = "Accept"
	HeaderUserAgent       = "User-Agent"
	HeaderContentType     = "Content-Type"
	HeaderEditorVersion   = "Editor-Version"
	HeaderEditorPluginVer = "Editor-Plugin-Version"

	ContentTypeJSON = "application/json"

	TokenPrefixBearer = "Bearer "
	TokenPrefixToken  = "token "

	PathChatCompletions = "/chat/completions"
	PathModels          = "/models"
	PathEmbeddings      = "/embeddings"

	SSEDataPrefix = "data: "
	SSEDone       = "[DONE]"
)
