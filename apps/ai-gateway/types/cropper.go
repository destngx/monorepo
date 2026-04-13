package types

import (
	"log"
)

// CropRequest ensures that a ChatRequest stays within token limits by pruning older messages.
// It uses a character-based heuristic (conservatively 28,000 chars for an 8k token limit).
func CropRequest(req ChatRequest, maxTokens int) ChatRequest {
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

	var newMessages []Message
	currentChars := 0

	// 1. Always try to keep the System message if it exists at the start
	var systemMsg *Message
	if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
		msg := req.Messages[0]
		systemMsg = &msg
		currentChars += len(msg.Role) + len(msg.Content)
	}

	// 2. Always keep the Last message
	var lastMsg Message
	if len(req.Messages) > 0 {
		lastMsg = req.Messages[len(req.Messages)-1]
		currentChars += len(lastMsg.Role) + len(lastMsg.Content)
	}

	// 3. Fill the middle with the most recent messages from the end (before the last message)
	var middleMessages []Message
	for i := len(req.Messages) - 2; i >= 1; i-- {
		msg := req.Messages[i]
		msgLen := len(msg.Role) + len(msg.Content)
		if currentChars+msgLen > maxChars {
			break
		}
		middleMessages = append([]Message{msg}, middleMessages...)
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
