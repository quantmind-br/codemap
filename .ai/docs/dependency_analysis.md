The project is a Go application named `codemap` that functions as a codebase analysis tool, generating visualizations and dependency graphs for LLM context. It has two main entry points: the command-line tool (`main.go`) and an MCP (Model Context Protocol) server (`mcp/main.go`).

The core functionality revolves around file scanning, parsing code using Tree-sitter grammars, and rendering the results.

# Dependency Analysis

## Internal Dependencies Map

The project is structured into three main internal packages: `main`, `scanner`, and `render`. The `mcp` directory also contains a `main` package for the server implementation.

| Package | Depends On | Description |
| :--- | :--- | :--- |
| **main** (`/`) | `codemap/render` | Uses `render` for `Skyline`, `Tree`, and `Depgraph` visualizations. |
| | `codemap/scanner` | Uses `scanner` for all core logic: `LoadGitignore`, `GitDiffInfo`, `ScanFiles`, `AnalyzeImpact`, `NewGrammarLoader`, `ScanForDeps`, `FilterAnalysisToChanged`, and `ReadExternalDeps`. |
| **mcp/main** | `codemap/render` | Uses `render` for `Tree`, `Depgraph`, and `APIView` visualizations within the MCP server handlers. |
| | `codemap/scanner` | Uses `scanner` for all core logic: `LoadGitignore`, `ScanFiles`, `NewGrammarLoader`, `ScanForDeps`, `GitDiffInfo`, `FilterToChangedWithInfo`, `AnalyzeImpact`, `FindFiles`, `FindImporters`, `ListProjects`, and `FindSymbol`. |
| **scanner** | *None* | The `scanner` package is the core logic module. It is highly cohesive, containing all file system operations, git integration, Tree-sitter parsing, and dependency reading logic. It does not depend on `render` or `main`. |
| **render** | `codemap/scanner` | The `render` package depends on data structures defined in `scanner` (e.g., `scanner.Project`, `scanner.DepsProject`, `scanner.ImpactInfo`). |

**Key Dependency Flow:**
`main` / `mcp/main` (Entry Points) $\rightarrow$ `scanner` (Core Logic) $\rightarrow$ `render` (Output/Presentation)

*Note: The `render` package's dependency on `scanner` is for data types, which is a common and acceptable pattern for a presentation layer consuming a core data layer.*

## External Libraries Analysis

The project is built with Go and manages its dependencies via `go.mod`.

| Library | Version | Purpose |
| :--- | :--- | :--- |
| `github.com/charmbracelet/bubbletea` | `v1.3.10` | Used for building interactive terminal UIs (TUI), likely for the animated skyline or other interactive modes. |
| `github.com/ebitengine/purego` | `v0.9.1` | A library for calling C functions from Go without Cgo. This is crucial for interfacing with the Tree-sitter C libraries/grammars. |
| `github.com/sabhiram/go-gitignore` | `v0.0.0-20210923224102-525f6e181f06` | Used by the `scanner` package to load and apply `.gitignore` rules for file filtering. |
| `github.com/tree-sitter/go-tree-sitter` | `v0.25.0` | The primary library for parsing code. It provides the Go bindings for the Tree-sitter parsing engine. |
| `golang.org/x/term` | `v0.37.0` | Standard Go library extension for controlling the terminal, likely used by `bubbletea` or for raw terminal input/output. |
| `github.com/modelcontextprotocol/go-sdk` | `v1.1.0` | Used exclusively in `mcp/main.go` to implement the Model Context Protocol server, allowing the tool to be called by LLM agents. |
| `github.com/google/jsonschema-go` | `v0.3.0` | Used in `mcp/main.go` (indirectly via `go-sdk`) to generate JSON schemas for the MCP tool definitions. |
| **Charmbracelet Ecosystem** | *Various* | Indirect dependencies like `lipgloss`, `colorprofile`, `x/ansi`, `x/cellbuf`, `x/term` are pulled in by `bubbletea` for rich terminal rendering and styling. |

## Service Integrations

The project integrates with two primary "services" or external systems:

1.  **Git:** The `scanner` package (specifically `scanner/git.go`) integrates with the local Git system to perform:
    *   Loading `.gitignore` rules (`LoadGitignore`).
    *   Calculating file differences between the current state and a specified reference (`GitDiffInfo`).
    *   Analyzing the impact of changed files (`AnalyzeImpact`).

2.  **Model Context Protocol (MCP):** The `mcp/main.go` package acts as an MCP server, exposing the core `codemap` functionality as a set of tools for LLMs. This is a direct integration using the `github.com/modelcontextprotocol/go-sdk`. The exposed tools are:
    *   `get_structure`
    *   `get_dependencies`
    *   `get_diff`
    *   `find_file`
    *   `get_importers`
    *   `get_symbol`
    *   `list_projects`
    *   `status`

