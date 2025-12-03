package analyze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const anthropicAPIVersion = "2023-06-01"

// AnthropicClient implements LLMClient for Anthropic's Claude API.
type AnthropicClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewAnthropicClient creates a new Anthropic Claude client.
func NewAnthropicClient(apiKey string, cfg ClientConfig) (*AnthropicClient, error) {
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}
	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &AnthropicClient{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}, nil
}

// Name returns the provider name.
func (c *AnthropicClient) Name() string {
	return "anthropic"
}

// Ping checks if the Anthropic API is reachable.
func (c *AnthropicClient) Ping(ctx context.Context) error {
	// Anthropic doesn't have a dedicated health endpoint, so we make a minimal request
	req := &CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
		MaxTokens: 1,
	}
	_, err := c.Complete(ctx, req)
	return err
}

// Models returns a list of available models.
// Anthropic doesn't have a models listing API, so we return known models.
func (c *AnthropicClient) Models(ctx context.Context) ([]string, error) {
	return []string{
		"claude-sonnet-4-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

// anthropicRequest is the request format for Anthropic's messages API.
type anthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	Messages      []anthropicMessage `json:"messages"`
	System        string             `json:"system,omitempty"`
	Temperature   float64            `json:"temperature,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response format from Anthropic's messages API.
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// anthropicErrorResponse is the error format from Anthropic.
type anthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete generates a completion using Anthropic's messages API.
func (c *AnthropicClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = c.config.Model
	}

	temp := req.Temperature
	if temp == 0 {
		temp = c.config.Temperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.MaxTokens
	}

	// Convert messages - Anthropic uses a different format
	var systemPrompt string
	messages := make([]anthropicMessage, 0, len(req.Messages))

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	anthropicReq := anthropicRequest{
		Model:         model,
		MaxTokens:     maxTokens,
		Messages:      messages,
		System:        systemPrompt,
		Temperature:   temp,
		StopSequences: req.Stop,
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	// Retry logic
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		resp, lastErr = c.httpClient.Do(httpReq)
		if lastErr == nil {
			// Check for rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				lastErr = ErrRateLimited
				continue
			}
			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		if resp != nil {
			resp.Body.Close()
		}

		if ctx.Err() != nil {
			return nil, ErrTimeout
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errResp anthropicErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("anthropic error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from content blocks
	var content string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content:      content,
		Model:        anthropicResp.Model,
		FinishReason: anthropicResp.StopReason,
		Duration:     time.Since(start),
		Usage: TokenUsage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

// Embed generates an embedding vector.
// Note: Anthropic doesn't provide an embeddings API, so this returns an error.
// Use a different provider (Ollama, OpenAI) for embeddings.
func (c *AnthropicClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("anthropic does not provide an embeddings API; use ollama or openai for embeddings")
}
