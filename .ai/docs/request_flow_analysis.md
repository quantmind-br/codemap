# Request Flow Analysis

The application, `codemap`, is a command-line interface (CLI) tool written in Go. Therefore, the "request flow" is interpreted as the **execution flow** triggered by command-line arguments and flags. The `main` function acts as the central router, dispatching control to specialized functions based on the mode flags provided by the user.

## Entry Points Overview

The single entry point is the `main` function in `main.go`. Execution is immediately branched based on the presence of specific flags, defining distinct operational modes.

| Mode Flag | Handler Function | Description | Core Components Involved |
| :--- | :--- | :--- | :--- |
| `--help` | (Inline in `main`) | Prints usage information and exits. | `fmt`, `os` |
| `--index` | `runIndexMode` | Builds the knowledge graph index. | `scanner`, `graph`, `cache` |
| `--query` | `runQueryMode` | Queries the knowledge graph for symbol paths. | `graph` |
| `--explain` | `runExplainMode` | Uses an LLM to explain a specific symbol. | `graph`, `analyze`, `config`, `cache` |
| `--summarize` | `runSummarizeMode` | Uses an LLM to summarize a directory or module. | `scanner`, `analyze`, `config`, `cache` |
| `--embed` | `runEmbedMode` | Generates vector embeddings for the knowledge graph. | `graph`, `analyze`, `config` |
| **Default** | `runDefaultMode` (Implicit) | Generates a file tree, dependency graph, or skyline visualization. | `scanner`, `render` |

## Request Routing Map

The routing mechanism is a sequential series of conditional checks in `main.go` after flag parsing.

1.  **Initialization & Flag Parsing:**
    *   `flag.Parse()` processes all command-line arguments.
    *   The root path is determined (`root := flag.Arg(0)`, defaults to `"."`).
    *   Absolute path (`absRoot`) is calculated.
2.  **Preprocessing/Setup:**
    *   `.gitignore` is loaded via `scanner.LoadGitignore(root)`.
    *   If `--debug` is set, debug information is printed.
    *   If `--diff` is set, `scanner.GitDiffInfo` is called to determine the set of files to process.
3.  **Mode Dispatch (Exclusive Modes):**
    *   If `--index` is set, execution is dispatched to `runIndexMode` and terminates.
    *   If `--query` is set, execution is dispatched to `runQueryMode` and terminates.
    *   If `--explain` is set, execution is dispatched to `runExplainMode` and terminates.
    *   If `--summarize` is set, execution is dispatched to `runSummarizeMode` and terminates.
    *   If `--embed` is set, execution is dispatched to `runEmbedMode` and terminates.
4.  **Default/Visualization Mode:**
    *   If none of the exclusive modes are set, the flow continues to the default rendering logic.
    *   `scanner.Scan` is called to analyze the codebase.
    *   The results are passed to the `render` package (`render.Tree`, `render.DepGraph`, or `render.Skyline`) based on `--deps` or `--skyline` flags.

## Middleware Pipeline

In this CLI context, the "Middleware Pipeline" is the sequence of setup and preprocessing steps applied to the input path before the main handler logic is executed.

1.  **Argument Validation:** Checks if the root path is valid and converts it to an absolute path.
2.  **Configuration Loading:** (Implicitly handled by components like `analyze` and `cache`) The `config.Load()` function is used to load settings, particularly for LLM and caching parameters.
3.  **Git Ignore Filtering:** `scanner.LoadGitignore(root)` loads the `.gitignore` file, which is then used by `scanner.Scan` to exclude files from processing. This acts as a file-level filter middleware.
4.  **Diff Filtering (Conditional):** If the `--diff` flag is present, `scanner.GitDiffInfo` is executed. This step filters the list of files to be processed down to only those changed relative to the specified reference branch.
5.  **Caching Setup:** The `cache.NewCache` function is called within the `run*Mode` handlers to initialize the caching layer, which acts as a pre-check for expensive operations (like LLM calls or graph building).

## Controller/Handler Analysis

The core logic resides in the `run*Mode` functions and the default rendering logic.

### 1. `runIndexMode` (Graph Building)

*   **Input:** `absRoot`, `gitignore`, `forceReindex`, `graphOutput`.
*   **Process:**
    *   Initializes `graph.Store` and `graph.Builder`.
    *   Calls `scanner.Scan` to traverse the file system and parse code symbols.
    *   `scanner.Scan` returns a list of `scanner.File` objects.
    *   `graph.Builder.Build` processes the scanned files to create the knowledge graph (nodes and edges).
    *   `graph.Store.Save` persists the graph to disk (default: `.codemap/graph.gob`).

### 2. `runQueryMode` (Graph Querying)

*   **Input:** `absRoot`, `queryFrom`, `queryTo`, `queryDepth`.
*   **Process:**
    *   Initializes `graph.Store` and loads the graph from disk.
    *   If both `--from` and `--to` are provided, it calls `graph.Query.FindPath`.
    *   If only `--from` or `--to` is provided, it calls `graph.Query.FindEdges`.
    *   Results are formatted and printed (or output as JSON).

