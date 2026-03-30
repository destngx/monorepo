package domain

type RoleMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StructuredAIResponse struct {
	Persona string   `json:"persona"`
	Summary string   `json:"summary"`
	Actions []string `json:"actions"`
	Roles   []string `json:"roles"`
}
