// Package analyze provides LLM integration for code analysis and summarization.
package analyze

import (
	"context"
	"errors"
	"time"
)

// Common errors returned by LLM clients.
var (
	ErrNoAPIKey       = errors.New("API key not configured")
	ErrRateLimited    = errors.New("rate limited by provider")
	ErrModelNotFound  = errors.New("model not found")
	ErrContextTooLong = errors.New("context exceeds model limit")
	ErrTimeout        = errors.New("request timed out")
)

// Message represents a chat message in a conversation.
type Message struct {
	Role    string // "system", "user", or "assistant"
	Content string
}

// CompletionRequest holds parameters for a completion request.
type CompletionRequest struct {
	// Messages is the conversation history
	Messages []Message

	// Model overrides the default model (optional)
	Model string

	// Temperature controls randomness (0.0 = deterministic, 1.0 = creative)
	Temperature float64

	// MaxTokens limits the response length
	MaxTokens int

	// Stop sequences that will halt generation
	Stop []string
}

// CompletionResponse holds the result of a completion request.
type CompletionResponse struct {
	// Content is the generated text
	Content string

	// Model is the model that was used
	Model string

	// Usage contains token counts
	Usage TokenUsage

	// FinishReason indicates why generation stopped
	FinishReason string

	// Duration is how long the request took
	Duration time.Duration
}

// EmbeddingRequest holds parameters for an embedding request.
type EmbeddingRequest struct {
	// Text is the content to embed
	Text string

	// Model overrides the default embedding model (optional)
	Model string
}

// EmbeddingResponse holds the result of an embedding request.
type EmbeddingResponse struct {
	// Embedding is the vector representation
	Embedding []float64

	// Model is the model that was used
	Model string

	// Usage contains token counts
	Usage TokenUsage

	// Duration is how long the request took
	Duration time.Duration
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMClient defines the interface for interacting with LLM providers.
type LLMClient interface {
	// Complete generates a completion for the given messages.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// Models returns a list of available models.
	Models(ctx context.Context) ([]string, error)

	// Ping checks if the provider is reachable.
	Ping(ctx context.Context) error

	// Name returns the provider name.
	Name() string
}

// ClientConfig holds common configuration for LLM clients.
type ClientConfig struct {
	// Model is the default model to use
	Model string

	// EmbeddingModel is the default model for embeddings
	EmbeddingModel string

	// Timeout is the request timeout
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// Temperature is the default temperature
	Temperature float64

	// MaxTokens is the default max tokens
	MaxTokens int

	// Debug enables verbose logging
	Debug bool
}

// DefaultClientConfig returns a configuration with sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:     60 * time.Second,
		MaxRetries:  3,
		Temperature: 0.1,
		MaxTokens:   2048,
		Debug:       false,
	}
}
