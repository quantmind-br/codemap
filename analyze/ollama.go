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

// OllamaClient implements LLMClient for Ollama local inference.
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(baseURL string, cfg ClientConfig) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "nomic-embed-text"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}
}

// Name returns the provider name.
func (c *OllamaClient) Name() string {
	return "ollama"
}

// Ping checks if Ollama is reachable.
func (c *OllamaClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

// Models returns a list of available models.
func (c *OllamaClient) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// ollamaChatRequest is the request format for Ollama's chat API.
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64  `json:"temperature,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// ollamaChatResponse is the response format from Ollama's chat API.
type ollamaChatResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done            bool   `json:"done"`
	DoneReason      string `json:"done_reason,omitempty"`
	TotalDuration   int64  `json:"total_duration,omitempty"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
}

// Complete generates a completion using Ollama's chat API.
func (c *OllamaClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
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

	// Convert messages to Ollama format
	messages := make([]ollamaMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ollamaMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	ollamaReq := ollamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
		Options: ollamaOptions{
			Temperature: temp,
			NumPredict:  maxTokens,
			Stop:        req.Stop,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Retry logic with exponential backoff
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		// Check context before each attempt
		if ctx.Err() != nil {
			return nil, ErrTimeout
		}

		// Create new request for each attempt (body reader is consumed after each request)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, lastErr = c.httpClient.Do(httpReq)
		if lastErr != nil {
			// Network error - retry
			if ctx.Err() != nil {
				return nil, ErrTimeout
			}
			continue
		}

		// Check response status
		switch {
		case resp.StatusCode == http.StatusOK:
			// Success - break out of retry loop
			lastErr = nil
		case resp.StatusCode == http.StatusTooManyRequests:
			// Rate limited - close body and retry
			resp.Body.Close()
			lastErr = ErrRateLimited
			continue
		case resp.StatusCode >= 500:
			// Server error - close body and retry
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		default:
			// Client error (4xx) - don't retry, break to handle error with body
			lastErr = nil
		}
		break
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &CompletionResponse{
		Content:      ollamaResp.Message.Content,
		Model:        ollamaResp.Model,
		FinishReason: ollamaResp.DoneReason,
		Duration:     time.Since(start),
		Usage: TokenUsage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

// ollamaEmbedRequest is the request format for Ollama's embed API.
type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// ollamaEmbedResponse is the response format from Ollama's embed API.
type ollamaEmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
}

// Embed generates an embedding vector using Ollama.
func (c *OllamaClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = c.config.EmbeddingModel
	}

	ollamaReq := ollamaEmbedRequest{
		Model: model,
		Input: req.Text,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(ollamaResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return &EmbeddingResponse{
		Embedding: ollamaResp.Embeddings[0],
		Model:     ollamaResp.Model,
		Duration:  time.Since(start),
	}, nil
}
