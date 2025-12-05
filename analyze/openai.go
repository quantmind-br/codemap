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

// OpenAIClient implements LLMClient for OpenAI's API.
type OpenAIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(apiKey, baseURL string, cfg ClientConfig) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-3-small"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}, nil
}

// Name returns the provider name.
func (c *OpenAIClient) Name() string {
	return "openai"
}

// Ping checks if OpenAI is reachable.
func (c *OpenAIClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openai not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrNoAPIKey
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	return nil
}

// Models returns a list of available models.
func (c *OpenAIClient) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}

	return models, nil
}

// openaiChatRequest is the request format for OpenAI's chat API.
type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiChatResponse is the response format from OpenAI's chat API.
type openaiChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// openaiErrorResponse is the error format from OpenAI.
type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Complete generates a completion using OpenAI's chat API.
func (c *OpenAIClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
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

	// Convert messages to OpenAI format
	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	openaiReq := openaiChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stop:        req.Stop,
	}

	body, err := json.Marshal(openaiReq)
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
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errResp openaiErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("openai error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	return &CompletionResponse{
		Content:      openaiResp.Choices[0].Message.Content,
		Model:        openaiResp.Model,
		FinishReason: openaiResp.Choices[0].FinishReason,
		Duration:     time.Since(start),
		Usage: TokenUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}, nil
}

// openaiEmbedRequest is the request format for OpenAI's embeddings API.
type openaiEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// openaiEmbedResponse is the response format from OpenAI's embeddings API.
type openaiEmbedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed generates an embedding vector using OpenAI.
func (c *OpenAIClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = c.config.EmbeddingModel
	}

	openaiReq := openaiEmbedRequest{
		Model: model,
		Input: req.Text,
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embed error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp openaiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openaiResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return &EmbeddingResponse{
		Embedding: openaiResp.Data[0].Embedding,
		Model:     openaiResp.Model,
		Duration:  time.Since(start),
		Usage: TokenUsage{
			PromptTokens: openaiResp.Usage.PromptTokens,
			TotalTokens:  openaiResp.Usage.TotalTokens,
		},
	}, nil
}
