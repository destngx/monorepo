package domain

type ToolConversation struct {
	Turns []ConversationTurn `json:"turns"`
}

type ConversationTurn struct {
	Role         string `json:"role"`
	Type         string `json:"type"`
	Content      string `json:"content,omitempty"`
	ToolCallJSON string `json:"toolCallJson,omitempty"`
}
