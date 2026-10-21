package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apps/health-watcher/port"
)

func TestGivenMaskedPagesWhenReadingThenTheRequestIsStrictPrivateAndCarriesImageAndText(t *testing.T) {
	var got map[string]any
	var header http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		if r.URL.Path != "/v1/responses" {
			t.Errorf("path = %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		answer := `{"document":{"kind":"lab_result","title":"t","date":"2025-03-14","visit_type":"lab","provider_name":"p","provider_kind":"hospital","confidence":90},"observations":[{"page":1,"analyte":"Glucose","value":"6.4","unit":"mmol/L","reference":"3.9 - 6.1","flag":"H","sample_date":"","confidence":95}],"findings":[]}`
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "gpt-6-luna", "status": "completed",
			"output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": answer}}}}})
	}))
	defer srv.Close()
	r := New(Config{BaseURL: srv.URL + "/v1/", APIKey: "k", Model: "gpt-6-luna", Effort: "low"}, nil)
	res, err := r.Read(context.Background(), []port.ReadPage{{Number: 1, JPEG: []byte{0xFF, 0xD8}, Text: "Glucose 6.4"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Model != "gpt-6-luna" || len(res.Output.Observations) != 1 || res.Output.Observations[0].Flag != "H" {
		t.Fatalf("result = %+v", res)
	}
	if got["store"] != false || got["model"] != "gpt-6-luna" {
		t.Fatalf("request = %v", got)
	}
	if got["reasoning"].(map[string]any)["effort"] != "low" {
		t.Fatalf("reasoning = %v", got["reasoning"])
	}
	format := got["text"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["strict"] != true {
		t.Fatalf("format = %v", format)
	}
	raw, _ := json.Marshal(got["input"])
	if !strings.Contains(string(raw), "data:image/jpeg;base64,/9g=") || !strings.Contains(string(raw), "Glucose 6.4") {
		t.Fatalf("input = %s", raw)
	}
	if header.Get("X-AI-Provider") != "openai" || header.Get("Authorization") != "Bearer k" {
		t.Fatalf("headers = %v", header)
	}
}

func TestGivenAGatewayErrorWithAStackTraceWhenReadingThenOnlyTheMessageIsKept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"model not supported","stack":"goroutine 1 [running]: ..."}`))
	}))
	defer srv.Close()
	_, err := New(Config{BaseURL: srv.URL, Model: "m"}, nil).Read(context.Background(), []port.ReadPage{{Number: 1}})
	if err == nil || !strings.Contains(err.Error(), "502: model not supported") || strings.Contains(err.Error(), "goroutine") {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenAnIncompleteOrRefusedAnswerWhenReadingThenItIsAnError(t *testing.T) {
	for name, body := range map[string]string{
		"incomplete": `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`,
		"refusal":    `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"no"}]}]}`,
		"not json":   `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		if _, err := New(Config{BaseURL: srv.URL, Model: "m"}, nil).Read(context.Background(), []port.ReadPage{{Number: 1}}); err == nil {
			t.Errorf("%s: no error", name)
		}
		srv.Close()
	}
}