### 3. `runExplainMode` (LLM Explanation)

*   **Input:** `absRoot`, `explainSymbol`, `llmModel`, `noCache`.
*   **Process:**
    *   Loads configuration and initializes the LLM client (`analyze.NewClient`).
    *   Initializes `graph.Store` and loads the graph.
    *   `analyze.ExplainSymbol` is called, which uses the graph to retrieve the symbol's context and then sends a request to the LLM client for explanation.
    *   The LLM response is printed.

### 4. `runSummarizeMode` (LLM Summarization)

*   **Input:** `root` (path to summarize), `llmModel`, `noCache`.
*   **Process:**
    *   Loads configuration and initializes the LLM client.
    *   `scanner.Scan` is used to get the contents of the target directory/module.
    *   `analyze.SummarizeModule` is called, which uses the LLM client to generate a summary based on the file contents.
    *   The summary is printed.

### 5. Default Rendering Mode

*   **Process:**
    *   `scanner.Scan` is the primary operation, performing file traversal, token counting, and dependency analysis (if `--deps` is set).
    *   The resulting `scanner.ScanResult` is passed to the `render` package.
    *   If `--skyline` is set, `render.Skyline` is called.
    *   If `--deps` is set, `render.DepGraph` is called.
    *   Otherwise, `render.Tree` is called for the default file tree view.

## Authentication & Authorization Flow

The `codemap` application is a local CLI tool and does not handle external user requests or maintain user sessions.

*   **Authentication:** Not applicable.
*   **Authorization:** Not applicable. Access control is managed by the operating system's file permissions for the user running the tool.
*   **API Keys:** For LLM-related modes (`--explain`, `--summarize`, `--embed`), API keys (e.g., `OPENAI_API_KEY`, `GEMINI_API_KEY`) are loaded from the environment or configuration file via the `config` package. This is a form of service authentication, not user authentication.

## Error Handling Pathways

Error handling is primarily localized within each mode handler and the core utility functions.

1.  **Initial Setup Errors:**
    *   Failure to get the absolute path (`filepath.Abs`) results in an error message to `os.Stderr` and `os.Exit(1)`.
    *   Failure to get git diff info (`scanner.GitDiffInfo`) results in an error message and `os.Exit(1)`.
2.  **Graph/LLM Mode Errors:**
    *   In `runQueryMode`, `runExplainMode`, and `runEmbedMode`, failure to load the graph (`graph.Store.Load`) results in a fatal error message indicating the index must be built first.
    *   LLM client initialization errors (e.g., missing API key, invalid model) are handled by `analyze.NewClient` and typically result in a fatal error message.
    *   Errors during the core operation (e.g., `analyze.ExplainSymbol` returning an error) are printed to `os.Stderr` and the program exits gracefully or continues, depending on the severity.
3.  **Scanner Errors:**
    *   `scanner.Scan` handles file reading and parsing errors internally, often logging them to debug output or skipping the problematic file, allowing the overall scan to complete.

## Request Lifecycle Diagram

The lifecycle is a linear execution path determined by the initial command-line flags.

```mermaid
graph TD
    A[Start: main()] --> B{Parse Flags & Args};
    B --> C[Determine Root Path];
    C --> D[Load .gitignore];
    D --> E{Is --diff?};
    E -- Yes --> F[scanner.GitDiffInfo];
    E -- No --> G;
    F --> G[File List Determined];

    G --> H{Mode Dispatch};

    subgraph Exclusive Modes (Exit after execution)
        H -- --index --> I[runIndexMode];
        I --> I1[scanner.Scan];
        I1 --> I2[graph.Builder.Build];
        I2 --> I3[graph.Store.Save];
        I3 --> Z[Exit];

        H -- --query --> J[runQueryMode];
        J --> J1[graph.Store.Load];
        J1 --> J2[graph.Query.FindPath/Edges];
        J2 --> Z;

        H -- --explain --> K[runExplainMode];
        K --> K1[analyze.NewClient];
        K1 --> K2[analyze.ExplainSymbol];
        K2 --> Z;

        H -- --summarize --> L[runSummarizeMode];
        L --> L1[analyze.NewClient];
        L1 --> L2[scanner.Scan];
        L2 --> L3[analyze.SummarizeModule];
        L3 --> Z;

        H -- --embed --> M[runEmbedMode];
        M --> M1[analyze.NewClient];
        M1 --> M2[graph.Embedder.Embed];
        M2 --> Z;
    end

    H -- Default/Render --> N[Default Mode Execution];
    N --> O[scanner.Scan];
    O --> P{Check Render Flags};
    P -- --skyline --> Q[render.Skyline];
    P -- --deps --> R[render.DepGraph];
    P -- Default --> S[render.Tree];
    Q --> Z;
    R --> Z;
    S --> Z;
```