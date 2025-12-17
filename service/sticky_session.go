package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"

	"github.com/QuantumNous/new-api/dto"
)

var sessionIdRegex = regexp.MustCompile(`session_([a-zA-Z0-9_-]+)`)

// ExtractSessionId extracts session ID from request
// Priority: 1. metadata.user_id with session_xxx pattern
//
//	2. SHA256 hash of first user message content (first 16 chars)
func ExtractSessionId(request interface{}) string {
	switch r := request.(type) {
	case *dto.GeneralOpenAIRequest:
		return extractFromOpenAIRequest(r)
	case *dto.ClaudeRequest:
		return extractFromClaudeRequest(r)
	default:
		return ""
	}
}

// ExtractSessionIdFromBody extracts session ID from raw request body
func ExtractSessionIdFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	// Try to extract metadata.user_id first
	var requestMap map[string]json.RawMessage
	if err := json.Unmarshal(body, &requestMap); err != nil {
		return ""
	}

	// Check metadata field
	if metadataRaw, ok := requestMap["metadata"]; ok {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataRaw, &metadata); err == nil {
			if userId, ok := metadata["user_id"].(string); ok {
				if match := sessionIdRegex.FindStringSubmatch(userId); len(match) > 1 {
					return match[1]
				}
			}
		}
	}

	// Fallback: hash of first user message
	if messagesRaw, ok := requestMap["messages"]; ok {
		var messages []map[string]interface{}
		if err := json.Unmarshal(messagesRaw, &messages); err == nil {
			for _, msg := range messages {
				if role, ok := msg["role"].(string); ok && role == "user" {
					content := extractContentString(msg["content"])
					if content != "" {
						return hashContent(content)
					}
				}
			}
		}
	}

	return ""
}

func extractFromOpenAIRequest(r *dto.GeneralOpenAIRequest) string {
	// Try to extract from metadata.user_id
	if r.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(r.Metadata, &metadata); err == nil {
			if userId, ok := metadata["user_id"].(string); ok {
				if match := sessionIdRegex.FindStringSubmatch(userId); len(match) > 1 {
					return match[1]
				}
			}
		}
	}

	// Fallback: Hash of first user message
	if len(r.Messages) > 0 {
		for _, msg := range r.Messages {
			if msg.Role == "user" {
				content := msg.StringContent()
				if content != "" {
					return hashContent(content)
				}
			}
		}
	}

	return ""
}

func extractFromClaudeRequest(r *dto.ClaudeRequest) string {
	// Try to extract from metadata.user_id
	if r.Metadata != nil {
		var metadata dto.ClaudeMetadata
		if err := json.Unmarshal(r.Metadata, &metadata); err == nil {
			if match := sessionIdRegex.FindStringSubmatch(metadata.UserId); len(match) > 1 {
				return match[1]
			}
		}
	}

	// Fallback: Hash of first user message
	if len(r.Messages) > 0 {
		for _, msg := range r.Messages {
			if msg.Role == "user" {
				content := msg.GetStringContent()
				if content != "" {
					return hashContent(content)
				}
			}
		}
	}

	return ""
}

func extractContentString(content interface{}) string {
	if content == nil {
		return ""
	}

	// String content
	if str, ok := content.(string); ok {
		return str
	}

	// Array content (multimodal)
	if arr, ok := content.([]interface{}); ok {
		for _, item := range arr {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemMap["type"] == "text" {
					if text, ok := itemMap["text"].(string); ok {
						return text
					}
				}
			}
		}
	}

	return ""
}

func hashContent(content string) string {
	// Use first 500 characters for hashing to avoid long content
	if len(content) > 500 {
		content = content[:500]
	}
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars
}
