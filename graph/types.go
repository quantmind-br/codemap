// Package graph provides the knowledge graph data structures for codemap.
// It represents code entities (functions, types, files) and their relationships
// (calls, imports, contains) in a queryable graph structure.
package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// NodeKind represents the type of a code entity in the graph.
type NodeKind int

const (
	KindFile NodeKind = iota
	KindPackage
	KindFunction
	KindMethod
	KindType
	KindVariable
	KindConstant
)

func (k NodeKind) String() string {
	switch k {
	case KindFile:
		return "file"
	case KindPackage:
		return "package"
	case KindFunction:
		return "function"
	case KindMethod:
		return "method"
	case KindType:
		return "type"
	case KindVariable:
		return "variable"
	case KindConstant:
		return "constant"
	default:
		return "unknown"
	}
}

// EdgeKind represents the type of relationship between nodes.
type EdgeKind int

const (
	EdgeImports EdgeKind = iota
	EdgeCalls
	EdgeDefines
	EdgeContains
	EdgeReferences
	EdgeImplements
	EdgeExtends
)

func (e EdgeKind) String() string {
	switch e {
	case EdgeImports:
		return "imports"
	case EdgeCalls:
		return "calls"
	case EdgeDefines:
		return "defines"
	case EdgeContains:
		return "contains"
	case EdgeReferences:
		return "references"
	case EdgeImplements:
		return "implements"
	case EdgeExtends:
		return "extends"
	default:
		return "unknown"
	}
}

// NodeID is a unique identifier for a node in the graph.
// Generated deterministically from path + symbol name.
type NodeID string

// GenerateNodeID creates a deterministic NodeID from path and symbol.
func GenerateNodeID(path, symbol string) NodeID {
	data := fmt.Sprintf("%s:%s", path, symbol)
	hash := sha256.Sum256([]byte(data))
	return NodeID(hex.EncodeToString(hash[:16])) // Use first 16 bytes (32 hex chars)
}

// Node represents a code entity in the knowledge graph.
type Node struct {
	ID        NodeID   `json:"id"`
	Kind      NodeKind `json:"kind"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`                // File path relative to project root
	Line      int      `json:"line,omitempty"`      // Line number (1-indexed)
	EndLine   int      `json:"end_line,omitempty"`  // End line number
	Signature string   `json:"signature,omitempty"` // Function/method signature
	DocString string   `json:"doc,omitempty"`       // Documentation comment
	Exported  bool     `json:"exported,omitempty"`  // Is publicly visible
	Package   string   `json:"package,omitempty"`   // Package/module name
}

// Edge represents a relationship between two nodes.
type Edge struct {
	From     NodeID   `json:"from"`
	To       NodeID   `json:"to"`
	Kind     EdgeKind `json:"kind"`
	Line     int      `json:"line,omitempty"`     // Line where the reference occurs
	Weight   float64  `json:"weight,omitempty"`   // Relationship strength (0-1)
	CallSite string   `json:"callsite,omitempty"` // For calls: the call expression text
}

// CodeGraph is the main knowledge graph structure with indexed lookups.
type CodeGraph struct {
	// Core storage
	Nodes map[NodeID]*Node `json:"nodes"`
	Edges []*Edge          `json:"edges"`

	// Indexes for fast lookup (rebuilt on load)
	nodesByPath map[string][]*Node // path -> nodes in that file
	nodesByName map[string][]*Node // name -> nodes with that name
	edgesByFrom map[NodeID][]*Edge // from -> outgoing edges
	edgesByTo   map[NodeID][]*Edge // to -> incoming edges

	// Metadata
	RootPath    string `json:"root"`
	Version     int    `json:"version"`
	NodeCount   int    `json:"node_count"`
	EdgeCount   int    `json:"edge_count"`
	LastIndexed int64  `json:"last_indexed"` // Unix timestamp
}

// NewCodeGraph creates an empty CodeGraph with initialized maps.
func NewCodeGraph(rootPath string) *CodeGraph {
	return &CodeGraph{
		Nodes:       make(map[NodeID]*Node),
		Edges:       make([]*Edge, 0),
		nodesByPath: make(map[string][]*Node),
		nodesByName: make(map[string][]*Node),
		edgesByFrom: make(map[NodeID][]*Edge),
		edgesByTo:   make(map[NodeID][]*Edge),
		RootPath:    rootPath,
		Version:     1,
	}
}

// AddNode adds a node to the graph and updates indexes.
func (g *CodeGraph) AddNode(n *Node) {
	if _, exists := g.Nodes[n.ID]; exists {
		return // Already exists
	}
	g.Nodes[n.ID] = n
	g.NodeCount++

	// Update indexes
	g.nodesByPath[n.Path] = append(g.nodesByPath[n.Path], n)
	g.nodesByName[n.Name] = append(g.nodesByName[n.Name], n)
}

// AddEdge adds an edge to the graph and updates indexes.
func (g *CodeGraph) AddEdge(e *Edge) {
	g.Edges = append(g.Edges, e)
	g.EdgeCount++

	// Update indexes
	g.edgesByFrom[e.From] = append(g.edgesByFrom[e.From], e)
	g.edgesByTo[e.To] = append(g.edgesByTo[e.To], e)
}

// GetNode retrieves a node by ID.
func (g *CodeGraph) GetNode(id NodeID) *Node {
	return g.Nodes[id]
}

// GetNodesByPath returns all nodes in a given file path.
func (g *CodeGraph) GetNodesByPath(path string) []*Node {
	return g.nodesByPath[path]
}

// GetNodesByName returns all nodes with a given name.
func (g *CodeGraph) GetNodesByName(name string) []*Node {
	return g.nodesByName[name]
}

// GetOutgoingEdges returns all edges originating from a node.
func (g *CodeGraph) GetOutgoingEdges(id NodeID) []*Edge {
	return g.edgesByFrom[id]
}

// GetIncomingEdges returns all edges pointing to a node.
func (g *CodeGraph) GetIncomingEdges(id NodeID) []*Edge {
	return g.edgesByTo[id]
}

// GetCallers returns all nodes that call the given node.
func (g *CodeGraph) GetCallers(id NodeID) []*Node {
	var callers []*Node
	for _, edge := range g.edgesByTo[id] {
		if edge.Kind == EdgeCalls {
			if caller := g.Nodes[edge.From]; caller != nil {
				callers = append(callers, caller)
			}
		}
	}
	return callers
}

// GetCallees returns all nodes that the given node calls.
func (g *CodeGraph) GetCallees(id NodeID) []*Node {
	var callees []*Node
	for _, edge := range g.edgesByFrom[id] {
		if edge.Kind == EdgeCalls {
			if callee := g.Nodes[edge.To]; callee != nil {
				callees = append(callees, callee)
			}
		}
	}
	return callees
}

// RebuildIndexes rebuilds the in-memory indexes from Nodes and Edges.
// Call this after loading from disk.
func (g *CodeGraph) RebuildIndexes() {
	g.nodesByPath = make(map[string][]*Node)
	g.nodesByName = make(map[string][]*Node)
	g.edgesByFrom = make(map[NodeID][]*Edge)
	g.edgesByTo = make(map[NodeID][]*Edge)

	for _, n := range g.Nodes {
		g.nodesByPath[n.Path] = append(g.nodesByPath[n.Path], n)
		g.nodesByName[n.Name] = append(g.nodesByName[n.Name], n)
	}

	for _, e := range g.Edges {
		g.edgesByFrom[e.From] = append(g.edgesByFrom[e.From], e)
		g.edgesByTo[e.To] = append(g.edgesByTo[e.To], e)
	}
}
