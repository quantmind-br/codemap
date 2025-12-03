package graph

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// DefaultVectorFile is the default name for vector storage
const DefaultVectorFile = "vectors.gob"

// VectorEntry holds an embedding vector for a node
type VectorEntry struct {
	NodeID    NodeID
	Vector    []float64
	Text      string // The text that was embedded (for debugging/inspection)
	Dimension int    // Vector dimension for validation
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	NodeID     NodeID
	Score      float64 // Cosine similarity score (0-1, higher is better)
	Node       *Node   // Optional: populated if graph is available
	SourceText string  // The text that was embedded
}

// VectorIndex defines the interface for vector storage and search
type VectorIndex interface {
	// Add stores a vector for a node
	Add(nodeID NodeID, vector []float64, text string) error

	// Remove deletes a vector for a node
	Remove(nodeID NodeID) error

	// Search finds the top-k most similar vectors
	Search(query []float64, k int) ([]SearchResult, error)

	// Has checks if a node has a stored vector
	Has(nodeID NodeID) bool

	// Count returns the number of stored vectors
	Count() int

	// Dimension returns the expected vector dimension
	Dimension() int

	// Save persists the index to disk
	Save(path string) error

	// Clear removes all vectors
	Clear()
}

// InMemoryVectorIndex implements VectorIndex with in-memory storage
type InMemoryVectorIndex struct {
	mu        sync.RWMutex
	vectors   map[NodeID]*VectorEntry
	dimension int
}

// NewVectorIndex creates a new in-memory vector index
// dimension specifies the expected embedding dimension (e.g., 768 for many models)
func NewVectorIndex(dimension int) *InMemoryVectorIndex {
	return &InMemoryVectorIndex{
		vectors:   make(map[NodeID]*VectorEntry),
		dimension: dimension,
	}
}

// Add stores a vector for a node
func (idx *InMemoryVectorIndex) Add(nodeID NodeID, vector []float64, text string) error {
	if len(vector) == 0 {
		return fmt.Errorf("empty vector for node %s", nodeID)
	}

	// Set dimension from first vector if not set
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.dimension == 0 {
		idx.dimension = len(vector)
	} else if len(vector) != idx.dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", idx.dimension, len(vector))
	}

	idx.vectors[nodeID] = &VectorEntry{
		NodeID:    nodeID,
		Vector:    vector,
		Text:      text,
		Dimension: len(vector),
	}

	return nil
}

// Remove deletes a vector for a node
func (idx *InMemoryVectorIndex) Remove(nodeID NodeID) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.vectors[nodeID]; !exists {
		return fmt.Errorf("vector not found for node %s", nodeID)
	}

	delete(idx.vectors, nodeID)
	return nil
}

// Search finds the top-k most similar vectors using cosine similarity
func (idx *InMemoryVectorIndex) Search(query []float64, k int) ([]SearchResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(query) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	if idx.dimension > 0 && len(query) != idx.dimension {
		return nil, fmt.Errorf("query dimension mismatch: expected %d, got %d", idx.dimension, len(query))
	}

	if len(idx.vectors) == 0 {
		return nil, nil
	}

	// Calculate cosine similarity for all vectors
	type scored struct {
		nodeID NodeID
		score  float64
		text   string
	}

	results := make([]scored, 0, len(idx.vectors))
	queryNorm := vectorNorm(query)

	for nodeID, entry := range idx.vectors {
		similarity := cosineSimilarity(query, entry.Vector, queryNorm)
		results = append(results, scored{
			nodeID: nodeID,
			score:  similarity,
			text:   entry.Text,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Take top-k
	if k > len(results) {
		k = len(results)
	}

	searchResults := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		searchResults[i] = SearchResult{
			NodeID:     results[i].nodeID,
			Score:      results[i].score,
			SourceText: results[i].text,
		}
	}

	return searchResults, nil
}

// Has checks if a node has a stored vector
func (idx *InMemoryVectorIndex) Has(nodeID NodeID) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, exists := idx.vectors[nodeID]
	return exists
}

// Count returns the number of stored vectors
func (idx *InMemoryVectorIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.vectors)
}

// Dimension returns the expected vector dimension
func (idx *InMemoryVectorIndex) Dimension() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.dimension
}

// Clear removes all vectors
func (idx *InMemoryVectorIndex) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.vectors = make(map[NodeID]*VectorEntry)
}

// Save persists the index to disk using gob with gzip compression
func (idx *InMemoryVectorIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	// Use gzip compression
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	encoder := gob.NewEncoder(gzWriter)

	// Encode dimension first
	if err := encoder.Encode(idx.dimension); err != nil {
		return fmt.Errorf("encoding dimension: %w", err)
	}

	// Encode vector count
	count := len(idx.vectors)
	if err := encoder.Encode(count); err != nil {
		return fmt.Errorf("encoding count: %w", err)
	}

	// Encode each vector entry
	for _, entry := range idx.vectors {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("encoding vector entry: %w", err)
		}
	}

	return nil
}

// LoadVectorIndex loads a vector index from disk
func LoadVectorIndex(path string) (*InMemoryVectorIndex, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	// Use gzip decompression
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzReader.Close()

	decoder := gob.NewDecoder(gzReader)

	// Decode dimension
	var dimension int
	if err := decoder.Decode(&dimension); err != nil {
		return nil, fmt.Errorf("decoding dimension: %w", err)
	}

	// Decode vector count
	var count int
	if err := decoder.Decode(&count); err != nil {
		return nil, fmt.Errorf("decoding count: %w", err)
	}

	// Create index
	idx := &InMemoryVectorIndex{
		vectors:   make(map[NodeID]*VectorEntry, count),
		dimension: dimension,
	}

	// Decode each vector entry
	for i := 0; i < count; i++ {
		var entry VectorEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, fmt.Errorf("decoding vector entry %d: %w", i, err)
		}
		idx.vectors[entry.NodeID] = &entry
	}

	return idx, nil
}

// VectorIndexExists checks if a vector index file exists
func VectorIndexExists(root string) bool {
	path := filepath.Join(root, DefaultGraphDir, DefaultVectorFile)
	_, err := os.Stat(path)
	return err == nil
}

// VectorIndexPath returns the default path for the vector index
func VectorIndexPath(root string) string {
	return filepath.Join(root, DefaultGraphDir, DefaultVectorFile)
}

// cosineSimilarity calculates cosine similarity between two vectors
// queryNorm is pre-computed for efficiency
func cosineSimilarity(a, b []float64, queryNorm float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var bNormSq float64

	for i := range a {
		dotProduct += a[i] * b[i]
		bNormSq += b[i] * b[i]
	}

	bNorm := math.Sqrt(bNormSq)

	if queryNorm == 0 || bNorm == 0 {
		return 0
	}

	return dotProduct / (queryNorm * bNorm)
}

// vectorNorm calculates the L2 norm of a vector
func vectorNorm(v []float64) float64 {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}
