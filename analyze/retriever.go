package analyze

import (
	"context"
	"sort"
	"strings"

	"codemap/graph"
)

// SearchMode controls how hybrid search operates
type SearchMode int

const (
	// SearchModeHybrid combines vector and graph search with rank fusion
	SearchModeHybrid SearchMode = iota
	// SearchModeVector uses only vector/semantic search
	SearchModeVector
	// SearchModeGraph uses only graph-based name matching
	SearchModeGraph
)

// SearchConfig configures the hybrid search behavior
type SearchConfig struct {
	// Mode controls the search strategy
	Mode SearchMode

	// Limit is the maximum number of results to return
	Limit int

	// VectorWeight is the weight for vector search results (0-1)
	VectorWeight float64

	// GraphWeight is the weight for graph search results (0-1)
	GraphWeight float64

	// ExpandContext if true, includes callers/callees in results
	ExpandContext bool

	// ExpandDepth is how many levels of context to expand
	ExpandDepth int

	// FuzzyMatch if true, uses substring matching for graph search
	FuzzyMatch bool
}

// DefaultSearchConfig returns sensible search defaults
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		Mode:          SearchModeHybrid,
		Limit:         10,
		VectorWeight:  0.6, // Slightly favor semantic search
		GraphWeight:   0.4,
		ExpandContext: false,
		ExpandDepth:   1,
		FuzzyMatch:    true,
	}
}

// HybridResult represents a combined search result
type HybridResult struct {
	Node        *graph.Node
	VectorScore float64 // Similarity from vector search (0-1)
	GraphScore  float64 // Match score from graph search (0-1)
	FinalScore  float64 // Combined score after rank fusion
	MatchReason string  // Why this result matched

	// Context expansion (optional)
	Callers []*graph.Node
	Callees []*graph.Node
	Snippet string // Code snippet if available
}

// Retriever performs hybrid search over a codebase
type Retriever struct {
	graph       *graph.CodeGraph
	vectorIndex *graph.InMemoryVectorIndex
	llmClient   LLMClient
	projectRoot string
}

// NewRetriever creates a new hybrid retriever
func NewRetriever(
	cg *graph.CodeGraph,
	index *graph.InMemoryVectorIndex,
	client LLMClient,
	projectRoot string,
) *Retriever {
	return &Retriever{
		graph:       cg,
		vectorIndex: index,
		llmClient:   client,
		projectRoot: projectRoot,
	}
}

// Search performs hybrid search combining vector and graph search
func (r *Retriever) Search(ctx context.Context, query string, config SearchConfig) ([]HybridResult, error) {
	var vectorResults []graph.SearchResult
	var graphResults []graphMatch

	// Vector search
	if config.Mode == SearchModeHybrid || config.Mode == SearchModeVector {
		if r.vectorIndex != nil && r.vectorIndex.Count() > 0 && r.llmClient != nil {
			queryVec, err := EmbedQuery(ctx, r.llmClient, query)
			if err == nil && len(queryVec) > 0 {
				results, err := r.vectorIndex.Search(queryVec, config.Limit*2)
				if err == nil {
					vectorResults = results
				}
			}
		}
	}

	// Graph search (name matching)
	if config.Mode == SearchModeHybrid || config.Mode == SearchModeGraph {
		graphResults = r.graphSearch(query, config.Limit*2, config.FuzzyMatch)
	}

	// Combine results using Reciprocal Rank Fusion
	combined := r.rankFusion(vectorResults, graphResults, config)

	// Apply limit
	if len(combined) > config.Limit {
		combined = combined[:config.Limit]
	}

	// Expand context if requested
	if config.ExpandContext {
		for i := range combined {
			r.expandContext(&combined[i], config.ExpandDepth)
		}
	}

	return combined, nil
}

// graphMatch represents a graph-based search match
type graphMatch struct {
	node      *graph.Node
	score     float64
	matchType string // "exact", "prefix", "contains"
}

