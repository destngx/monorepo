package domain

import (
	"encoding/json"
	"fmt"
)

// ResponsesRequest preserves the full OpenAI Responses API payload while exposing
// routing fields needed by the gateway.
type ResponsesRequest struct {
	Model  string         `json:"model,omitempty"`
	Stream bool           `json:"stream,omitempty"`
	Body   map[string]any `json:"-"`
}

func (r *ResponsesRequest) UnmarshalJSON(data []byte) error {
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	r.Body = body
	if model, ok := body["model"].(string); ok {
		r.Model = model
	}
	if stream, ok := body["stream"].(bool); ok {
		r.Stream = stream
	}
	return nil
}

func (r ResponsesRequest) MarshalJSON() ([]byte, error) {
	body := r.CloneBody()
	return json.Marshal(body)
}

func (r ResponsesRequest) CloneBody() map[string]any {
	body := make(map[string]any, len(r.Body)+2)
	for key, value := range r.Body {
		body[key] = value
	}
	if r.Model != "" {
		body["model"] = r.Model
	}
	body["stream"] = r.Stream
	return body
}

func (r ResponsesRequest) WithModel(model string) ResponsesRequest {
	r.Model = model
	if r.Body == nil {
		r.Body = make(map[string]any)
	}
	r.Body["model"] = model
	return r
}

func (r ResponsesRequest) WithStream(stream bool) ResponsesRequest {
	r.Stream = stream
	if r.Body == nil {
		r.Body = make(map[string]any)
	}
	r.Body["stream"] = stream
	return r
}

type ResponsesResponse map[string]any

func UsageFromResponsesValue(value any) Usage {
	obj, ok := value.(map[string]any)
	if !ok {
		return Usage{}
	}
	usageObj, ok := obj["usage"].(map[string]any)
	if !ok {
		return Usage{}
	}
	return UsageFromResponsesUsage(usageObj)
}

func UsageFromResponsesUsage(usageObj map[string]any) Usage {
	inputTokens := numberToInt(usageObj["input_tokens"])
	outputTokens := numberToInt(usageObj["output_tokens"])
	totalTokens := numberToInt(usageObj["total_tokens"])
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	return Usage{
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		TotalTokens:      totalTokens,
	}
}

func numberToInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func UnsupportedResponsesError(provider string) error {
	return fmt.Errorf("responses API not supported for provider %s", provider)
}
