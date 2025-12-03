package analyze

import (
	"context"
	"fmt"
	"strings"
	"time"

	"codemap/graph"
)

// EmbeddingConfig configures the embedding pipeline
type EmbeddingConfig struct {
	// BatchSize is the number of nodes to embed per batch
	BatchSize int

	// MaxRetries is the number of retries for failed embeddings
	MaxRetries int

	// RetryDelay is the initial delay between retries
	RetryDelay time.Duration

	// ProgressFn is called after each batch with (completed, total)
	ProgressFn func(completed, total int)

	// SkipExisting skips nodes that already have embeddings
	SkipExisting bool
}

// DefaultEmbeddingConfig returns sensible defaults
func DefaultEmbeddingConfig() EmbeddingConfig {
	return EmbeddingConfig{
		BatchSize:    10,
		MaxRetries:   3,
		RetryDelay:   time.Second,
		SkipExisting: true,
	}
}

// EmbeddingStats tracks embedding generation statistics
type EmbeddingStats struct {
	Total     int
	Embedded  int
	Skipped   int
	Failed    int
	Duration  time.Duration
	TokensIn  int
	TokensOut int
}

// NodeToText converts a graph node to text suitable for embedding.
// The strategy is: Signature + DocString + Path (for context)
func NodeToText(node *graph.Node) string {
	var parts []string

	// Add kind and name for context
	kindStr := node.Kind.String()
	if node.Name != "" {
		parts = append(parts, fmt.Sprintf("%s: %s", kindStr, node.Name))
	}

	// Add signature if available (most important for functions/methods)
	if node.Signature != "" {
		parts = append(parts, fmt.Sprintf("Signature: %s", node.Signature))
	}

	// Add docstring if available
	if node.DocString != "" {
		// Truncate long docstrings
		doc := node.DocString
		if len(doc) > 500 {
			doc = doc[:500] + "..."
		}
		parts = append(parts, fmt.Sprintf("Description: %s", doc))
	}

	// Add path for file context
	if node.Path != "" {
		parts = append(parts, fmt.Sprintf("Location: %s", node.Path))
	}

	return strings.Join(parts, "\n")
}

// NodeToTextWithSource converts a node to text including source code.
// This provides richer embeddings but uses more tokens.
func NodeToTextWithSource(projectRoot string, node *graph.Node) string {
	baseText := NodeToText(node)

	// Try to read source code
	source, err := ReadSymbolSource(projectRoot, node)
	if err != nil || source == nil {
		return baseText
	}

	// Truncate source if too long
	code := source.Source
	if len(code) > 1000 {
		code = code[:1000] + "\n// ... truncated"
	}

	return fmt.Sprintf("%s\n\nCode:\n```%s\n%s\n```", baseText, source.Language, code)
}

// EmbedNodes generates embeddings for a slice of nodes.
// Returns a map of NodeID -> embedding vector.
func EmbedNodes(
	ctx context.Context,
	client LLMClient,
	nodes []*graph.Node,
	config EmbeddingConfig,
) (map[graph.NodeID][]float64, *EmbeddingStats, error) {
	stats := &EmbeddingStats{
		Total: len(nodes),
	}
	start := time.Now()

	if len(nodes) == 0 {
		stats.Duration = time.Since(start)
		return nil, stats, nil
	}

	// Prepare texts for embedding
	type nodeText struct {
		node *graph.Node
		text string
	}
	nodesToEmbed := make([]nodeText, 0, len(nodes))

	for _, node := range nodes {
		text := NodeToText(node)
		if text == "" {
			stats.Skipped++
			continue
		}
		nodesToEmbed = append(nodesToEmbed, nodeText{node: node, text: text})
	}

	if len(nodesToEmbed) == 0 {
		stats.Duration = time.Since(start)
		return nil, stats, nil
	}

	results := make(map[graph.NodeID][]float64)
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	// Process in batches
	for i := 0; i < len(nodesToEmbed); i += batchSize {
		end := i + batchSize
		if end > len(nodesToEmbed) {
			end = len(nodesToEmbed)
		}
		batch := nodesToEmbed[i:end]

		// Embed each item in batch (most APIs don't support batch embedding)
		for _, item := range batch {
			// Check context cancellation
			select {
			case <-ctx.Done():
				stats.Duration = time.Since(start)
				return results, stats, ctx.Err()
			default:
			}

			// Embed with retries
			var embedding []float64
			var err error
			for attempt := 0; attempt <= config.MaxRetries; attempt++ {
				resp, embedErr := client.Embed(ctx, &EmbeddingRequest{
					Text: item.text,
				})
				if embedErr == nil {
					embedding = resp.Embedding
					stats.TokensIn += resp.Usage.PromptTokens
					break
				}
				err = embedErr

				if attempt < config.MaxRetries {
					delay := config.RetryDelay * time.Duration(1<<attempt)
					select {
					case <-ctx.Done():
						stats.Duration = time.Since(start)
						return results, stats, ctx.Err()
					case <-time.After(delay):
					}
				}
			}

			if embedding != nil {
				results[item.node.ID] = embedding
				stats.Embedded++
			} else {
				stats.Failed++
				// Log error but continue with other nodes
				if err != nil {
					fmt.Printf("Warning: failed to embed %s: %v\n", item.node.Name, err)
				}
			}
		}

		// Report progress
		if config.ProgressFn != nil {
			config.ProgressFn(stats.Embedded+stats.Failed+stats.Skipped, stats.Total)
		}
	}

	stats.Duration = time.Since(start)
	return results, stats, nil
}

// EmbedGraph generates embeddings for all function/method nodes in a graph.
// It updates the vector index in place.
func EmbedGraph(
	ctx context.Context,
	client LLMClient,
	cg *graph.CodeGraph,
	index *graph.InMemoryVectorIndex,
	config EmbeddingConfig,
) (*EmbeddingStats, error) {
	// Collect nodes to embed (functions and methods are most useful)
	var nodes []*graph.Node
	for _, node := range cg.Nodes {
		// Skip if already has embedding and config says to skip
		if config.SkipExisting && index.Has(node.ID) {
			continue
		}

		// Only embed functions, methods, and types for now
		switch node.Kind {
		case graph.KindFunction, graph.KindMethod, graph.KindType:
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return &EmbeddingStats{Total: 0, Skipped: len(cg.Nodes)}, nil
	}

	// Generate embeddings
	embeddings, stats, err := EmbedNodes(ctx, client, nodes, config)
	if err != nil {
		return stats, err
	}

	// Add to index
	for _, node := range nodes {
		if embedding, ok := embeddings[node.ID]; ok {
			text := NodeToText(node)
			if err := index.Add(node.ID, embedding, text); err != nil {
				return stats, fmt.Errorf("adding to index: %w", err)
			}
		}
	}

	return stats, nil
}

// EmbedQuery generates an embedding for a search query
func EmbedQuery(ctx context.Context, client LLMClient, query string) ([]float64, error) {
	resp, err := client.Embed(ctx, &EmbeddingRequest{
		Text: query,
	})
	if err != nil {
		return nil, fmt.Errorf("embedding query: %w", err)
	}
	return resp.Embedding, nil
}