## Dependency Injection Patterns

The project uses a simple, explicit dependency passing pattern, typical for small-to-medium Go applications, rather than a formal Dependency Injection (DI) container.

*   **GrammarLoader:** The `scanner.NewGrammarLoader()` function creates the `GrammarLoader` instance, which is then explicitly passed to `scanner.ScanForDeps` in both `main.go` and `mcp/main.go`. This decouples the scanning logic from the grammar discovery/loading mechanism.
*   **GitIgnore:** The `ignore.GitIgnore` object is loaded in the entry points (`main.go`, `mcp/main.go`) and then passed down to the `scanner` functions (`ScanFiles`, `ScanForDeps`) that need to respect file exclusion rules.
*   **Context Passing (MCP):** The MCP server handlers (`handleGetStructure`, `handleGetDependencies`, etc.) receive a `context.Context` and the parsed input struct, adhering to the standard Go pattern for service handlers.

## Module Coupling Assessment

The project exhibits a high degree of **cohesion** within the `scanner` package and a clear separation of concerns between the three main modules.

*   **High Cohesion in `scanner`:** The `scanner` package is responsible for all data acquisition and processing (file I/O, git, parsing, dependency calculation). This is excellent, as all related logic is contained in one place.
*   **Low Coupling between `scanner` and `render`:** The `render` package is only coupled to the data structures (structs) defined in `scanner`. It does not call back into `scanner` for logic, making it a pure presentation layer.
*   **Entry Point Coupling:** Both `main` and `mcp/main` are tightly coupled to both `scanner` and `render`, as they orchestrate the entire application flow (load config $\rightarrow$ scan $\rightarrow$ render). This is expected for entry point packages.

The overall coupling is healthy, with a unidirectional flow of control and data from the entry points down to the core logic and then to the presentation layer.

## Dependency Graph

The project's dependency graph is primarily linear and hierarchical.

```mermaid
graph TD
    subgraph Entry Points
        A[main.go (CLI)]
        B[mcp/main.go (Server)]
    end

    subgraph Core Logic
        C[scanner Package]
    end

    subgraph Presentation
        D[render Package]
    end

    subgraph External Libraries
        E[go-tree-sitter]
        F[purego]
        G[go-gitignore]
        H[bubbletea]
        I[go-sdk (MCP)]
    end

    A --> C
    A --> D
    B --> C
    B --> D

    C --> E
    C --> F
    C --> G

    D --> H

    B --> I

    D .-> C
    C .-> C
    style D fill:#f9f,stroke:#333
    style C fill:#ccf,stroke:#333
    style A fill:#afa,stroke:#333
    style B fill:#afa,stroke:#333
    style E fill:#eee,stroke:#999
    style F fill:#eee,stroke:#999
    style G fill:#eee,stroke:#999
    style H fill:#eee,stroke:#999
    style I fill:#eee,stroke:#999
```

**Key Relationships:**
*   **`scanner` $\leftrightarrow$ `render` (Data Coupling):** `render` consumes data types from `scanner`.
*   **`scanner` $\rightarrow$ Tree-sitter (`E`, `F`):** Core dependency for code parsing.
*   **`render` $\rightarrow$ `bubbletea` (`H`):** Core dependency for TUI rendering.
*   **`mcp/main` $\rightarrow$ `go-sdk` (`I`):** Core dependency for server functionality.

## Potential Dependency Issues

1.  **Tree-sitter Grammar Management:** The `scanner` package relies on the availability of pre-built Tree-sitter grammars (C libraries) and uses `purego` to load them dynamically. This creates a build/deployment dependency where the Go binary must be bundled with or have access to the correct grammar files. The `runDepsMode` function in `main.go` explicitly checks for this, indicating it's a known point of failure.
2.  **Tight Coupling to Charmbracelet:** The `render` package is heavily dependent on the `bubbletea` and `lipgloss` ecosystem for its terminal output. While this provides a rich UI, changing the rendering strategy would require a significant rewrite of the `render` package.
3.  **Manual Dependency Parsing:** The `scanner/deps.go` file contains manual parsing logic (`parseGoMod`, `parsePackageJson`, etc.) for various manifest files. This logic is brittle and will break if the format of these files changes or if more complex features (like conditional dependencies) are introduced. Relying on official package manager tools or more robust parsing libraries would improve maintainability.
4.  **Indirect Dependencies:** The `go.mod` file shows a large number of indirect dependencies, primarily from the `charmbracelet` ecosystem. While Go modules manage this, it increases the overall complexity and potential attack surface of the build.