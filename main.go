package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codemap/analyze"
	"codemap/cache"
	"codemap/config"
	"codemap/graph"
	"codemap/render"
	"codemap/scanner"

	ignore "github.com/sabhiram/go-gitignore"
)

func main() {
	skylineMode := flag.Bool("skyline", false, "Enable skyline visualization mode")
	animateMode := flag.Bool("animate", false, "Enable animation (use with --skyline)")
	depsMode := flag.Bool("deps", false, "Enable dependency graph mode (function/import analysis)")
	diffMode := flag.Bool("diff", false, "Only show files changed vs main (or use --ref to specify branch)")
	diffRef := flag.String("ref", "main", "Branch/ref to compare against (use with --diff)")
	jsonMode := flag.Bool("json", false, "Output JSON (for Python renderer compatibility)")
	debugMode := flag.Bool("debug", false, "Show debug info (gitignore loading, paths, etc.)")
	helpMode := flag.Bool("help", false, "Show help")

	// New flags for enhanced analysis
	detailLevel := flag.Int("detail", 0, "Detail level: 0=names, 1=signatures, 2=full (use with --deps)")
	apiMode := flag.Bool("api", false, "Show public API surface only (compact view, use with --deps)")

	// Graph/RAG mode flags
	indexMode := flag.Bool("index", false, "Build knowledge graph index")
	queryMode := flag.Bool("query", false, "Query the knowledge graph")
	queryFrom := flag.String("from", "", "Query: symbol to trace from")
	queryTo := flag.String("to", "", "Query: symbol to trace to")
	queryDepth := flag.Int("depth", 5, "Query: max traversal depth")
	forceReindex := flag.Bool("force", false, "Force rebuild index even if up-to-date")
	graphOutput := flag.String("output", "", "Output path for graph file (default: .codemap/graph.gob)")

	// LLM analysis flags
	explainMode := flag.Bool("explain", false, "Explain a symbol using LLM")
	explainSymbol := flag.String("symbol", "", "Symbol name to explain (use with --explain)")
	summarizeMode := flag.Bool("summarize", false, "Summarize a module/directory using LLM")
	llmModel := flag.String("model", "", "LLM model to use (overrides config)")
	noCache := flag.Bool("no-cache", false, "Bypass cache for LLM requests")

	// Search mode flags
	searchMode := flag.Bool("search", false, "Search the codebase using natural language")
	searchQuery := flag.String("q", "", "Search query (use with --search)")
	searchLimit := flag.Int("limit", 10, "Number of search results to return")
	searchExpand := flag.Bool("expand", false, "Expand results with callers/callees context")
	embedMode := flag.Bool("embed", false, "Generate embeddings for the knowledge graph")

	flag.Parse()

	if *helpMode {
		fmt.Println("codemap - Generate a brain map of your codebase for LLM context")
		fmt.Println()
		fmt.Println("Usage: codemap [options] [path]")
		fmt.Println()
		fmt.Println("Modes:")
		fmt.Println("  (default)          Tree view with token estimates and file sizes")
		fmt.Println("  --deps             Dependency flow map (functions, types & imports)")
		fmt.Println("  --skyline          City skyline visualization")
		fmt.Println("  --diff             Only show files changed vs a branch")
		fmt.Println("  --index            Build knowledge graph index (.codemap/graph.gob)")
		fmt.Println("  --query            Query the knowledge graph")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --help             Show this help message")
		fmt.Println("  --json             Output JSON (for programmatic use)")
		fmt.Println()
		fmt.Println("Dependency mode (--deps):")
		fmt.Println("  --detail <level>   Detail level: 0=names, 1=signatures, 2=full")
		fmt.Println("  --api              Show public API surface only (compact view)")
		fmt.Println()
		fmt.Println("Diff mode (--diff):")
		fmt.Println("  --ref <branch>     Branch to compare against (default: main)")
		fmt.Println()
		fmt.Println("Skyline mode (--skyline):")
		fmt.Println("  --animate          Enable terminal animation")
		fmt.Println()
		fmt.Println("Index mode (--index):")
		fmt.Println("  --force            Force rebuild even if index is up-to-date")
		fmt.Println("  --output <path>    Output path for graph file (default: .codemap/graph.gob)")
		fmt.Println()
		fmt.Println("Query mode (--query):")
		fmt.Println("  --from <symbol>    Find outgoing edges from symbol")
		fmt.Println("  --to <symbol>      Find incoming edges to symbol")
		fmt.Println("  --depth <n>        Max traversal depth (default: 5)")
		fmt.Println()
		fmt.Println("Explain mode (--explain):")
		fmt.Println("  --symbol <name>    Symbol name to explain")
		fmt.Println("  --model <name>     LLM model to use (overrides config)")
		fmt.Println("  --no-cache         Bypass cache for LLM requests")
		fmt.Println()
		fmt.Println("Summarize mode (--summarize):")
		fmt.Println("  --model <name>     LLM model to use (overrides config)")
		fmt.Println("  --no-cache         Bypass cache for LLM requests")
		fmt.Println()
		fmt.Println("Search mode (--search):")
		fmt.Println("  --q <query>        Natural language search query")
		fmt.Println("  --limit <n>        Number of results (default: 10)")
		fmt.Println("  --expand           Include callers/callees context")
		fmt.Println()
		fmt.Println("Embed mode (--embed):")
		fmt.Println("  --force            Force re-embedding of all symbols")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  codemap .                              # Tree with tokens")
		fmt.Println("  codemap --deps .                       # Dependencies")
		fmt.Println("  codemap --index .                      # Build graph index")
		fmt.Println("  codemap --query --from main .          # Find what main calls")
		fmt.Println("  codemap --query --to Scanner .         # Find what calls Scanner")
		fmt.Println("  codemap --query --from A --to B .      # Find path from A to B")
		fmt.Println("  codemap --explain --symbol main .      # Explain main function")
		fmt.Println("  codemap --summarize src/              # Summarize directory")
		fmt.Println("  codemap --embed .                      # Generate embeddings")
		fmt.Println("  codemap --search --q \"parse config\" . # Semantic search")
		fmt.Println("  codemap --skyline --animate .          # Animated skyline")
		fmt.Println()
		fmt.Println("Output notes:")
		fmt.Println("  ⭐️  = Top 5 largest source files")
		fmt.Println("  [!] = Large file (>8k tokens) - may need chunking for LLMs")
		os.Exit(0)
	}

	root := flag.Arg(0)
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	// Load .gitignore if it exists
	gitignore := scanner.LoadGitignore(root)

	if *debugMode {
		fmt.Fprintf(os.Stderr, "[debug] Root path: %s\n", root)
		fmt.Fprintf(os.Stderr, "[debug] Absolute path: %s\n", absRoot)
		gitignorePath := filepath.Join(root, ".gitignore")
		if gitignore != nil {
			fmt.Fprintf(os.Stderr, "[debug] Loaded .gitignore from: %s\n", gitignorePath)
		} else {
			fmt.Fprintf(os.Stderr, "[debug] No .gitignore found at: %s\n", gitignorePath)
		}
	}

	// Get changed files if --diff is specified
	var diffInfo *scanner.DiffInfo
	if *diffMode {
		var err error
		diffInfo, err = scanner.GitDiffInfo(absRoot, *diffRef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting git diff: %v\n", err)
			fmt.Fprintf(os.Stderr, "Make sure '%s' is a valid branch/ref\n", *diffRef)
			os.Exit(1)
		}
		if len(diffInfo.Changed) == 0 {
			fmt.Printf("No files changed vs %s\n", *diffRef)
			os.Exit(0)
		}
	}

	// Handle --index mode
	if *indexMode {
		runIndexMode(absRoot, root, gitignore, *forceReindex, *jsonMode, *graphOutput)
		return
	}

	// Handle --query mode
	if *queryMode {
		runQueryMode(absRoot, *queryFrom, *queryTo, *queryDepth, *jsonMode)
		return
	}

	// Handle --explain mode
	if *explainMode {
		runExplainMode(absRoot, *explainSymbol, *llmModel, *noCache, *jsonMode)
		return
	}

	// Handle --summarize mode
	if *summarizeMode {
		runSummarizeMode(absRoot, root, *llmModel, *noCache, *jsonMode)
		return
	}

	// Handle --embed mode
	if *embedMode {
		runEmbedMode(absRoot, *llmModel, *forceReindex, *jsonMode)
		return
	}

	// Handle --search mode
	if *searchMode {
		runSearchMode(absRoot, *searchQuery, *searchLimit, *searchExpand, *llmModel, *jsonMode)
		return
	}

	// Handle --deps mode separately
	if *depsMode {
		var changedFiles map[string]bool
		if diffInfo != nil {
			changedFiles = diffInfo.Changed
		}
		runDepsMode(absRoot, root, gitignore, *jsonMode, *diffRef, changedFiles, *detailLevel, *apiMode)
		return
	}

	mode := "tree"
	if *skylineMode {
		mode = "skyline"
	}

	// Scan files
	files, err := scanner.ScanFiles(root, gitignore)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking tree: %v\n", err)
		os.Exit(1)
	}

	// Filter to changed files if --diff specified (with diff info annotations)
	var impact []scanner.ImpactInfo
	var activeDiffRef string
	if diffInfo != nil {
		files = scanner.FilterToChangedWithInfo(files, diffInfo)
		impact = scanner.AnalyzeImpact(absRoot, files)
		activeDiffRef = *diffRef
	}

	project := scanner.Project{
		Root:    absRoot,
		Mode:    mode,
		Animate: *animateMode,
		Files:   files,
		DiffRef: activeDiffRef,
		Impact:  impact,
	}

	// Render or output JSON
	if *jsonMode {
		json.NewEncoder(os.Stdout).Encode(project)
	} else if *skylineMode {
		render.Skyline(project, *animateMode)
	} else {
		render.Tree(project)
	}
}

