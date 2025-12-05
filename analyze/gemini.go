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

// GeminiClient implements LLMClient for Google's Gemini API.
type GeminiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewGeminiClient creates a new Gemini client.
func NewGeminiClient(apiKey, baseURL string, cfg ClientConfig) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.0-flash"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "gemini-embedding-001"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}, nil
}

// Name returns the provider name.
func (c *GeminiClient) Name() string {
	return "gemini"
}

// Ping checks if the Gemini API is reachable.
func (c *GeminiClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gemini not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrNoAPIKey
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gemini returned status %d", resp.StatusCode)
	}

	return nil
}

// Models returns a list of available Gemini models.
func (c *GeminiClient) Models(ctx context.Context) ([]string, error) {
	return []string{
		// Completion models
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-2.0-flash-exp",
		// Embedding models
		"gemini-embedding-001",
	}, nil
}

// geminiPart represents a content part in Gemini API.
type geminiPart struct {
	Text string `json:"text"`
}

// geminiContent represents a message in Gemini API.
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

// geminiGenerationConfig holds generation parameters.
type geminiGenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// geminiRequest is the request format for Gemini's generateContent API.
type geminiRequest struct {
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	Contents          []geminiContent         `json:"contents"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

// geminiResponse is the response format from Gemini's generateContent API.
type geminiResponse struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
		Index        int           `json:"index"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
}

// geminiErrorResponse is the error format from Gemini API.
type geminiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// geminiEmbedRequest is the request format for Gemini's embedContent API.
// For gemini-embedding-001, supports output_dimensionality to control vector size.
type geminiEmbedRequest struct {
	Content              geminiContent `json:"content"`
	OutputDimensionality int           `json:"output_dimensionality,omitempty"` // 768, 1536, or 3072 (default)
	TaskType             string        `json:"task_type,omitempty"`             // SEMANTIC_SIMILARITY, RETRIEVAL_DOCUMENT, etc.
}

// Embedding dimension constants for gemini-embedding-001
const (
	GeminiEmbedDim768  = 768  // Compatible with most other providers
	GeminiEmbedDim1536 = 1536 // Medium quality/size tradeoff
	GeminiEmbedDim3072 = 3072 // Full quality (default for gemini-embedding-001)
)

// geminiEmbedResponse is the response format from Gemini's embedContent API.
type geminiEmbedResponse struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
}

// Complete generates a completion using Gemini's generateContent API.
func (c *GeminiClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
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

	// Convert messages to Gemini format
	// Extract system messages into systemInstruction
	var systemInstruction *geminiContent
	contents := make([]geminiContent, 0, len(req.Messages))

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			// Gemini uses a separate systemInstruction field
			if systemInstruction == nil {
				systemInstruction = &geminiContent{
					Parts: []geminiPart{{Text: m.Content}},
				}
			} else {
				// Append to existing system instruction
				systemInstruction.Parts = append(systemInstruction.Parts, geminiPart{Text: m.Content})
			}
		case "assistant":
			// Gemini uses "model" instead of "assistant"
			contents = append(contents, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: m.Content}},
			})
		default:
			// "user" role stays the same
			contents = append(contents, geminiContent{
				Role:  m.Role,
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}

	geminiReq := geminiRequest{
		SystemInstruction: systemInstruction,
		Contents:          contents,
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     temp,
			MaxOutputTokens: maxTokens,
			StopSequences:   req.Stop,
		},
	}

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, model)

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
		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-goog-api-key", c.apiKey)

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
		case resp.StatusCode == http.StatusNotFound:
			// Model not found - don't retry
			resp.Body.Close()
			return nil, ErrModelNotFound
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
		var errResp geminiErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("gemini error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	// Extract text from parts
	var content string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	// Map finish reason
	finishReason := mapGeminiFinishReason(geminiResp.Candidates[0].FinishReason)

	return &CompletionResponse{
		Content:      content,
		Model:        geminiResp.ModelVersion,
		FinishReason: finishReason,
		Duration:     time.Since(start),
		Usage: TokenUsage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// mapGeminiFinishReason converts Gemini finish reasons to standard format.
func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION":
		return "content_filter"
	default:
		return "other"
	}
}

// Embed generates an embedding vector using Gemini's embedContent API.
func (c *GeminiClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = c.config.EmbeddingModel
	}

	geminiReq := geminiEmbedRequest{
		Content: geminiContent{
			Parts: []geminiPart{{Text: req.Text}},
		},
		// Use 768 dimensions for compatibility with other providers (OpenAI, Ollama)
		// gemini-embedding-001 defaults to 3072 but supports truncation
		OutputDimensionality: GeminiEmbedDim768,
		TaskType:             "SEMANTIC_SIMILARITY",
	}

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:embedContent", c.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrModelNotFound
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini embed error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return &EmbeddingResponse{
		Embedding: geminiResp.Embedding.Values,
		Model:     model,
		Duration:  time.Since(start),
		// Note: Gemini embed endpoint doesn't return token usage
	}, nil
}
