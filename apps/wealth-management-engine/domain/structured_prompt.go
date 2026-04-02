package domain

type RoleMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StructuredToolUsage struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Status  string `json:"status"`
}

type StructuredSource struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Provider string `json:"provider"`
}

type StructuredAIResponse struct {
	Version           string                `json:"version"`
	ThreadID          string                `json:"threadId"`
	ResponseType      string                `json:"responseType"`
	Persona           string                `json:"persona"`
	Summary           string                `json:"summary"`
	Answer            string                `json:"answer"`
	Actions           []string              `json:"actions"`
	FollowUpQuestions []string              `json:"followUpQuestions"`
	ActiveSkills      []string              `json:"activeSkills"`
	ToolsUsed         []StructuredToolUsage `json:"toolsUsed"`
	Sources           []StructuredSource    `json:"sources"`
	Roles             []string              `json:"roles"`
}