func runDepsMode(absRoot, root string, gitignore *ignore.GitIgnore, jsonMode bool, diffRef string, changedFiles map[string]bool, detailLevel int, apiMode bool) {
	loader := scanner.NewGrammarLoader()

	// Check if grammars are available
	if !loader.HasGrammars() {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "⚠️  No tree-sitter grammars found for --deps mode.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To enable dependency analysis, either:")
		fmt.Fprintln(os.Stderr, "  • Install via Homebrew: brew install JordanCoin/tap/codemap")
		fmt.Fprintln(os.Stderr, "  • Download release with grammars: https://github.com/JordanCoin/codemap/releases")
		fmt.Fprintln(os.Stderr, "  • Build from source: make deps && go build")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Or set CODEMAP_GRAMMAR_DIR to your grammars directory.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	analyses, err := scanner.ScanForDeps(root, gitignore, loader, scanner.DetailLevel(detailLevel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for deps: %v\n", err)
		os.Exit(1)
	}

	// Filter to changed files if --diff specified
	if changedFiles != nil {
		analyses = scanner.FilterAnalysisToChanged(analyses, changedFiles)
	}

	depsProject := scanner.DepsProject{
		Root:         absRoot,
		Mode:         "deps",
		Files:        analyses,
		ExternalDeps: scanner.ReadExternalDeps(absRoot),
		DiffRef:      diffRef,
		DetailLevel:  detailLevel,
	}

	// Render or output JSON
	if jsonMode {
		json.NewEncoder(os.Stdout).Encode(depsProject)
	} else if apiMode {
		render.APIView(depsProject)
	} else {
		render.Depgraph(depsProject)
	}
}

func runIndexMode(absRoot, root string, gitignore *ignore.GitIgnore, forceReindex, jsonMode bool, graphOutput string) {
	graphPath := graphOutput
	if graphPath == "" {
		graphPath = graph.GraphPath(absRoot)
	}
	loader := scanner.NewGrammarLoader()

	// Check if grammars are available
	if !loader.HasGrammars() {
		fmt.Fprintln(os.Stderr, "⚠️  No tree-sitter grammars found. Index requires --deps mode grammars.")
		os.Exit(1)
	}

	// Check if we can do an incremental update
	var existingGraph *graph.CodeGraph
	var modifiedFiles, deletedFiles []string
	isIncremental := false

	if !forceReindex && graph.Exists(graphPath) {
		existing, err := graph.LoadBinary(graphPath)
		if err == nil {
			stale, _ := graph.IsStale(existing, absRoot)
			if !stale {
				stats := existing.GetStats()
				if jsonMode {
					json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
						"status":     "up-to-date",
						"path":       graphPath,
						"nodes":      stats.TotalNodes,
						"edges":      stats.TotalEdges,
						"indexed_at": time.Unix(existing.LastIndexed, 0).Format(time.RFC3339),
					})
				} else {
					fmt.Printf("✓ Index is up-to-date (%d nodes, %d edges)\n", stats.TotalNodes, stats.TotalEdges)
					fmt.Printf("  Path: %s\n", graphPath)
					fmt.Printf("  Last indexed: %s\n", time.Unix(existing.LastIndexed, 0).Format(time.RFC3339))
				}
				return
			}
			// Graph is stale - use incremental update
			existingGraph = existing
			modifiedFiles, _ = graph.GetModifiedFiles(existing, absRoot)
			deletedFiles = graph.GetDeletedFiles(existing, absRoot)
			isIncremental = true
		}
	}

	start := time.Now()

	// Scan all files to get current state
	analyses, err := scanner.ScanForDeps(root, gitignore, loader, scanner.DetailFull)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
		os.Exit(1)
	}

	// Build set of files that need processing
	filesToProcess := make(map[string]bool)
	if isIncremental {
		// Remove deleted files from graph
		for _, path := range deletedFiles {
			existingGraph.RemoveNodesForPath(path)
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "  Removed %s\n", path)
			}
		}

		// Mark modified files for reprocessing
		for _, path := range modifiedFiles {
			existingGraph.RemoveNodesForPath(path)
			filesToProcess[path] = true
		}

		// Find new files (not in existing graph)
		for _, a := range analyses {
			if !existingGraph.IsFileInGraph(a.Path) {
				filesToProcess[a.Path] = true
			}
		}

		// If no files need updating, index is up-to-date
		if len(filesToProcess) == 0 {
			stats := existingGraph.GetStats()
			if jsonMode {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"status":     "up-to-date",
					"path":       graphPath,
					"nodes":      stats.TotalNodes,
					"edges":      stats.TotalEdges,
					"indexed_at": time.Unix(existingGraph.LastIndexed, 0).Format(time.RFC3339),
				})
			} else {
				fmt.Printf("✓ Index is up-to-date (%d nodes, %d edges)\n", stats.TotalNodes, stats.TotalEdges)
				fmt.Printf("  Path: %s\n", graphPath)
			}
			return
		}

		if !jsonMode {
			fmt.Fprintf(os.Stderr, "Incremental update: %d files\n", len(filesToProcess))
		}
	} else {
		fmt.Fprintf(os.Stderr, "Building knowledge graph index...\n")
		// Full rebuild - process all files
		for _, a := range analyses {
			filesToProcess[a.Path] = true
		}
	}

	// Create builder (with existing graph for incremental, new for full rebuild)
	var builder *graph.Builder
	if isIncremental {
		builder = graph.NewBuilder(absRoot, graph.WithProgress(func(msg string) {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "  %s\n", msg)
			}
		}), graph.WithGitignore(gitignore), graph.WithExistingGraph(existingGraph))
	} else {
		builder = graph.NewBuilder(absRoot, graph.WithProgress(func(msg string) {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "  %s\n", msg)
			}
		}), graph.WithGitignore(gitignore))
	}

	// Process files that need updating
	for _, a := range analyses {
		if !filesToProcess[a.Path] {
			continue
		}

		fa := &graph.FileAnalysis{
			Path:     a.Path,
			Language: a.Language,
			Imports:  a.Imports,
		}

		// Convert functions
		for _, f := range a.Functions {
			fa.Functions = append(fa.Functions, graph.FuncInfo{
				Name:       f.Name,
				Signature:  f.Signature,
				Receiver:   f.Receiver,
				IsExported: f.IsExported,
				Line:       f.Line,
				ParamCount: f.ParamCount,
			})
		}

		// Convert types
		for _, t := range a.Types {
			fa.Types = append(fa.Types, graph.TypeInfo{
				Name:       t.Name,
				Kind:       string(t.Kind),
				IsExported: t.IsExported,
				Line:       t.Line,
			})
		}

		// Extract calls
		callAnalysis, err := loader.ExtractCalls(filepath.Join(absRoot, a.Path))
		if err == nil && callAnalysis != nil {
			for _, c := range callAnalysis.Calls {
				fa.Calls = append(fa.Calls, graph.CallInfo{
					CallerFunc: c.CallerFunc,
					CallerLine: c.CallerLine,
					CalleeName: c.CalleeName,
					CallLine:   c.CallLine,
					Args:       c.Args,
					Receiver:   c.Receiver,
				})
			}
		}

		builder.AddFile(fa)
	}

	// Finalize graph
	builder.ResolveCallEdges()
	builder.FilterCallEdges()
	codeGraph := builder.Build()

	// Save to disk
	if err := codeGraph.SaveBinary(graphPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving index: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	stats := codeGraph.GetStats()

	status := "created"
	statusMsg := "Index built"
	if isIncremental {
		status = "updated"
		statusMsg = "Index updated"
	}

	if jsonMode {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":        status,
			"incremental":   isIncremental,
			"files_updated": len(filesToProcess),
			"path":          graphPath,
			"nodes":         stats.TotalNodes,
			"edges":         stats.TotalEdges,
			"files":         stats.FileCount,
			"functions":     stats.FunctionCount,
			"elapsed_ms":    elapsed.Milliseconds(),
		})
	} else {
		fmt.Printf("\n✓ %s in %v\n", statusMsg, elapsed.Round(time.Millisecond))
		if isIncremental {
			fmt.Printf("  Updated: %d files\n", len(filesToProcess))
		}
		fmt.Printf("  Path: %s\n", graphPath)
		fmt.Printf("  Nodes: %d (files: %d, functions: %d)\n", stats.TotalNodes, stats.FileCount, stats.FunctionCount)
		fmt.Printf("  Edges: %d\n", stats.TotalEdges)
	}
}

