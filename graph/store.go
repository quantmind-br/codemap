package graph

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultGraphDir is the directory name for codemap data
	DefaultGraphDir = ".codemap"
	// DefaultGraphFile is the default graph file name
	DefaultGraphFile = "graph.gob"
)

// GraphPath returns the default graph file path for a project root.
func GraphPath(rootPath string) string {
	return filepath.Join(rootPath, DefaultGraphDir, DefaultGraphFile)
}

// EnsureDir creates the .codemap directory if it doesn't exist.
func EnsureDir(rootPath string) error {
	dir := filepath.Join(rootPath, DefaultGraphDir)
	return os.MkdirAll(dir, 0755)
}

// SaveBinary writes the graph to disk using gob encoding with gzip compression.
func (g *CodeGraph) SaveBinary(path string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Update metadata
	g.LastIndexed = time.Now().Unix()
	g.NodeCount = len(g.Nodes)
	g.EdgeCount = len(g.Edges)

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// Wrap with gzip for compression
	gz := gzip.NewWriter(f)
	defer gz.Close()

	// Encode with gob
	enc := gob.NewEncoder(gz)
	if err := enc.Encode(g); err != nil {
		return fmt.Errorf("encode graph: %w", err)
	}

	return nil
}

// LoadBinary reads a graph from disk and rebuilds indexes.
func LoadBinary(path string) (*CodeGraph, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Unwrap gzip
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	// Decode with gob
	var g CodeGraph
	dec := gob.NewDecoder(gz)
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}

	// Rebuild in-memory indexes
	g.RebuildIndexes()

	return &g, nil
}

// Exists checks if a graph file exists at the given path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsStale checks if the graph needs to be rebuilt.
// Returns true if any source file has been modified after the graph was indexed.
func IsStale(g *CodeGraph, rootPath string) (bool, error) {
	if g == nil || g.LastIndexed == 0 {
		return true, nil
	}

	indexTime := time.Unix(g.LastIndexed, 0)

	// Check each file in the graph
	for path := range g.nodesByPath {
		fullPath := filepath.Join(rootPath, path)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File was deleted - graph is stale
				return true, nil
			}
			continue // Skip files we can't stat
		}
		if info.ModTime().After(indexTime) {
			return true, nil
		}
	}

	return false, nil
}

// GetModifiedFiles returns a list of files that have been modified since the graph was indexed.
// Only considers paths that look like actual source files (contain a file extension).
func GetModifiedFiles(g *CodeGraph, rootPath string) ([]string, error) {
	if g == nil || g.LastIndexed == 0 {
		return nil, nil
	}

	indexTime := time.Unix(g.LastIndexed, 0)
	var modified []string

	for path := range g.nodesByPath {
		// Skip package paths (no extension) - only check actual file paths
		ext := filepath.Ext(path)
		if ext == "" {
			continue
		}
		fullPath := filepath.Join(rootPath, path)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				modified = append(modified, path)
			}
			continue
		}
		if info.ModTime().After(indexTime) {
			modified = append(modified, path)
		}
	}

	return modified, nil
}

// GetDeletedFiles returns files that were in the graph but no longer exist.
// Only considers paths that look like actual source files (contain a file extension).
func GetDeletedFiles(g *CodeGraph, rootPath string) []string {
	if g == nil {
		return nil
	}

	var deleted []string
	for path := range g.nodesByPath {
		// Skip package paths (no extension) - only check actual file paths
		ext := filepath.Ext(path)
		if ext == "" {
			continue
		}
		fullPath := filepath.Join(rootPath, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			deleted = append(deleted, path)
		}
	}

	return deleted
}

// IsFileInGraph checks if a file path is already in the graph.
func (g *CodeGraph) IsFileInGraph(path string) bool {
	_, exists := g.nodesByPath[path]
	return exists
}

func init() {
	// Register types for gob encoding
	gob.Register(&CodeGraph{})
	gob.Register(&Node{})
	gob.Register(&Edge{})
}
