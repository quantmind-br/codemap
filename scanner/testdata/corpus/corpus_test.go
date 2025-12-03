package corpus_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codemap/graph"
)

// Expected describes the expected graph structure for a test corpus.
type Expected struct {
	Description string `json:"description"`
	Nodes       struct {
		Count     int      `json:"count"`
		Files     []string `json:"files"`
		Functions []string `json:"functions"`
		Types     []string `json:"types,omitempty"`
	} `json:"nodes"`
	Edges struct {
		Count int `json:"count"`
		Calls []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"calls"`
	} `json:"edges"`
}

func TestGoCorpus(t *testing.T) {
	testCorpus(t, "go")
}

func TestPythonCorpus(t *testing.T) {
	testCorpus(t, "python")
}

func TestTypeScriptCorpus(t *testing.T) {
	testCorpus(t, "typescript")
}

func testCorpus(t *testing.T, lang string) {
	corpusDir := filepath.Join(".", lang)
	graphPath := filepath.Join(corpusDir, ".codemap", "graph.gob")

	// Load expected
	expectedPath := filepath.Join(corpusDir, "expected.json")
	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read expected.json: %v", err)
	}
	var expected Expected
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatalf("Failed to parse expected.json: %v", err)
	}

	// Load graph
	g, err := graph.LoadBinary(graphPath)
	if err != nil {
		t.Fatalf("Failed to load graph: %v", err)
	}

	// Verify node count
	if g.NodeCount != expected.Nodes.Count {
		t.Errorf("Node count mismatch: got %d, want %d", g.NodeCount, expected.Nodes.Count)
	}

	// Verify edge count
	if g.EdgeCount != expected.Edges.Count {
		t.Errorf("Edge count mismatch: got %d, want %d", g.EdgeCount, expected.Edges.Count)
	}

	// Verify files exist
	for _, file := range expected.Nodes.Files {
		nodes := g.GetNodesByPath(file)
		if len(nodes) == 0 {
			t.Errorf("Expected file node not found: %s", file)
		}
	}

	// Verify functions exist
	for _, fn := range expected.Nodes.Functions {
		nodes := g.GetNodesByName(fn)
		if len(nodes) == 0 {
			t.Errorf("Expected function not found: %s", fn)
		}
	}

	// Verify types exist
	for _, typ := range expected.Nodes.Types {
		nodes := g.GetNodesByName(typ)
		if len(nodes) == 0 {
			t.Errorf("Expected type not found: %s", typ)
		}
	}

	// Verify expected call edges exist
	for _, call := range expected.Edges.Calls {
		found := false
		fromNodes := g.GetNodesByName(call.From)
		for _, fromNode := range fromNodes {
			edges := g.GetOutgoingEdges(fromNode.ID)
			for _, edge := range edges {
				if edge.Kind == graph.EdgeCalls {
					toNode := g.GetNode(edge.To)
					if toNode != nil && toNode.Name == call.To {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("Expected call edge not found: %s -> %s", call.From, call.To)
		}
	}
}