func runQueryMode(absRoot, fromSymbol, toSymbol string, maxDepth int, jsonMode bool) {
	graphPath := graph.GraphPath(absRoot)

	if !graph.Exists(graphPath) {
		fmt.Fprintln(os.Stderr, "No index found. Run 'codemap --index' first.")
		os.Exit(1)
	}

	codeGraph, err := graph.LoadBinary(graphPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading index: %v\n", err)
		os.Exit(1)
	}

	// No symbols specified - show stats
	if fromSymbol == "" && toSymbol == "" {
		stats := codeGraph.GetStats()
		if jsonMode {
			json.NewEncoder(os.Stdout).Encode(stats)
		} else {
			fmt.Printf("Graph Statistics:\n")
			fmt.Printf("  Total nodes: %d\n", stats.TotalNodes)
			fmt.Printf("  Total edges: %d\n", stats.TotalEdges)
			fmt.Printf("  Files: %d\n", stats.FileCount)
			fmt.Printf("  Functions: %d\n", stats.FunctionCount)
			fmt.Printf("\nNode types:\n")
			for kind, count := range stats.NodesByKind {
				fmt.Printf("  %s: %d\n", kind, count)
			}
			fmt.Printf("\nEdge types:\n")
			for kind, count := range stats.EdgesByKind {
				fmt.Printf("  %s: %d\n", kind, count)
			}
		}
		return
	}

	// Path query: from A to B
	if fromSymbol != "" && toSymbol != "" {
		fromNodes := codeGraph.FindNodesByPattern(fromSymbol, nil)
		toNodes := codeGraph.FindNodesByPattern(toSymbol, nil)

		if len(fromNodes) == 0 {
			fmt.Fprintf(os.Stderr, "No nodes found matching '%s'\n", fromSymbol)
			os.Exit(1)
		}
		if len(toNodes) == 0 {
			fmt.Fprintf(os.Stderr, "No nodes found matching '%s'\n", toSymbol)
			os.Exit(1)
		}

		// Find path between first matches
		path := codeGraph.FindPath(fromNodes[0].ID, toNodes[0].ID, maxDepth)
		if path == nil {
			fmt.Printf("No path found from '%s' to '%s' (depth=%d)\n", fromSymbol, toSymbol, maxDepth)
			return
		}

		if jsonMode {
			json.NewEncoder(os.Stdout).Encode(path)
		} else {
			fmt.Printf("Path from %s to %s (length: %d):\n\n", fromSymbol, toSymbol, path.Length)
			for i, node := range path.Path {
				fmt.Printf("  %d. %s [%s] %s:%d\n", i+1, node.Name, node.Kind, node.Path, node.Line)
				if i < len(path.Edges) {
					fmt.Printf("     └─ %s ──>\n", path.Edges[i].Kind)
				}
			}
		}
		return
	}

	// From query: what does X call?
	if fromSymbol != "" {
		nodes := codeGraph.FindNodesByPattern(fromSymbol, nil)
		if len(nodes) == 0 {
			fmt.Fprintf(os.Stderr, "No nodes found matching '%s'\n", fromSymbol)
			os.Exit(1)
		}

		type edgeResult struct {
			From  *graph.Node   `json:"from"`
			Edges []*graph.Edge `json:"edges"`
			To    []*graph.Node `json:"targets"`
		}

		var results []edgeResult
		for _, node := range nodes {
			edges := codeGraph.GetOutgoingEdges(node.ID)
			var targets []*graph.Node
			for _, e := range edges {
				if target := codeGraph.GetNode(e.To); target != nil {
					targets = append(targets, target)
				}
			}
			if len(edges) > 0 {
				results = append(results, edgeResult{From: node, Edges: edges, To: targets})
			}
		}

		if jsonMode {
			json.NewEncoder(os.Stdout).Encode(results)
		} else {
			fmt.Printf("Outgoing edges from '%s':\n\n", fromSymbol)
			for _, r := range results {
				fmt.Printf("%s [%s] %s:%d\n", r.From.Name, r.From.Kind, r.From.Path, r.From.Line)
				for i, e := range r.Edges {
					target := r.To[i]
					if target != nil {
						fmt.Printf("  └─ %s ──> %s [%s] %s:%d\n", e.Kind, target.Name, target.Kind, target.Path, target.Line)
					}
				}
				fmt.Println()
			}
		}
		return
	}

	// To query: what calls X?
	if toSymbol != "" {
		nodes := codeGraph.FindNodesByPattern(toSymbol, nil)
		if len(nodes) == 0 {
			fmt.Fprintf(os.Stderr, "No nodes found matching '%s'\n", toSymbol)
			os.Exit(1)
		}

		type edgeResult struct {
			To      *graph.Node   `json:"to"`
			Edges   []*graph.Edge `json:"edges"`
			Callers []*graph.Node `json:"callers"`
		}

		var results []edgeResult
		for _, node := range nodes {
			edges := codeGraph.GetIncomingEdges(node.ID)
			var callers []*graph.Node
			for _, e := range edges {
				if caller := codeGraph.GetNode(e.From); caller != nil {
					callers = append(callers, caller)
				}
			}
			if len(edges) > 0 {
				results = append(results, edgeResult{To: node, Edges: edges, Callers: callers})
			}
		}

		if jsonMode {
			json.NewEncoder(os.Stdout).Encode(results)
		} else {
			fmt.Printf("Incoming edges to '%s':\n\n", toSymbol)
			for _, r := range results {
				fmt.Printf("%s [%s] %s:%d\n", r.To.Name, r.To.Kind, r.To.Path, r.To.Line)
				for i, e := range r.Edges {
					caller := r.Callers[i]
					if caller != nil {
						fmt.Printf("  <── %s ── %s [%s] %s:%d\n", e.Kind, caller.Name, caller.Kind, caller.Path, caller.Line)
					}
				}
				fmt.Println()
			}
		}
	}
}

