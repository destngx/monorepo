package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
)

func sendAnthroEvent(w io.Writer, eventType string, data any) {
	if m, ok := data.(map[string]any); ok {
		m["type"] = eventType
	}
	b, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(b))
}
