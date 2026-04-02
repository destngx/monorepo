package domain

import "io"

type SkillFilter struct {
	Whitelist []string `json:"whitelist,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
}

type SkillDefinition struct {
	Name          string         `json:"name"`
	Path          string         `json:"path,omitempty"`
	RootPath      string         `json:"rootPath,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	Description   string         `json:"description,omitempty"`
	Content       string         `json:"content,omitempty"`
	License       string         `json:"license,omitempty"`
	Compatibility string         `json:"compatibility,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Scripts       []string       `json:"scripts,omitempty"`
	References    []string       `json:"references,omitempty"`
	Assets        []string       `json:"assets,omitempty"`
}

type SkillResource struct {
	Skill   string `json:"skill"`
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Backend     string         `json:"backend"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

type ToolResult struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Content string `json:"content"`
}

type ConversationThread struct {
	ID    string             `json:"id"`
	Turns []ConversationTurn `json:"turns"`
}

type ChatRunRequest struct {
	ThreadID       string      `json:"threadId"`
	Prompt         string      `json:"prompt,omitempty"`
	SystemPrompt   string      `json:"systemPrompt,omitempty"`
	BusinessPrompt string      `json:"businessPrompt,omitempty"`
	UserPrompt     string      `json:"userPrompt,omitempty"`
	Model          string      `json:"model,omitempty"`
	SkillFilter    SkillFilter `json:"skillFilter,omitempty"`
}

type JSONRunRequest struct {
	ThreadID       string      `json:"threadId"`
	SystemPrompt   string      `json:"systemPrompt,omitempty"`
	BusinessPrompt string      `json:"businessPrompt,omitempty"`
	UserPrompt     string      `json:"userPrompt,omitempty"`
	Prompt         string      `json:"prompt,omitempty"`
	Model          string      `json:"model,omitempty"`
	SkillFilter    SkillFilter `json:"skillFilter,omitempty"`
}

type OrchestrationResponse struct {
	ThreadID       string                `json:"threadId"`
	Turns          []ConversationTurn    `json:"turns"`
	Message        string                `json:"message,omitempty"`
	JSON           *StructuredAIResponse `json:"json,omitempty"`
	ActiveSkills   []string              `json:"activeSkills,omitempty"`
	AvailableTools []string              `json:"availableTools,omitempty"`
	Stream         io.ReadCloser         `json:"-"`
}

type GraphDecision struct {
	Type          string               `json:"type"`
	Message       string               `json:"message,omitempty"`
	ToolName      string               `json:"toolName,omitempty"`
	Arguments     map[string]any       `json:"arguments,omitempty"`
	SelectionMode string               `json:"selectionMode,omitempty"`
	JSONResponse  StructuredAIResponse `json:"jsonResponse,omitempty"`
}