// runExplainMode handles the --explain command for LLM-powered symbol explanation.
func runExplainMode(absRoot, symbol, modelOverride string, noCache, jsonMode bool) {
	if symbol == "" {
		fmt.Fprintln(os.Stderr, "Error: --symbol is required with --explain")
		fmt.Fprintln(os.Stderr, "Usage: codemap --explain --symbol <name> [path]")
		os.Exit(1)
	}

	// Load graph to find symbol
	graphPath := filepath.Join(absRoot, ".codemap", "graph.gob")
	if !graph.Exists(graphPath) {
		fmt.Fprintln(os.Stderr, "No index found. Run 'codemap --index' first.")
		os.Exit(1)
	}

	codeGraph, err := graph.LoadBinary(graphPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading index: %v\n", err)
		os.Exit(1)
	}

	// Find matching nodes
	nodes := codeGraph.FindNodesByPattern(symbol, nil)
	if len(nodes) == 0 {
		fmt.Fprintf(os.Stderr, "No symbols found matching '%s'\n", symbol)
		os.Exit(1)
	}

	// Use first match
	node := nodes[0]
	if len(nodes) > 1 && !jsonMode {
		fmt.Fprintf(os.Stderr, "Found %d matches for '%s', using: %s (%s:%d)\n",
			len(nodes), symbol, node.Name, node.Path, node.Line)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Use defaults if no config
		cfg = config.DefaultConfig()
	}

	// Override model if specified
	if modelOverride != "" {
		cfg.LLM.Model = modelOverride
	}

	// Create LLM client
	client, err := analyze.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating LLM client: %v\n", err)
		fmt.Fprintln(os.Stderr, "Check your configuration (~/.config/codemap/config.yaml)")
		os.Exit(1)
	}

	// Check connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to LLM provider (%s): %v\n", client.Name(), err)
		os.Exit(1)
	}

	// Read source code
	source, err := analyze.ReadSymbolSource(absRoot, node)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source: %v\n", err)
		os.Exit(1)
	}

	// Initialize cache
	cacheOpts := cache.Options{
		Dir:     filepath.Join(absRoot, ".codemap", "cache"),
		Enabled: !noCache && cfg.Cache.Enabled,
	}
	responseCache, _ := cache.New(cacheOpts)

	// Check cache
	operation := "explain"
	if responseCache != nil && responseCache.Enabled() {
		if entry, ok := responseCache.GetByContentHash(source.ContentHash, operation, cfg.LLM.Model); ok {
			// Cache hit
			if jsonMode {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"symbol":      node.Name,
					"path":        node.Path,
					"line":        node.Line,
					"cached":      true,
					"model":       entry.Model,
					"explanation": entry.Response,
				})
			} else {
				fmt.Printf("## %s\n\n", node.Name)
				fmt.Printf("*%s:%d* (cached)\n\n", node.Path, node.Line)
				fmt.Println(entry.Response)
			}
			return
		}
	}

	// Generate prompt
	messages := analyze.ExplainSymbolPrompt(source)

	// Make request
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "Explaining %s using %s...\n", node.Name, cfg.LLM.Model)
	}

	reqCtx, reqCancel := context.WithTimeout(context.Background(), time.Duration(cfg.LLM.Timeout)*time.Second)
	defer reqCancel()

	resp, err := client.Complete(reqCtx, &analyze.CompletionRequest{
		Messages:    messages,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from LLM: %v\n", err)
		os.Exit(1)
	}

	// Cache response
	if responseCache != nil && responseCache.Enabled() {
		responseCache.SetResponse(source.ContentHash, operation, resp.Model, resp.Content, &cache.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// Output result
	if jsonMode {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"symbol":      node.Name,
			"path":        node.Path,
			"line":        node.Line,
			"kind":        node.Kind.String(),
			"cached":      false,
			"model":       resp.Model,
			"explanation": resp.Content,
			"usage": map[string]int{
				"prompt_tokens":     resp.Usage.PromptTokens,
				"completion_tokens": resp.Usage.CompletionTokens,
				"total_tokens":      resp.Usage.TotalTokens,
			},
			"duration_ms": resp.Duration.Milliseconds(),
		})
	} else {
		fmt.Printf("\n## %s\n\n", node.Name)
		fmt.Printf("*%s:%d* | %s | %d tokens | %v\n\n", node.Path, node.Line, resp.Model,
			resp.Usage.TotalTokens, resp.Duration.Round(time.Millisecond))
		fmt.Println(resp.Content)
	}
}

