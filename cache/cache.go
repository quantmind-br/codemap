// Package cache provides response caching for LLM interactions.
// Cache entries are keyed by content hash to ensure cache invalidation
// when source code changes.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a cached response.
type Entry struct {
	// Key is the cache key (typically content hash + operation)
	Key string `json:"key"`

	// ContentHash is the SHA256 hash of the source content
	ContentHash string `json:"content_hash"`

	// Response is the cached LLM response
	Response string `json:"response"`

	// Model is the model that generated the response
	Model string `json:"model"`

	// CreatedAt is when the entry was created
	CreatedAt time.Time `json:"created_at"`

	// Usage contains token counts
	Usage *TokenUsage `json:"usage,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Stats tracks cache performance.
type Stats struct {
	Hits       int64 `json:"hits"`
	Misses     int64 `json:"misses"`
	Writes     int64 `json:"writes"`
	Evictions  int64 `json:"evictions"`
	TotalBytes int64 `json:"total_bytes"`
}

// Cache is a file-backed cache for LLM responses.
type Cache struct {
	dir     string
	ttl     time.Duration
	stats   Stats
	mu      sync.RWMutex
	enabled bool
}

// Options configures the cache.
type Options struct {
	// Dir is the cache directory (default: .codemap/cache)
	Dir string

	// TTL is the cache entry TTL (0 = no expiry)
	TTL time.Duration

	// Enabled controls whether caching is active
	Enabled bool
}

// DefaultOptions returns default cache options.
func DefaultOptions() Options {
	return Options{
		Dir:     ".codemap/cache",
		TTL:     0, // No expiry
		Enabled: true,
	}
}

// New creates a new cache.
func New(opts Options) (*Cache, error) {
	if !opts.Enabled {
		return &Cache{enabled: false}, nil
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	return &Cache{
		dir:     opts.Dir,
		ttl:     opts.TTL,
		enabled: true,
	}, nil
}

// MakeKey creates a cache key from content hash and operation.
func MakeKey(contentHash, operation, model string) string {
	// Include operation and model in key to separate different uses
	combined := fmt.Sprintf("%s:%s:%s", contentHash, operation, model)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter keys
}

// ContentHash computes a SHA256 hash of content.
func ContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Get retrieves a cached entry.
func (c *Cache) Get(key string) (*Entry, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	path := c.keyPath(key)
	data, err := os.ReadFile(path)
	c.mu.RUnlock() // Release read lock before calling recordMiss/recordHit

	if err != nil {
		c.recordMiss()
		return nil, false
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		c.recordMiss()
		return nil, false
	}

	// Check TTL
	if c.ttl > 0 && time.Since(entry.CreatedAt) > c.ttl {
		// Entry expired
		os.Remove(path)
		c.recordMiss()
		return nil, false
	}

	c.recordHit()
	return &entry, true
}

// GetByContentHash retrieves a cached entry by content hash and operation.
func (c *Cache) GetByContentHash(contentHash, operation, model string) (*Entry, bool) {
	key := MakeKey(contentHash, operation, model)
	return c.Get(key)
}

// Set stores an entry in the cache.
func (c *Cache) Set(entry *Entry) error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	path := c.keyPath(entry.Key)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	c.stats.Writes++
	c.stats.TotalBytes += int64(len(data))

	return nil
}

// SetResponse is a convenience method to cache a response.
func (c *Cache) SetResponse(contentHash, operation, model, response string, usage *TokenUsage) error {
	key := MakeKey(contentHash, operation, model)
	entry := &Entry{
		Key:         key,
		ContentHash: contentHash,
		Response:    response,
		Model:       model,
		CreatedAt:   time.Now(),
		Usage:       usage,
	}
	return c.Set(entry)
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(key string) error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.keyPath(key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	c.stats.Evictions++
	return nil
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(c.dir, entry.Name())
			os.Remove(path)
			c.stats.Evictions++
		}
	}

	c.stats.TotalBytes = 0
	return nil
}

// Stats returns cache statistics.
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// HitRate returns the cache hit rate.
func (c *Cache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total)
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	if !c.enabled {
		return 0
	}

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}
	return count
}

// Enabled returns whether caching is enabled.
func (c *Cache) Enabled() bool {
	return c.enabled
}

// keyPath returns the file path for a cache key.
func (c *Cache) keyPath(key string) string {
	return filepath.Join(c.dir, key+".json")
}

// recordHit increments the hit counter.
func (c *Cache) recordHit() {
	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()
}

// recordMiss increments the miss counter.
func (c *Cache) recordMiss() {
	c.mu.Lock()
	c.stats.Misses++
	c.mu.Unlock()
}

// Cleanup removes expired entries.
func (c *Cache) Cleanup() error {
	if !c.enabled || c.ttl == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(c.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var cached Entry
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}

		if now.Sub(cached.CreatedAt) > c.ttl {
			os.Remove(path)
			c.stats.Evictions++
		}
	}

	return nil
}
