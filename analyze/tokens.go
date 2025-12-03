package analyze

import (
	"strings"
	"unicode"
)

// EstimateTokens provides a rough estimate of token count for text.
// This is a simple heuristic based on typical tokenization patterns.
// For accurate counts, use the provider's actual tokenizer.
//
// Typical ratios:
// - English: ~4 characters per token
// - Code: ~3-4 characters per token
// - Mixed: ~3.5 characters per token
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count words and special tokens
	words := 0
	specialTokens := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				words++
				inWord = false
			}
		} else if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if inWord {
				words++
				inWord = false
			}
			specialTokens++
		} else {
			inWord = true
		}
	}

	if inWord {
		words++
	}

	// Rough estimate: each word is ~1.3 tokens, each special char is ~1 token
	return int(float64(words)*1.3) + specialTokens
}

// EstimateTokensForMessages estimates token count for a conversation.
// Includes overhead for message formatting.
func EstimateTokensForMessages(messages []Message) int {
	total := 0
	for _, m := range messages {
		// ~4 tokens overhead per message for role, formatting
		total += 4
		total += EstimateTokens(m.Content)
	}
	return total
}

// TruncateToTokenLimit truncates text to fit within a token limit.
// Returns the truncated text and whether truncation occurred.
func TruncateToTokenLimit(text string, maxTokens int) (string, bool) {
	estimated := EstimateTokens(text)
	if estimated <= maxTokens {
		return text, false
	}

	// Rough conversion: ~4 chars per token
	targetChars := maxTokens * 4

	// Find a good break point (end of line or sentence)
	if len(text) <= targetChars {
		return text, false
	}

	truncated := text[:targetChars]

	// Try to break at newline
	if idx := strings.LastIndex(truncated, "\n"); idx > targetChars/2 {
		return truncated[:idx] + "\n... (truncated)", true
	}

	// Try to break at sentence
	if idx := strings.LastIndex(truncated, ". "); idx > targetChars/2 {
		return truncated[:idx+1] + " ... (truncated)", true
	}

	// Break at word boundary
	if idx := strings.LastIndex(truncated, " "); idx > targetChars/2 {
		return truncated[:idx] + " ... (truncated)", true
	}

	return truncated + "... (truncated)", true
}

// TokenBudget helps manage token allocation across multiple content blocks.
type TokenBudget struct {
	Total     int
	Remaining int
}

// NewTokenBudget creates a new token budget.
func NewTokenBudget(total int) *TokenBudget {
	return &TokenBudget{
		Total:     total,
		Remaining: total,
	}
}

// Allocate reserves tokens for content, returning available tokens.
// Returns 0 if budget is exhausted.
func (b *TokenBudget) Allocate(content string) int {
	needed := EstimateTokens(content)
	if needed > b.Remaining {
		return 0
	}
	b.Remaining -= needed
	return needed
}

// Reserve sets aside tokens for later use (e.g., response).
func (b *TokenBudget) Reserve(tokens int) bool {
	if tokens > b.Remaining {
		return false
	}
	b.Remaining -= tokens
	return true
}

// Used returns how many tokens have been used.
func (b *TokenBudget) Used() int {
	return b.Total - b.Remaining
}

// Available returns how many tokens are available.
func (b *TokenBudget) Available() int {
	return b.Remaining
}
