The application is a command-line interface (CLI) tool written in Go, designed to analyze a codebase and generate various outputs (tree view, dependency graph, knowledge graph, LLM-based analysis). The "request flow" is therefore the **command execution flow**.

# Request Flow Analysis

## Entry Points Overview

The sole entry point for the application is the standard Go `main()` function in `main.go`.

1.  **Initial Receipt:** The application receives a command-line invocation, including flags and an optional path argument.
2.  **Flag Parsing:** `flag.Parse()` processes all command-line arguments, setting boolean flags (e.g., `--deps`, `--index`, `--skyline`) and string/int parameters (e.g., `--ref`, `--symbol`, `--depth`).
3.  **Path Resolution:** The root path is determined from the positional argument (`flag.Arg(0)`) or defaults to `.` (current directory). The absolute path (`absRoot`) is calculated.

## Request Routing Map

The `main()` function acts as the central command router, dispatching control flow based on the parsed flags. The routing is sequential and mutually exclusive for the major modes:

| Command Mode | Flag | Handler Function | Core Logic |
| :--- | :--- | :--- | :--- |
| **Help** | `--help` | (Inline) | Prints usage and exits. |
| **Index** | `--index` | `runIndexMode` | Builds/updates the knowledge graph (`.codemap/graph.gob`). |
| **Query** | `--query` | `runQueryMode` | Loads the graph and performs pathfinding or edge traversal. |
| **Explain** | `--explain` | `runExplainMode` | Loads the graph, retrieves context, and calls LLM for explanation. |
| **Summarize** | `--summarize` | `runSummarizeMode` | Scans files, retrieves context, and calls LLM for summary. |
| **Embed** | `--embed` | `runEmbedMode` | Loads the graph and generates vector embeddings for nodes. |
| **Search** | `--search` | `runSearchMode` | Performs semantic search using embeddings and LLM. |
| **Dependencies** | `--deps` | `runDepsMode` | Scans files for dependencies and renders the dependency graph. |
| **Default/Tree/Skyline** | (None/`--skyline`) | (Inline) | Scans files for tokens/size and renders the file tree or skyline visualization. |

## Middleware Pipeline

The following steps are executed in `main()` before control is passed to a specific mode handler, acting as a global preprocessing pipeline:

1.  **Configuration Loading:** `config.Load()` is called implicitly by the `analyze` package functions (e.g., `analyze.NewClient`) and explicitly in `run*Mode` functions to get LLM settings.
2.  **Gitignore Filtering:** `scanner.LoadGitignore(root)` loads the `.gitignore` file, which is passed to all subsequent scanning functions to exclude ignored files.
3.  **Diff Analysis (Conditional):** If the `--diff` flag is set, `scanner.GitDiffInfo` is called to determine the set of changed files relative to the specified `--ref`. This information is used to filter the results in the default and `--deps` modes.

## Controller/Handler Analysis

The `run*Mode` functions encapsulate the core business logic:

| Handler | Key Components | Request/Context Propagation | Response Formation |
| :--- | :--- | :--- | :--- |
| `runIndexMode` | `scanner.ScanForDeps`, `graph.NewBuilder`, `graph.CodeGraph.SaveBinary` | `absRoot`, `gitignore`, `forceReindex` | Prints status/stats to `os.Stderr`, outputs JSON to `os.Stdout` if `--json`. |
| `runQueryMode` | `graph.LoadBinary`, `codeGraph.FindNodesByPattern`, `codeGraph.FindPath` | `absRoot` (for graph path), `fromSymbol`, `toSymbol`, `maxDepth` | Prints path/edges/stats to `os.Stdout`, or outputs JSON. |
| `runExplainMode` | `graph.LoadBinary`, `analyze.NewClient`, `analyze.ExplainSymbol` | `absRoot`, `explainSymbol`, `llmModel`, `noCache` | Prints LLM explanation to `os.Stdout`, or outputs JSON. |
| `runSummarizeMode` | `scanner.ScanFiles`, `analyze.NewClient`, `analyze.SummarizeModule` | `root`, `llmModel`, `noCache` | Prints LLM summary to `os.Stdout`, or outputs JSON. |
| `runEmbedMode` | `graph.LoadBinary`, `analyze.NewClient`, `analyze.EmbedGraph` | `absRoot`, `llmModel`, `forceReindex` | Prints status/stats to `os.Stdout`, or outputs JSON. |
| `runSearchMode` | `graph.LoadBinary`, `analyze.NewClient`, `analyze.Search` | `absRoot`, `searchQuery`, `searchLimit`, `searchExpand` | Prints search results to `os.Stdout`, or outputs JSON. |
| `runDepsMode` | `scanner.NewGrammarLoader`, `scanner.ScanForDeps`, `render.Depgraph`, `render.APIView` | `root`, `gitignore`, `detailLevel`, `apiMode` | Renders the dependency graph using `render` package, or outputs JSON. |
| **Default Flow** | `scanner.ScanFiles`, `scanner.AnalyzeImpact`, `render.Tree`, `render.Skyline` | `root`, `gitignore`, `diffInfo`, `animateMode` | Renders the file tree or skyline visualization, or outputs JSON. |

