package graph

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

// Builder constructs a CodeGraph from a codebase.
type Builder struct {
	graph      *CodeGraph
	rootPath   string
	gitignore  *ignore.GitIgnore
	progress   func(msg string)
	fileCount  int
	errorCount int
}

// BuilderOption configures the graph builder.
type BuilderOption func(*Builder)

// WithProgress sets a progress callback.
func WithProgress(fn func(msg string)) BuilderOption {
	return func(b *Builder) {
		b.progress = fn
	}
}

// WithGitignore sets the gitignore matcher.
func WithGitignore(gi *ignore.GitIgnore) BuilderOption {
	return func(b *Builder) {
		b.gitignore = gi
	}
}

// WithExistingGraph sets an existing graph for incremental updates.
func WithExistingGraph(g *CodeGraph) BuilderOption {
	return func(b *Builder) {
		b.graph = g
	}
}

// NewBuilder creates a new graph builder.
func NewBuilder(rootPath string, opts ...BuilderOption) *Builder {
	b := &Builder{
		graph:    NewCodeGraph(rootPath),
		rootPath: rootPath,
		progress: func(msg string) {}, // Default: no-op
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// FileAnalysis represents the analysis result from scanner.
type FileAnalysis struct {
	Path      string
	Language  string
	Functions []FuncInfo
	Types     []TypeInfo
	Imports   []string
	Calls     []CallInfo
}

// FuncInfo represents a function/method from scanner.
type FuncInfo struct {
	Name       string
	Signature  string
	Receiver   string
	IsExported bool
	Line       int
	EndLine    int
	ParamCount int // Number of parameters (-1 if unknown/variadic)
}

// TypeInfo represents a type definition from scanner.
type TypeInfo struct {
	Name       string
	Kind       string
	IsExported bool
	Line       int
}

// CallInfo represents a function call from scanner.
type CallInfo struct {
	CallerFunc string
	CallerLine int
	CalleeName string
	CallLine   int
	Args       int
	Receiver   string
}

// AddFile adds a file's analysis to the graph.
func (b *Builder) AddFile(analysis *FileAnalysis) error {
	if analysis == nil {
		return nil
	}

	b.fileCount++
	b.progress(fmt.Sprintf("Processing %s", analysis.Path))

	// Create file node
	fileID := GenerateNodeID(analysis.Path, "")
	fileNode := &Node{
		ID:      fileID,
		Kind:    KindFile,
		Name:    filepath.Base(analysis.Path),
		Path:    analysis.Path,
		Package: getPackageFromPath(analysis.Path),
	}
	b.graph.AddNode(fileNode)

	// Process functions
	funcNodes := make(map[string]NodeID) // name -> nodeID for call resolution
	for _, fn := range analysis.Functions {
		funcID := GenerateNodeID(analysis.Path, fn.Name)
		funcNode := &Node{
			ID:         funcID,
			Kind:       kindFromFunc(fn),
			Name:       fn.Name,
			Path:       analysis.Path,
			Line:       fn.Line,
			EndLine:    fn.EndLine,
			Signature:  fn.Signature,
			Exported:   fn.IsExported,
			ParamCount: fn.ParamCount,
		}
		b.graph.AddNode(funcNode)
		funcNodes[fn.Name] = funcID

		// File contains function
		b.graph.AddEdge(&Edge{
			From: fileID,
			To:   funcID,
			Kind: EdgeContains,
		})
	}

	// Process types
	for _, t := range analysis.Types {
		typeID := GenerateNodeID(analysis.Path, t.Name)
		typeNode := &Node{
			ID:       typeID,
			Kind:     KindType,
			Name:     t.Name,
			Path:     analysis.Path,
			Line:     t.Line,
			Exported: t.IsExported,
		}
		b.graph.AddNode(typeNode)

		// File contains type
		b.graph.AddEdge(&Edge{
			From: fileID,
			To:   typeID,
			Kind: EdgeContains,
		})
	}

	// Process imports
	for _, imp := range analysis.Imports {
		impID := GenerateNodeID(imp, "")
		impNode := &Node{
			ID:   impID,
			Kind: KindPackage,
			Name: filepath.Base(imp),
			Path: imp,
		}
		b.graph.AddNode(impNode)

		// File imports package
		b.graph.AddEdge(&Edge{
			From: fileID,
			To:   impID,
			Kind: EdgeImports,
		})
	}

	// Process calls (create edges between functions)
	for _, call := range analysis.Calls {
		// Find caller node
		callerID, ok := funcNodes[call.CallerFunc]
		if !ok {
			continue // Caller not found (might be global scope)
		}

		// Try to find callee node
		// First, look in the same file
		calleeID, ok := funcNodes[call.CalleeName]
		if !ok {
			// Create a placeholder node for the callee (will be resolved later)
			calleeID = GenerateNodeID("", call.CalleeName) // Global lookup
		}

		b.graph.AddEdge(&Edge{
			From:     callerID,
			To:       calleeID,
			Kind:     EdgeCalls,
			Line:     call.CallLine,
			CallSite: call.CalleeName,
			ArgCount: call.Args,
		})
	}

	return nil
}

// Build finalizes the graph and returns it.
func (b *Builder) Build() *CodeGraph {
	b.graph.LastIndexed = time.Now().Unix()
	b.graph.NodeCount = len(b.graph.Nodes)
	b.graph.EdgeCount = len(b.graph.Edges)

	b.progress(fmt.Sprintf("Built graph: %d nodes, %d edges from %d files",
		b.graph.NodeCount, b.graph.EdgeCount, b.fileCount))

	return b.graph
}

// ResolveCallEdges attempts to resolve placeholder callee nodes to actual nodes.
// Call this after all files have been added.
func (b *Builder) ResolveCallEdges() {
	// Build name lookup index
	nameToNodes := make(map[string][]*Node)
	for _, node := range b.graph.Nodes {
		if node.Kind == KindFunction || node.Kind == KindMethod {
			nameToNodes[node.Name] = append(nameToNodes[node.Name], node)
		}
	}

	// Update edges
	for _, edge := range b.graph.Edges {
		if edge.Kind != EdgeCalls {
			continue
		}

		// Check if callee exists
		if b.graph.GetNode(edge.To) != nil {
			continue
		}

		// Try to find by name
		candidates := nameToNodes[edge.CallSite]
		if len(candidates) == 1 {
			edge.To = candidates[0].ID
		} else if len(candidates) > 1 {
			// Ambiguous: use heuristics (same package preferred)
			callerNode := b.graph.GetNode(edge.From)
			if callerNode != nil {
				for _, c := range candidates {
					if c.Path == callerNode.Path || getPackageFromPath(c.Path) == getPackageFromPath(callerNode.Path) {
						edge.To = c.ID
						break
					}
				}
			}
		}
	}
}

// kindFromFunc determines the NodeKind based on function info.
func kindFromFunc(fn FuncInfo) NodeKind {
	if fn.Receiver != "" {
		return KindMethod
	}
	return KindFunction
}

// getPackageFromPath extracts a package name from a file path.
func getPackageFromPath(path string) string {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return ""
	}
	return filepath.Base(dir)
}

// FilterCallEdges removes call edges that violate import constraints.
// This implements the ImportGraphFilter from the plan.
func (b *Builder) FilterCallEdges() {
	// Build file -> imported packages map
	fileImports := make(map[NodeID]map[string]bool)

	for _, edge := range b.graph.Edges {
		if edge.Kind != EdgeImports {
			continue
		}
		fromNode := b.graph.GetNode(edge.From)
		toNode := b.graph.GetNode(edge.To)
		if fromNode == nil || toNode == nil {
			continue
		}

		if _, ok := fileImports[edge.From]; !ok {
			fileImports[edge.From] = make(map[string]bool)
		}
		fileImports[edge.From][toNode.Name] = true
	}

	// Filter call edges
	validEdges := make([]*Edge, 0, len(b.graph.Edges))
	for _, edge := range b.graph.Edges {
		if edge.Kind != EdgeCalls {
			validEdges = append(validEdges, edge)
			continue
		}

		// Get caller's file
		callerNode := b.graph.GetNode(edge.From)
		if callerNode == nil {
			continue
		}

		callerFileID := GenerateNodeID(callerNode.Path, "")

		// Get callee
		calleeNode := b.graph.GetNode(edge.To)
		if calleeNode == nil {
			continue // Unresolved callee
		}

		// Arity check: skip if argument count doesn't match parameter count
		// ParamCount -1 means variadic (always matches)
		// ParamCount 0 with ArgCount 0 is valid (no params)
		if calleeNode.ParamCount >= 0 && edge.ArgCount != calleeNode.ParamCount {
			continue // Arity mismatch
		}

		// Same file? Always valid
		if callerNode.Path == calleeNode.Path {
			validEdges = append(validEdges, edge)
			continue
		}

		// Check if callee's package is imported
		calleePackage := getPackageFromPath(calleeNode.Path)
		imports := fileImports[callerFileID]
		if imports != nil && imports[calleePackage] {
			validEdges = append(validEdges, edge)
			continue
		}

		// Also check for standard library and external calls
		// For now, keep edges where callee package looks like an import
		if strings.Contains(calleePackage, "/") {
			validEdges = append(validEdges, edge)
		}
	}

	b.graph.Edges = validEdges
	b.graph.RebuildIndexes()
}
