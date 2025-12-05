package analyze

import (
	"fmt"
	"time"

	"codemap/config"
)

// NewClient creates an LLMClient based on the configuration.
func NewClient(cfg *config.Config) (LLMClient, error) {
	timeout := time.Duration(cfg.LLM.Timeout) * time.Second
	if timeout == 0 {
		timeout = DefaultClientConfig().Timeout
	}

	clientCfg := ClientConfig{
		Model:          cfg.LLM.Model,
		EmbeddingModel: cfg.LLM.EmbeddingModel,
		Timeout:        timeout,
		MaxRetries:     cfg.LLM.MaxRetries,
		Temperature:    cfg.LLM.Temperature,
		MaxTokens:      cfg.LLM.MaxTokens,
		Debug:          cfg.Debug,
	}

	switch cfg.LLM.Provider {
	case config.ProviderOllama:
		return NewOllamaClient(cfg.LLM.OllamaURL, clientCfg), nil

	case config.ProviderOpenAI:
		return NewOpenAIClient(cfg.LLM.OpenAIAPIKey, cfg.LLM.OpenAIBaseURL, clientCfg)

	case config.ProviderAnthropic:
		return NewAnthropicClient(cfg.LLM.AnthropicAPIKey, clientCfg)

	case config.ProviderGemini:
		return NewGeminiClient(cfg.LLM.GeminiAPIKey, cfg.LLM.GeminiBaseURL, clientCfg)

	case "mock":
		return NewMockClient(clientCfg), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.LLM.Provider)
	}
}

// NewEmbeddingClient creates a client suitable for embedding operations.
// If the main provider doesn't support embeddings (e.g., Anthropic),
// it falls back to Ollama.
func NewEmbeddingClient(cfg *config.Config) (LLMClient, error) {
	// Check if embedding provider is explicitly set
	if cfg.LLM.EmbeddingProvider != "" {
		// Create a modified config with the embedding provider
		embCfg := *cfg
		embCfg.LLM.Provider = config.Provider(cfg.LLM.EmbeddingProvider)
		return NewClient(&embCfg)
	}

	// If using Anthropic (which doesn't have embeddings), default to Ollama
	if cfg.LLM.Provider == config.ProviderAnthropic {
		clientCfg := ClientConfig{
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			Timeout:        DefaultClientConfig().Timeout,
			Debug:          cfg.Debug,
		}
		return NewOllamaClient(cfg.LLM.OllamaURL, clientCfg), nil
	}

	// Otherwise use the main provider
	return NewClient(cfg)
}