## Authentication & Authorization Flow

The application is a local CLI tool and does not implement any internal authentication or authorization mechanisms for file access or command execution.

*   **External LLM Authentication:** The `analyze` package, used by `runExplainMode`, `runSummarizeMode`, `runEmbedMode`, and `runSearchMode`, relies on external configuration (e.g., environment variables or config file) to authenticate with LLM providers (OpenAI, Anthropic, Gemini, etc.). This is handled by `config.Load()` and `analyze.NewClient()`.

## Error Handling Pathways

Error handling is primarily focused on file system operations, git commands, and graph loading:

1.  **Fatal Errors (Exit 1):** Errors that prevent the core operation (e.g., invalid root path, failed git diff, missing tree-sitter grammars, failure to load/save the graph index) result in an error message printed to `os.Stderr` and a call to `os.Exit(1)`.
2.  **Grammar Check:** `runDepsMode` and `runIndexMode` explicitly check for the presence of tree-sitter grammars using `loader.HasGrammars()` and exit with a helpful message if they are missing.
3.  **Graph Existence Check:** `runQueryMode`, `runExplainMode`, `runEmbedMode`, and `runSearchMode` check if the graph file (`.codemap/graph.gob`) exists and exit if it does not, prompting the user to run `--index`.
4.  **LLM Errors:** LLM-related functions (e.g., `analyze.ExplainSymbol`) return errors which are printed to `os.Stderr` in their respective `run*Mode` functions.

## Request Lifecycle Diagram

```mermaid
graph TD
    A[CLI Invocation] --> B{main()};
    B --> C[flag.Parse()];
    C --> D[Path & Gitignore Setup];
    D --> E{Mode Check};

    E -- --help --> F[Print Help & Exit];

    E -- --index --> G[runIndexMode];
    G --> G1[Load/Scan Files (ScanForDeps)];
    G1 --> G2[Build Graph (graph.Builder)];
    G2 --> G3[Save Graph (graph.SaveBinary)];

    E -- --query --> H[runQueryMode];
    H --> H1[Load Graph (graph.LoadBinary)];
    H1 --> H2[Execute Query (FindPath/GetOutgoingEdges)];

    E -- --explain/--summarize/--embed/--search --> I[LLM Modes];
    I --> I1[Load Graph (if needed)];
    I1 --> I2[Init LLM Client (analyze.NewClient)];
    I2 --> I3[Execute LLM Task (analyze.*)];

    E -- --deps --> J[runDepsMode];
    J --> J1[Scan Files (ScanForDeps)];
    J1 --> J2[Render Depgraph (render.Depgraph/APIView)];

    E -- Default/--skyline --> K[Default Flow];
    K --> K1[Scan Files (ScanFiles)];
    K1 --> K2[Filter/Analyze Impact (if --diff)];
    K2 --> K3[Render Output (render.Tree/Skyline)];

    G3 & H2 & I3 & J2 & K3 --> L{Output Format?};
    L -- --json --> M[JSON to stdout];
    L -- Default --> N[Formatted Text/Visualization to stdout];
```