# Request Flow Analysis
## Entry Points Overview
The application is a command-line interface (CLI) tool written in Go. The entire control flow begins and is dispatched from a single entry point:

*   **Entry Point:** `main.main()` in `main.go`.

The `main` function is responsible for parsing command-line flags, performing initial setup (like path resolution and gitignore loading), and then dispatching control to a specific mode function based on the flags provided by the user.

## Request Routing Map
The "routing" mechanism is a sequential check of boolean flags within the `main()` function, which determines the operational mode and calls the corresponding handler function. Only one mode is executed per invocation.

| Mode Flag | Handler Function | Description |
| :--- | :--- | :--- |
| `--help` | (Inline) | Prints usage information and exits. |
| `--index` | `runIndexMode` | Builds the knowledge graph index. |
| `--query` | `runQueryMode` | Queries the existing knowledge graph. |
| `--explain` | `runExplainMode` | Explains a symbol using an LLM. |
| `--summarize` | `runSummarizeMode` | Summarizes a module/directory using an LLM. |
| `--embed` | `runEmbedMode` | Generates vector embeddings for the graph. |
| `--search` | `runSearchMode` | Performs semantic or graph search. |
| `--deps` | `runDepsMode` | Generates a dependency flow map. |
| **Default** | (Inline) | Executes the default file tree view or skyline visualization. |

## Middleware Pipeline
In a CLI context, the middleware pipeline consists of initial setup and preprocessing steps executed before the main mode handler is called.

1.  **Flag Parsing:** `flag.Parse()` processes all command-line arguments and populates the mode flags and options (e.g., `--skyline`, `--ref`, `--detail`).
2.  **Path Resolution:** The root path argument is resolved to an absolute path (`filepath.Abs(root)`).
3.  **Gitignore Loading:** `scanner.LoadGitignore(root)` loads the project's `.gitignore` file to filter files during scanning.
4.  **Diff Analysis (Conditional):** If the `--diff` flag is set, `scanner.GitDiffInfo` is called to determine the set of changed files against the specified reference branch (`--ref`). This result is passed to subsequent mode handlers for filtering.

## Controller/Handler Analysis
The core logic resides in the `run*Mode` functions, which act as controllers for the application's various features.

### Data Flow for Graph/LLM Modes (`runIndexMode`, `runQueryMode`, `runExplainMode`, `runSummarizeMode`, `runEmbedMode`, `runSearchMode`)
1.  **Configuration Loading:** `config.Load()` is called to retrieve application settings, particularly for LLM and caching.
2.  **Graph Loading:** Most modes (`query`, `explain`, `embed`, `search`) start by loading the serialized `graph.CodeGraph` from `.codemap/graph.gob` using `graph.LoadBinary`.
3.  **LLM Client Initialization:** LLM-dependent modes (`explain`, `summarize`, `embed`, `search`) create an LLM client via `analyze.NewClient(cfg)`.
4.  **Caching:** `cache.New` is initialized. For `explain` and `summarize`, a cache check (`responseCache.GetByContentHash`) is performed before making an LLM request.
5.  **LLM Interaction:**
    *   **Prompt Generation:** Functions like `analyze.ExplainSymbolPrompt` or `analyze.SummarizeModulePrompt` prepare the request payload.
    *   **API Call:** `client.Complete` is called with a context timeout (`context.WithTimeout`) to execute the LLM request.
    *   **Caching:** The response is stored using `responseCache.SetResponse`.

### Core Scanning and Rendering Modes
*   **`runDepsMode`:** Uses `scanner.ScanForDeps` with a `scanner.NewGrammarLoader` to perform deep dependency analysis (functions, types, calls). The results are rendered by `render.Depgraph` or `render.APIView`.
*   **Default Mode:** Uses `scanner.ScanFiles` for a shallow file scan (size, token count). Results are passed to `render.Tree` or `render.Skyline`.

## Authentication & Authorization Flow
The application is a local CLI tool and does not implement any user-facing authentication or authorization flow. Access control is managed by the operating system's file permissions.

For LLM interactions, the application relies on API keys or credentials configured in the `~/.config/codemap/config.yaml` file, which are loaded by `config.Load()` and used by `analyze.NewClient` to establish a connection with the LLM provider.

## Error Handling Pathways
Error handling is synchronous and immediate, following standard Go CLI practices:

1.  **Error Check:** Errors are checked immediately after functions that can fail (e.g., `filepath.Abs`, `scanner.GitDiffInfo`, `graph.LoadBinary`, `client.Ping`, `client.Complete`).
2.  **Output to Stderr:** Error messages are printed to the standard error stream using `fmt.Fprintf(os.Stderr, ...)`.
3.  **Termination:** The application terminates with a non-zero exit code (`os.Exit(1)`) upon encountering a critical error (e.g., file not found, failed LLM connection, missing grammars).
4.  **Graceful Failures:** In `runEmbedMode` and `runSearchMode`, failure to load the vector index results in a warning and a fallback to graph-only search, rather than immediate termination.

## Request Lifecycle Diagram

```mermaid
graph TD
    A[CLI Invocation] --> B(main.main);
    B --> C{Parse Flags};
    C --> D(Resolve Root Path & Load .gitignore);
    D -- --diff flag --> E(scanner.GitDiffInfo);
    D -- No diff --> F{Mode Dispatch};

    subgraph Mode Handlers
        F -- --index --> G(runIndexMode);
        F -- --query --> H(runQueryMode);
        F -- --explain/--summarize/--embed/--search --> I(LLM/Graph Modes);
        F -- --deps --> J(runDepsMode);
        F -- Default/--skyline --> K(Default Scan & Render);
    end

    subgraph LLM/Graph Modes
        I --> I1(config.Load);
        I1 --> I2(graph.LoadBinary);
        I2 --> I3(analyze.NewClient);
        I3 --> I4(client.Ping);
        I4 --> I5{Cache Check};
        I5 -- Cache Miss --> I6(client.Complete);
        I6 --> I7(Cache Write);
        I7 --> I8(Output Result);
        I5 -- Cache Hit --> I8;
    end

    subgraph Index Mode
        G --> G1(scanner.ScanForDeps);
        G1 --> G2{Incremental Check};
        G2 -- Stale/Force --> G3(graph.NewBuilder);
        G3 --> G4(builder.AddFile & Resolve Edges);
        G4 --> G5(codeGraph.SaveBinary);
    end

    subgraph Dependency Mode
        J --> J1(scanner.ScanForDeps);
        J1 --> J2(render.Depgraph/APIView);
    end

    subgraph Default Mode
        K --> K1(scanner.ScanFiles);
        K1 --> K2(render.Tree/Skyline);
    end

    E --> F;
    G5 --> Z(Exit 0);
    H --> Z;
    I8 --> Z;
    J2 --> Z;
    K2 --> Z;
    B --> B_Err{Error Check};
    B_Err -- Error --> Z_Err(Output Error & Exit 1);
```