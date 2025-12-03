package analyze

import (
	"context"
	"fmt"
	"time"
)

// MockClient is a mock LLM client for testing.
type MockClient struct {
	// Responses maps prompt prefixes to responses
	Responses map[string]string

	// DefaultResponse is returned when no matching response is found
	DefaultResponse string

	// EmbeddingDimension is the dimension of mock embeddings
	EmbeddingDimension int

	// SimulateLatency adds artificial latency to simulate real requests
	SimulateLatency time.Duration

	// Error can be set to simulate errors
	Error error

	// RecordedRequests stores all requests for verification
	RecordedRequests []MockRequest

	config ClientConfig
}

// MockRequest records a request for testing verification.
type MockRequest struct {
	Type     string // "complete" or "embed"
	Messages []Message
	Text     string
}

// NewMockClient creates a new mock client for testing.
func NewMockClient(cfg ClientConfig) *MockClient {
	return &MockClient{
		Responses: make(map[string]string),
		DefaultResponse: `This is a mock response for testing.

**Purpose**: The code performs its intended functionality.

**Key Logic**: The implementation follows standard patterns.

**Parameters**: Accepts typical inputs and produces expected outputs.`,
		EmbeddingDimension: 768, // Common embedding dimension
		config:             cfg,
	}
}

// WithResponse adds a response mapping for a prompt prefix.
func (c *MockClient) WithResponse(promptPrefix, response string) *MockClient {
	c.Responses[promptPrefix] = response
	return c
}

// WithError configures the mock to return an error.
func (c *MockClient) WithError(err error) *MockClient {
	c.Error = err
	return c
}

// WithLatency configures simulated latency.
func (c *MockClient) WithLatency(d time.Duration) *MockClient {
	c.SimulateLatency = d
	return c
}

// Name returns the provider name.
func (c *MockClient) Name() string {
	return "mock"
}

// Ping always succeeds for mock client.
func (c *MockClient) Ping(ctx context.Context) error {
	if c.Error != nil {
		return c.Error
	}
	return nil
}

// Models returns mock models.
func (c *MockClient) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model", "mock-embed"}, nil
}

// Complete generates a mock completion.
func (c *MockClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	// Record request
	c.RecordedRequests = append(c.RecordedRequests, MockRequest{
		Type:     "complete",
		Messages: req.Messages,
	})

	// Check for configured error
	if c.Error != nil {
		return nil, c.Error
	}

	// Simulate latency
	if c.SimulateLatency > 0 {
		select {
		case <-time.After(c.SimulateLatency):
		case <-ctx.Done():
			return nil, ErrTimeout
		}
	}

	// Find matching response
	response := c.DefaultResponse
	for prefix, resp := range c.Responses {
		for _, msg := range req.Messages {
			if len(msg.Content) >= len(prefix) && msg.Content[:len(prefix)] == prefix {
				response = resp
				break
			}
		}
	}

	// Estimate tokens
	promptTokens := 0
	for _, msg := range req.Messages {
		promptTokens += EstimateTokens(msg.Content)
	}
	completionTokens := EstimateTokens(response)

	return &CompletionResponse{
		Content:      response,
		Model:        "mock-model",
		FinishReason: "stop",
		Duration:     time.Since(start),
		Usage: TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

// Embed generates a mock embedding vector.
func (c *MockClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	start := time.Now()

	// Record request
	c.RecordedRequests = append(c.RecordedRequests, MockRequest{
		Type: "embed",
		Text: req.Text,
	})

	// Check for configured error
	if c.Error != nil {
		return nil, c.Error
	}

	// Simulate latency
	if c.SimulateLatency > 0 {
		select {
		case <-time.After(c.SimulateLatency):
		case <-ctx.Done():
			return nil, ErrTimeout
		}
	}

	// Generate deterministic mock embedding based on text content
	embedding := make([]float64, c.EmbeddingDimension)
	for i := range embedding {
		// Simple hash-based value for deterministic results
		h := 0
		for j, r := range req.Text {
			h += int(r) * (i + j + 1)
		}
		embedding[i] = float64(h%1000) / 1000.0
	}

	return &EmbeddingResponse{
		Embedding: embedding,
		Model:     "mock-embed",
		Duration:  time.Since(start),
		Usage: TokenUsage{
			PromptTokens: EstimateTokens(req.Text),
			TotalTokens:  EstimateTokens(req.Text),
		},
	}, nil
}

// GetRequests returns all recorded requests.
func (c *MockClient) GetRequests() []MockRequest {
	return c.RecordedRequests
}

// ClearRequests clears recorded requests.
func (c *MockClient) ClearRequests() {
	c.RecordedRequests = nil
}

// AssertCalled checks if a specific type of request was made.
func (c *MockClient) AssertCalled(requestType string) error {
	for _, req := range c.RecordedRequests {
		if req.Type == requestType {
			return nil
		}
	}
	return fmt.Errorf("no %s request was made", requestType)
}

// RequestCount returns the number of requests of a given type.
func (c *MockClient) RequestCount(requestType string) int {
	count := 0
	for _, req := range c.RecordedRequests {
		if req.Type == requestType {
			count++
		}
	}
	return count
}
