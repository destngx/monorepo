package shared

import (
	"apps/ai-gateway/internal/domain"
	"log"
	"reflect"
)

// CropRequest ensures that a ChatRequest stays within token limits by pruning older messages.
// It uses a character-based heuristic (conservatively 28,000 chars for an 8k token limit).
func CropRequest(req domain.ChatRequest, maxTokens int) domain.ChatRequest {
	// Conservative estimate: ~3.5 chars per token
	maxChars := maxTokens * 7 / 2
	if maxChars <= 0 {
		return req
	}

	totalChars := 0
	for _, m := range req.Messages {
		totalChars += len(m.Role) + len(m.Content)
	}

	if totalChars <= maxChars {
		return req
	}

	log.Printf("Request too large (%d chars), cropping to fits within %d chars", totalChars, maxChars)

	var newMessages []domain.Message
	currentChars := 0

	// 1. Always try to keep the System message if it exists at the start
	var systemMsg *domain.Message
	if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
		msg := req.Messages[0]
		systemMsg = &msg
		currentChars += len(msg.Role) + len(msg.Content)
	}

	// 2. Always keep the Last message
	var lastMsg domain.Message
	if len(req.Messages) > 0 {
		lastMsg = req.Messages[len(req.Messages)-1]
		currentChars += len(lastMsg.Role) + len(lastMsg.Content)
	}

	// 3. Fill the middle with the most recent messages from the end (before the last message)
	var middleMessages []domain.Message
	for i := len(req.Messages) - 2; i >= 1; i-- {
		msg := req.Messages[i]
		msgLen := len(msg.Role) + len(msg.Content)
		if currentChars+msgLen > maxChars {
			break
		}
		middleMessages = append([]domain.Message{msg}, middleMessages...)
		currentChars += msgLen
	}

	if systemMsg != nil {
		newMessages = append(newMessages, *systemMsg)
	}
	newMessages = append(newMessages, middleMessages...)
	if len(req.Messages) > 1 || (systemMsg == nil && len(req.Messages) > 0) {
		newMessages = append(newMessages, lastMsg)
	}

	req.Messages = newMessages
	return req
}

// CleanJSONSchema recursively removes unsupported fields (like additionalProperties, default)
// from a JSON schema for strict providers like Gemini.
func CleanJSONSchema(schema any) any {
	if schema == nil {
		return nil
	}

	// Handle pointer indirection
	val := reflect.ValueOf(schema)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		// We expect schemas to be map[string]any when decoded from JSON
		m, ok := schema.(map[string]any)
		if !ok {
			// If it's a map but not map[string]any, we try to reconstruct it
			newMap := make(map[string]any)
			iter := val.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				v := iter.Value().Interface()

				// Remove unsupported Gemini keys
				if k == "additionalProperties" || k == "default" {
					continue
				}

				// Check for format compatibility
				if k == "format" {
					if s, ok := v.(string); ok {
						allowed := map[string]bool{"enum": true, "date-time": true}
						if !allowed[s] {
							continue
						}
					}
				}

				newMap[k] = CleanJSONSchema(v)
			}
			return newMap
		}

		newMap := make(map[string]any)
		for k, v := range m {
			if k == "additionalProperties" || k == "default" {
				continue
			}

			if k == "format" {
				if s, ok := v.(string); ok {
					allowed := map[string]bool{"enum": true, "date-time": true}
					if !allowed[s] {
						continue
					}
				}
			}

			newMap[k] = CleanJSONSchema(v)
		}
		return newMap

	case reflect.Slice:
		newSlice := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			newSlice[i] = CleanJSONSchema(val.Index(i).Interface())
		}
		return newSlice

	default:
		return schema
	}
}