// runSummarizeMode handles the --summarize command for LLM-powered module summarization.
func runSummarizeMode(absRoot, targetPath, modelOverride string, noCache, jsonMode bool) {
	// Resolve target path (can be relative to current directory or absolute)
	var modulePath string
	if filepath.IsAbs(targetPath) {
		modulePath = targetPath
	} else {
		modulePath = filepath.Join(absRoot, targetPath)
	}

	// Ensure path exists
	info, err := os.Stat(modulePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: path not found: %s\n", targetPath)
		os.Exit(1)
	}

	// Get relative path for display
	relPath, _ := filepath.Rel(absRoot, modulePath)
	if relPath == "" || relPath == "." {
		relPath = filepath.Base(absRoot)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if modelOverride != "" {
		cfg.LLM.Model = modelOverride
	}

	// Create LLM client
	client, err := analyze.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating LLM client: %v\n", err)
		fmt.Fprintln(os.Stderr, "Check your configuration (~/.config/codemap/config.yaml)")
		os.Exit(1)
	}

	// Check connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to LLM provider (%s): %v\n", client.Name(), err)
		os.Exit(1)
	}

	// Read source files
	var sources []*analyze.SymbolSource

	if info.IsDir() {
		sources, err = analyze.ReadModuleSource(absRoot, relPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading module: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Single file
		source, err := analyze.ReadSymbolSource(absRoot, &graph.Node{
			Kind: graph.KindFile,
			Name: filepath.Base(targetPath),
			Path: relPath,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		sources = []*analyze.SymbolSource{source}
	}

	if len(sources) == 0 {
		fmt.Fprintln(os.Stderr, "No source files found in the specified path")
		os.Exit(1)
	}

	// Compute combined content hash for caching
	var combinedHash string
	{
		var hashes []string
		for _, s := range sources {
			hashes = append(hashes, s.ContentHash)
		}
		combinedHash = analyze.ContentHash(fmt.Sprintf("%v", hashes))
	}

	// Initialize cache
	cacheOpts := cache.Options{
		Dir:     filepath.Join(absRoot, ".codemap", "cache"),
		Enabled: !noCache && cfg.Cache.Enabled,
	}
	responseCache, _ := cache.New(cacheOpts)

	// Check cache
	operation := "summarize"
	if responseCache != nil && responseCache.Enabled() {
		if entry, ok := responseCache.GetByContentHash(combinedHash, operation, cfg.LLM.Model); ok {
			// Cache hit
			if jsonMode {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"path":    relPath,
					"files":   len(sources),
					"cached":  true,
					"model":   entry.Model,
					"summary": entry.Response,
				})
			} else {
				fmt.Printf("## %s\n\n", relPath)
				fmt.Printf("*%d files* (cached)\n\n", len(sources))
				fmt.Println(entry.Response)
			}
			return
		}
	}

	// Generate prompt
	messages := analyze.SummarizeModulePrompt(relPath, sources)

	// Make request
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "Summarizing %s (%d files) using %s...\n", relPath, len(sources), cfg.LLM.Model)
	}

	reqCtx, reqCancel := context.WithTimeout(context.Background(), time.Duration(cfg.LLM.Timeout)*time.Second)
	defer reqCancel()

	resp, err := client.Complete(reqCtx, &analyze.CompletionRequest{
		Messages:    messages,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from LLM: %v\n", err)
		os.Exit(1)
	}

	// Cache response
	if responseCache != nil && responseCache.Enabled() {
		responseCache.SetResponse(combinedHash, operation, resp.Model, resp.Content, &cache.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// Output result
	if jsonMode {
		// Collect file names
		var fileNames []string
		for _, s := range sources {
			fileNames = append(fileNames, s.Node.Name)
		}

		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"path":    relPath,
			"files":   fileNames,
			"cached":  false,
			"model":   resp.Model,
			"summary": resp.Content,
			"usage": map[string]int{
				"prompt_tokens":     resp.Usage.PromptTokens,
				"completion_tokens": resp.Usage.CompletionTokens,
				"total_tokens":      resp.Usage.TotalTokens,
			},
			"duration_ms": resp.Duration.Milliseconds(),
		})
	} else {
		fmt.Printf("\n## %s\n\n", relPath)
		fmt.Printf("*%d files* | %s | %d tokens | %v\n\n", len(sources), resp.Model,
			resp.Usage.TotalTokens, resp.Duration.Round(time.Millisecond))
		fmt.Println(resp.Content)
	}
}
