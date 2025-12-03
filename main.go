package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
		fmt.Println("Examples:")
		fmt.Println("  codemap .                        # Tree with tokens (~3.5 chars/token)")
		fmt.Println("  codemap --deps .                 # Dependencies with line numbers")
		fmt.Println("  codemap --deps --detail 1 .      # With function signatures")
		fmt.Println("  codemap --deps --api .           # Public API surface only")
		fmt.Println("  codemap --diff                   # Changed files vs main")
		fmt.Println("  codemap --diff --ref develop     # Changed files vs develop")
		fmt.Println("  codemap --skyline --animate .    # Animated skyline")
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