// graphSearch finds nodes by name matching
func (r *Retriever) graphSearch(query string, limit int, fuzzy bool) []graphMatch {
	if r.graph == nil {
		return nil
	}

	query = strings.ToLower(query)
	words := strings.Fields(query)

	var matches []graphMatch

	for _, node := range r.graph.Nodes {
		// Skip file nodes for search (usually not what users want)
		if node.Kind == graph.KindFile || node.Kind == graph.KindPackage {
			continue
		}

		score, matchType := r.matchScore(node, query, words, fuzzy)
		if score > 0 {
			matches = append(matches, graphMatch{
				node:      node,
				score:     score,
				matchType: matchType,
			})
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches
}

// matchScore calculates how well a node matches the query
func (r *Retriever) matchScore(node *graph.Node, query string, words []string, fuzzy bool) (float64, string) {
	name := strings.ToLower(node.Name)

	// Exact match (highest score)
	if name == query {
		return 1.0, "exact"
	}

	// Prefix match
	if strings.HasPrefix(name, query) {
		return 0.9, "prefix"
	}

	if fuzzy {
		// Contains match
		if strings.Contains(name, query) {
			return 0.7, "contains"
		}

		// Word matching (for multi-word queries)
		if len(words) > 1 {
			matchedWords := 0
			for _, word := range words {
				if strings.Contains(name, word) {
					matchedWords++
				}
				// Also check signature and docstring
				if node.Signature != "" && strings.Contains(strings.ToLower(node.Signature), word) {
					matchedWords++
				}
				if node.DocString != "" && strings.Contains(strings.ToLower(node.DocString), word) {
					matchedWords++
				}
			}
			if matchedWords > 0 {
				// Score based on percentage of words matched
				score := 0.5 * float64(matchedWords) / float64(len(words)*3) // *3 for name, sig, doc
				if score > 0.1 {
					return score, "words"
				}
			}
		}

		// Check path for package/module queries
		if node.Path != "" && strings.Contains(strings.ToLower(node.Path), query) {
			return 0.4, "path"
		}
	}

	return 0, ""
}

// rankFusion combines vector and graph results using Reciprocal Rank Fusion (RRF)
// RRF score = sum(1 / (k + rank)) for each list where the item appears
// k is a constant (typically 60) to prevent division by small numbers
func (r *Retriever) rankFusion(
	vectorResults []graph.SearchResult,
	graphResults []graphMatch,
	config SearchConfig,
) []HybridResult {
	const k = 60 // RRF constant

	// Map to track combined scores
	type fusedScore struct {
		node        *graph.Node
		vectorScore float64
		vectorRank  int
		graphScore  float64
		graphRank   int
		matchReason string
	}
	scores := make(map[graph.NodeID]*fusedScore)

	// Add vector results
	for rank, vr := range vectorResults {
		node := r.graph.GetNode(vr.NodeID)
		if node == nil {
			continue
		}

		if _, exists := scores[vr.NodeID]; !exists {
			scores[vr.NodeID] = &fusedScore{
				node:        node,
				vectorRank:  -1,
				graphRank:   -1,
				matchReason: "semantic match",
			}
		}
		scores[vr.NodeID].vectorScore = vr.Score
		scores[vr.NodeID].vectorRank = rank
	}

	// Add graph results
	for rank, gr := range graphResults {
		if _, exists := scores[gr.node.ID]; !exists {
			scores[gr.node.ID] = &fusedScore{
				node:       gr.node,
				vectorRank: -1,
				graphRank:  -1,
			}
		}
		scores[gr.node.ID].graphScore = gr.score
		scores[gr.node.ID].graphRank = rank
		if scores[gr.node.ID].matchReason == "" {
			scores[gr.node.ID].matchReason = gr.matchType + " name match"
		} else {
			scores[gr.node.ID].matchReason += ", " + gr.matchType + " name match"
		}
	}

	// Calculate RRF scores
	var results []HybridResult
	for _, fs := range scores {
		var rrfScore float64

		// Vector contribution (weighted)
		if fs.vectorRank >= 0 {
			rrfScore += config.VectorWeight * (1.0 / float64(k+fs.vectorRank))
		}

		// Graph contribution (weighted)
		if fs.graphRank >= 0 {
			rrfScore += config.GraphWeight * (1.0 / float64(k+fs.graphRank))
		}

		results = append(results, HybridResult{
			Node:        fs.node,
			VectorScore: fs.vectorScore,
			GraphScore:  fs.graphScore,
			FinalScore:  rrfScore,
			MatchReason: fs.matchReason,
		})
	}

	// Sort by final score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	return results
}

// expandContext adds callers and callees to a result
func (r *Retriever) expandContext(result *HybridResult, depth int) {
	if r.graph == nil || result.Node == nil || depth <= 0 {
		return
	}

	// Get callers (direct callers only)
	callers := r.graph.GetCallers(result.Node.ID)
	for _, caller := range callers {
		result.Callers = append(result.Callers, caller)
	}

	// Get callees (direct callees only)
	callees := r.graph.GetCallees(result.Node.ID)
	for _, callee := range callees {
		result.Callees = append(result.Callees, callee)
	}

	// Add code snippet if available
	if r.projectRoot != "" {
		source, err := ReadSymbolSource(r.projectRoot, result.Node)
		if err == nil && source != nil {
			// Truncate for readability
			snippet := source.Source
			if len(snippet) > 500 {
				snippet = snippet[:500] + "\n// ..."
			}
			result.Snippet = snippet
		}
	}
}

// SemanticSearch is a convenience function for pure semantic search
func (r *Retriever) SemanticSearch(ctx context.Context, query string, limit int) ([]HybridResult, error) {
	config := DefaultSearchConfig()
	config.Mode = SearchModeVector
	config.Limit = limit
	return r.Search(ctx, query, config)
}

// NameSearch is a convenience function for name-based search
func (r *Retriever) NameSearch(query string, limit int) ([]HybridResult, error) {
	config := DefaultSearchConfig()
	config.Mode = SearchModeGraph
	config.Limit = limit
	return r.Search(context.Background(), query, config)
}
