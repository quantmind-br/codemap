// Package config handles configuration loading and management for codemap.
// Configuration is loaded from:
// 1. ~/.config/codemap/config.yaml (user-level)
// 2. .codemap/config.yaml (project-level override)
// 3. Environment variables (highest priority)
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provider represents an LLM provider type.
type Provider string

const (
	ProviderOllama    Provider = "ollama"
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

// LLMConfig holds settings for LLM integration.
type LLMConfig struct {
	// Provider is the LLM provider to use (ollama, openai, anthropic)
	Provider Provider `yaml:"provider"`

	// Model is the model name/ID to use (e.g., "llama3", "gpt-4", "claude-3-sonnet")
	Model string `yaml:"model"`

	// Ollama-specific settings
	OllamaURL string `yaml:"ollama_url"` // Default: http://localhost:11434

	// OpenAI-specific settings
	OpenAIAPIKey  string `yaml:"openai_api_key"`  // Can also use OPENAI_API_KEY env var
	OpenAIBaseURL string `yaml:"openai_base_url"` // For Azure or compatible APIs

	// Anthropic-specific settings
	AnthropicAPIKey string `yaml:"anthropic_api_key"` // Can also use ANTHROPIC_API_KEY env var

	// Embedding settings
	EmbeddingModel    string `yaml:"embedding_model"`    // Model for embeddings
	EmbeddingProvider string `yaml:"embedding_provider"` // Provider for embeddings (can differ from main provider)

	// Request settings
	Timeout        int     `yaml:"timeout"`          // Request timeout in seconds (default: 60)
	MaxRetries     int     `yaml:"max_retries"`      // Max retry attempts (default: 3)
	Temperature    float64 `yaml:"temperature"`      // Temperature for generation (default: 0.1)
	MaxTokens      int     `yaml:"max_tokens"`       // Max tokens for response (default: 2048)
	RequestsPerMin int     `yaml:"requests_per_min"` // Rate limit (default: 60)
}

// CacheConfig holds settings for response caching.
type CacheConfig struct {
	// Enabled controls whether caching is active
	Enabled bool `yaml:"enabled"`

	// Dir is the cache directory (default: .codemap/cache)
	Dir string `yaml:"dir"`

	// TTLDays is the cache TTL in days (0 = no expiry)
	TTLDays int `yaml:"ttl_days"`
}

// Config is the main configuration structure.
type Config struct {
	// LLM holds LLM integration settings
	LLM LLMConfig `yaml:"llm"`

	// Cache holds caching settings
	Cache CacheConfig `yaml:"cache"`

	// Debug enables verbose logging
	Debug bool `yaml:"debug"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:       ProviderOllama,
			Model:          "llama3",
			OllamaURL:      "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			Timeout:        60,
			MaxRetries:     3,
			Temperature:    0.1,
			MaxTokens:      2048,
			RequestsPerMin: 60,
		},
		Cache: CacheConfig{
			Enabled: true,
			Dir:     ".codemap/cache",
			TTLDays: 0, // No expiry by default
		},
		Debug: false,
	}
}

// Load reads configuration from standard locations and merges with defaults.
// Priority (highest to lowest):
// 1. Environment variables
// 2. Project config (.codemap/config.yaml)
// 3. User config (~/.config/codemap/config.yaml)
// 4. Defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try user config first
	userConfigPath, err := userConfigPath()
	if err == nil {
		if data, err := os.ReadFile(userConfigPath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parsing user config %s: %w", userConfigPath, err)
			}
		}
	}

	// Try project config (overrides user config)
	projectConfigPath := filepath.Join(".codemap", "config.yaml")
	if data, err := os.ReadFile(projectConfigPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing project config %s: %w", projectConfigPath, err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// LoadFromPath reads configuration from a specific file path.
func LoadFromPath(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyEnvOverrides(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	var errs []string

	// Validate provider
	switch c.LLM.Provider {
	case ProviderOllama:
		if c.LLM.OllamaURL == "" {
			errs = append(errs, "ollama_url required for ollama provider")
		}
	case ProviderOpenAI:
		if c.LLM.OpenAIAPIKey == "" {
			errs = append(errs, "openai_api_key required for openai provider (or set OPENAI_API_KEY env var)")
		}
	case ProviderAnthropic:
		if c.LLM.AnthropicAPIKey == "" {
			errs = append(errs, "anthropic_api_key required for anthropic provider (or set ANTHROPIC_API_KEY env var)")
		}
	case "":
		// Will use default
	default:
		errs = append(errs, fmt.Sprintf("unknown provider: %s", c.LLM.Provider))
	}

	// Validate model
	if c.LLM.Model == "" {
		errs = append(errs, "model is required")
	}

	// Validate timeouts
	if c.LLM.Timeout < 0 {
		errs = append(errs, "timeout must be non-negative")
	}
	if c.LLM.MaxRetries < 0 {
		errs = append(errs, "max_retries must be non-negative")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// userConfigPath returns the path to the user configuration file.
func userConfigPath() (string, error) {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "codemap", "config.yaml"), nil
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "codemap", "config.yaml"), nil
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) {
	// Provider
	if v := os.Getenv("CODEMAP_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = Provider(strings.ToLower(v))
	}

	// Model
	if v := os.Getenv("CODEMAP_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}

	// Ollama
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		cfg.LLM.OllamaURL = v
	}
	if v := os.Getenv("CODEMAP_OLLAMA_URL"); v != "" {
		cfg.LLM.OllamaURL = v
	}

	// OpenAI
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.LLM.OpenAIAPIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.LLM.OpenAIBaseURL = v
	}

	// Anthropic
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.LLM.AnthropicAPIKey = v
	}

	// Debug
	if v := os.Getenv("CODEMAP_DEBUG"); v == "1" || strings.ToLower(v) == "true" {
		cfg.Debug = true
	}
}

// WriteDefault creates a default config file at the specified path.
func WriteDefault(path string) error {
	cfg := DefaultConfig()

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Add header comment
	content := "# Codemap Configuration\n# See https://github.com/owner/codemap for documentation\n\n" + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
