// Package openai sends masked pages to an OpenAI Responses API endpoint,
// ai-gateway by default (HW-AI-1, HW-READ-3).
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// providerHeader picks the provider behind ai-gateway; the official API ignores it.
const providerHeader = "X-AI-Provider"

// Config is the AI provider setting (HW-AI-1).
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	// Effort is the reasoning effort, "" to leave the provider default.
	Effort  string
	Timeout time.Duration
}

// Reader reads documents through the Responses API.
type Reader struct {
	cfg  Config
	http *http.Client
}

var _ port.Reader = (*Reader)(nil)

// New returns a reader; httpClient may be nil.
func New(cfg Config, httpClient *http.Client) *Reader {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Reader{cfg: cfg, http: httpClient}
}

type content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type message struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type textFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type request struct {
	Model        string    `json:"model"`
	Instructions string    `json:"instructions"`
	Input        []message `json:"input"`
	Text         struct {
		Format textFormat `json:"format"`
	} `json:"text"`
	Reasoning *struct {
		Effort string `json:"effort"`
	} `json:"reasoning,omitempty"`
	// Store is always false: the provider has no reason to keep health papers.
	Store bool `json:"store"`
}

type response struct {
	Model  string `json:"model"`
	Status string `json:"status"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Error json.RawMessage `json:"error"`
}

// Read sends each page's masked image and masked text in one request and
// returns the strict-schema answer.
func (r *Reader) Read(ctx context.Context, pages []port.ReadPage) (port.ReadResult, error) {
	if len(pages) == 0 {
		return port.ReadResult{}, errors.New("no pages to read")
	}
	parts := []content{{Type: "input_text", Text: fmt.Sprintf("The document has %d page(s).", len(pages))}}
	for _, p := range pages {
		parts = append(parts,
			content{Type: "input_text", Text: fmt.Sprintf("Page %d image:", p.Number)},
			content{Type: "input_image", ImageURL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(p.JPEG), Detail: "high"},
			content{Type: "input_text", Text: fmt.Sprintf("Page %d OCR text (personal lines replaced with %s):\n%s", p.Number, domain.MaskPlaceholder, p.Text)},
		)
	}
	body := request{Model: r.cfg.Model, Instructions: domain.ReadingInstructions, Input: []message{{Role: "user", Content: parts}}}
	body.Text.Format = textFormat{Type: "json_schema", Name: "document_reading", Strict: true, Schema: domain.ReadingSchema()}
	if r.cfg.Effort != "" {
		body.Reasoning = &struct {
			Effort string `json:"effort"`
		}{Effort: r.cfg.Effort}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return port.ReadResult{}, fmt.Errorf("encode AI request: %w", err)
	}
	if r.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.cfg.Timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.BaseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return port.ReadResult{}, fmt.Errorf("build AI request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(providerHeader, "openai")
	if r.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return port.ReadResult{}, fmt.Errorf("AI provider unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return port.ReadResult{}, fmt.Errorf("read AI response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return port.ReadResult{}, fmt.Errorf("AI provider returned %d: %s", resp.StatusCode, errorMessage(raw))
	}
	var decoded response
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return port.ReadResult{}, fmt.Errorf("decode AI response: %w", err)
	}
	if decoded.Status != "" && decoded.Status != "completed" {
		reason := decoded.Status
		if decoded.IncompleteDetails != nil && decoded.IncompleteDetails.Reason != "" {
			reason += ": " + decoded.IncompleteDetails.Reason
		}
		return port.ReadResult{}, fmt.Errorf("AI response was %s", reason)
	}
	var text strings.Builder
	for _, o := range decoded.Output {
		for _, c := range o.Content {
			if c.Refusal != "" {
				return port.ReadResult{}, errors.New("the AI provider refused to read the document")
			}
			if c.Type == "output_text" {
				text.WriteString(c.Text)
			}
		}
	}
	var out domain.ReadingOutput
	if err := json.Unmarshal([]byte(text.String()), &out); err != nil {
		return port.ReadResult{}, fmt.Errorf("the AI answer is not the expected JSON: %w", err)
	}
	model := decoded.Model
	if model == "" {
		model = r.cfg.Model
	}
	return port.ReadResult{Model: model, Output: out}, nil
}

// errorMessage pulls a short message out of an error body. ai-gateway may
// add a Go stack trace, which is dropped.
func errorMessage(raw []byte) string {
	var body struct {
		Error json.RawMessage `json:"error"`
	}
	msg := strings.TrimSpace(string(raw))
	if json.Unmarshal(raw, &body) == nil && len(body.Error) > 0 {
		var s string
		var obj struct {
			Message string `json:"message"`
		}
		switch {
		case json.Unmarshal(body.Error, &s) == nil:
			msg = s
		case json.Unmarshal(body.Error, &obj) == nil && obj.Message != "":
			msg = obj.Message
		}
	}
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	return msg
}
