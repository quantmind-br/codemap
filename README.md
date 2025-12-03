# codemap: Codebase Analysis and Visualization Tool

## Project Overview

**codemap** is a powerful command-line interface (CLI) tool written in Go designed to analyze, map, and visualize the structure and dependencies of a codebase. It provides developers and automated systems (like LLM agents) with a clear, structured view of a project's architecture without needing to manually traverse the source code.

### Purpose and Main Functionality
The primary purpose of `codemap` is to transform raw source code into structured data models (`Project`, `DepsProject`) that can be consumed for various purposes, including:
1.  **Visualization:** Generating hierarchical file trees, "city skyline" representations of file complexity, and dependency graphs.
2.  **Context Generation:** Providing detailed, structured JSON output of a codebase's structure, functions, types, and imports, which is ideal for feeding into Large Language Models (LLMs) for context-aware tasks.
3.  **Change Impact Analysis:** Identifying and analyzing only the files that have changed relative to a Git reference, allowing for focused analysis and visualization.

### Key Features and Capabilities
*   **Multi-Mode Analysis:** Supports Tree View, Skyline View, Dependency Graph, and Public API View.
*   **Deep Code Parsing:** Uses **Tree-sitter** grammars for language-aware parsing to extract functions, types, and imports with configurable detail levels.
*   **Git Integration:** Respects `.gitignore` rules and performs Git diff analysis to focus on changed files.
*   **Machine-Readable Output:** Can output all analysis results as structured JSON for programmatic consumption.
*   **Model Context Protocol (MCP) Server:** Includes an optional server mode to expose its capabilities as tools for LLM agents.

### Likely Intended Use Cases
*   **Developer Onboarding:** Quickly generating a visual map of a new codebase.
*   **Code Review:** Analyzing the structural impact and dependencies of a pull request (`--diff` mode).
*   **LLM Tooling:** Serving as a reliable, structured data source for AI-powered code analysis and generation tools.
*   **Architectural Audits:** Mapping internal and external dependencies to identify coupling issues.

## Table of Contents
1.  [Project Overview](#project-overview)
2.  [Architecture](#architecture)
3.  [C4 Model Architecture](#c4-model-architecture)
4.  [Repository Structure](#repository-structure)
5.  [Dependencies and Integration](#dependencies-and-integration)
6.  [API Documentation](#api-documentation)
7.  [Development Notes](#development-notes)
8.  [Known Issues and Limitations](#known-issues-and-limitations)
9.  [Additional Documentation](#additional-documentation)

## Architecture

The `codemap` application follows a clear **Layered Architecture** or **Pipeline Pattern**, ensuring a strong separation of concerns between data acquisition, core logic, and presentation.

### High-level Architecture Overview
The execution flow is strictly unidirectional:

1.  **Control/Orchestration Layer (`main`):** Parses CLI flags and determines the execution mode.
2.  **Data Acquisition/Analysis Layer (`scanner`):** Performs all I/O (file system, Git) and core domain logic (Tree-sitter parsing, data modeling).
3.  **Presentation Layer (`render`):** Consumes the structured data models from the `scanner` and formats them for terminal output (TUI) or JSON serialization.

### Technology Stack and Frameworks
| Component | Technology | Purpose |
| :--- | :--- | :--- |
| **Core Language** | Go | Primary development language. |
| **Code Parsing** | Tree-sitter | Language-agnostic parsing engine for deep code analysis. |
| **C Interface** | `github.com/ebitengine/purego` | Used to interface with the Tree-sitter C libraries without Cgo. |
| **Terminal UI** | `github.com/charmbracelet/bubbletea` | Framework for building rich, interactive terminal user interfaces (TUI), used for Skyline and Depgraph visualizations. |
| **Git Integration** | `github.com/sabhiram/go-gitignore` | Used for loading and applying `.gitignore` rules. |
| **Server Mode** | `github.com/modelcontextprotocol/go-sdk` | Used to implement the Model Context Protocol (MCP) server. |

### Component Relationships (with mermaid diagrams)

The following diagram illustrates the high-level flow of control and data between the core internal packages.

```mermaid
graph LR
    subgraph Orchestration
        A[main Package (CLI Entry)]
        B[mcp/main Package (Server Entry)]
    end

    subgraph Core Logic
        C[scanner Package]
        D[scanner/types.go (Data Models)]
    end

    subgraph Presentation
        E[render Package]
    end

    A --> C : Calls Scan/Analyze
    B --> C : Calls Scan/Analyze
    C --> D : Defines/Populates Data Models
    A --> E : Passes Data Models for Output
    B --> E : Passes Data Models for Output
    E .-> D : Consumes Data Models (Project, DepsProject)

    style C fill:#ccf,stroke:#333,stroke-width:2px
    style E fill:#f9f,stroke:#333,stroke-width:2px
    style A fill:#afa,stroke:#333
    style B fill:#afa,stroke:#333
    style D fill:#eee,stroke:#999
```

### Key Design Patterns
*   **Data Transfer Object (DTO) Pattern:** The `scanner/types.go` structs (`Project`, `DepsProject`, `FileInfo`, etc.) act as pure data containers, defining the contract between the `scanner` and `render` layers.
*   **Strategy Pattern (Implicit):** The `main` function selects the appropriate rendering strategy (`render.Tree`, `render.Skyline`, `render.Depgraph`) based on the user's CLI flags.
*   **Adapter Pattern (Implicit):** The `scanner` package acts as an adapter, translating the language-agnostic Abstract Syntax Tree (AST) output from the Tree-sitter C libraries into the application's canonical Go DTOs.

## C4 Model Architecture

### <details><summary>Context Diagram (Level 1)</summary>

```mermaid
C4Context
    title Context Diagram for codemap
    Person(developer, "Developer/User", "Interacts with the tool via CLI to analyze code.")
    Person(llm_agent, "LLM Agent", "Consumes analysis data via the MCP server.")
    System(codemap, "codemap", "Codebase Analysis and Visualization Tool. Generates structural and dependency maps.")
    System_Ext(git, "Code Repository (Git)", "Provides file history, diff information, and ignore rules.")
    System_Ext(filesystem, "Local File System", "Source of all code files and project structure.")

    developer --> codemap : Executes commands (CLI)
    llm_agent --> codemap : Requests analysis (MCP Protocol)
    codemap --> git : Reads diffs and ignore rules
    codemap --> filesystem : Scans and reads source files
```
</details>

### <details><summary>Container Diagram (Level 2)</summary>

```mermaid
C4Container
    title Container Diagram for codemap
    System_Boundary(codemap_system, "codemap")
        Container(cli, "CLI Application", "Go Executable", "Handles command-line arguments and orchestrates analysis/rendering.")
        Container(mcp_server, "MCP Server", "Go Executable (mcp/main)", "Exposes core functionality as tools via the Model Context Protocol.")
        Container(scanner, "Scanner Package", "Go Library (scanner)", "Core logic: file traversal, Git integration, Tree-sitter parsing, data modeling.")
        Container(renderer, "Renderer Package", "Go Library (render)", "Presentation layer: formats data for TUI (Tree, Skyline, Depgraph) or JSON output.")
    System_Boundary(codemap_system)

    System_Ext(git, "Git", "Provides diff and ignore data.")
    System_Ext(terminal, "Terminal/TUI", "Displays visual output (Tree, Skyline, Depgraph).")
    System_Ext(llm_agent, "LLM Agent", "Consumes structured data.")
    System_Ext(grammars, "Tree-sitter Grammars", "C Libraries/Data", "Language-specific parsing rules.")

    cli --> scanner : Calls analysis functions
    cli --> renderer : Passes Project/DepsProject for output
    mcp_server --> scanner : Calls analysis functions
    mcp_server --> renderer : Passes Project/DepsProject for output

    scanner --> git : Reads repository state
    scanner --> grammars : Loads language parsing logic (via purego)

    renderer --> terminal : Renders TUI/Text output (via bubbletea)
    mcp_server --> llm_agent : Serves JSON analysis (MCP Protocol)
```
</details>

## Repository Structure

| Directory/File | Purpose |
| :--- | :--- |
| `/` | Contains the main application entry point (`main.go`) and configuration files. |
| `/scanner` | **Core Logic:** Houses the analysis engine, including file system traversal, Git integration, Tree-sitter parsing, and all core data models (`types.go`). |
| `/render` | **Presentation Layer:** Contains logic for all visualization modes (Tree, Skyline, Depgraph) and terminal formatting. |
| `/mcp` | Contains the entry point (`main.go`) and handlers for the Model Context Protocol (MCP) server implementation. |
| `/development-docs` | Stores detailed planning and technical documentation for feature development. |

## Dependencies and Integration

### Internal Package Dependencies
The project maintains a clear, hierarchical dependency structure:

| Package | Depends On | Nature of Dependency |
| :--- | :--- | :--- |
| **main** (`/`) | `scanner`, `render` | Orchestration (Control flow and data passing). |
| **mcp/main** | `scanner`, `render` | Orchestration (Server handlers). |
| **render** | `scanner` | Data Coupling (Consumes data structures like `scanner.Project`). |
| **scanner** | *None* | Highly cohesive core logic module. |

### External Service Integrations
The application integrates with two primary external systems:

1.  **Git:**
    *   The `scanner` package executes Git commands to load `.gitignore` rules (`LoadGitignore`) and calculate file differences (`GitDiffInfo`) against a specified reference branch.
    *   This integration is crucial for performance and for enabling the change impact analysis (`--diff` mode).

2.  **Model Context Protocol (MCP):**
    *   The `mcp/main.go` package implements an MCP server using the `go-sdk`.
    *   This integration allows the `codemap` tool to be called programmatically by LLM agents, exposing its core analysis functions (`get_structure`, `get_dependencies`, `find_symbol`, etc.) as structured tools.

## API Documentation

The `codemap` tool's API is defined by its command-line interface and the structured JSON output it produces when the `--json` flag is used.

### 1. Basic Structure Analysis (Tree/Skyline Mode)

This mode provides basic file metadata, size, and optional diff statistics.

| Attribute | Detail |
| :--- | :--- |
| **Method** | CLI Execution |
| **Path** | `codemap [path] [--skyline] [--diff] --json` |
| **Output Model** | `scanner.Project` |

**Key Request Parameters (Flags):**
| Parameter | Description |
| :--- | :--- |
| `[path]` | The root directory to scan (defaults to `.`). |
| `--skyline` | Enables the skyline visualization mode (TUI output only). |
| `--diff` | Filters the output to only include files changed relative to the Git reference (`--ref`). |
| `--json` | **Required** for machine-readable output. |

**Response Format (`scanner.Project` JSON):**
The response is a JSON object containing project metadata and an array of `FileInfo` objects.

```json
{
  "root": "./src",
  "mode": "tree",
  "files": [
    {
      "path": "file.go",
      "size": 3500,
      "ext": ".go",
      "tokens": 1000,
      "added": 15,
      "removed": 0
    }
  ],
  "impact": [ /* ... ImpactInfo objects if --diff is used */ ]
}
```

### 2. Deep Dependency Analysis (Dependency Graph/API View Mode)

This mode performs deep code parsing to extract structural elements and dependencies.

| Attribute | Detail |
| :--- | :--- |
| **Method** | CLI Execution |
| **Path** | `codemap --deps [path] [--detail N] --json` |
| **Output Model** | `scanner.DepsProject` |

**Key Request Parameters (Flags):**
| Parameter | Description |
| :--- | :--- |
| `--deps` | **Required** to enable deep code analysis mode. |
| `--detail N` | Sets the verbosity of extracted symbols: `0` (names only), `1` (names + signatures), `2` (signatures + type fields). |
| `--api` | Renders a compact view of only public (exported) symbols (TUI output only). |
| `--json` | **Required** for machine-readable output. |

**Response Format (`scanner.DepsProject` JSON):**
The response is a JSON object containing project metadata, an array of `FileAnalysis` objects, and a map of external dependencies.

```json
{
  "root": ".",
  "mode": "deps",
  "detail_level": 1,
  "files": [
    {
      "path": "service/api.go",
      "language": "go",
      "functions": [
        {
          "name": "NewClient",
          "signature": "func NewClient(cfg Config) *Client",
          "exported": true,
          "line": 42
        }
      ],
      "types": [ /* ... TypeInfo objects */ ],
      "imports": [ "fmt", "net/http" ]
    }
  ],
  "external_deps": {
    "github.com/external/lib": [ "service/api.go" ]
  }
}
```

## Development Notes

### Project-Specific Conventions
*   **Explicit Dependency Passing:** The application favors explicit dependency passing (e.g., passing the `GrammarLoader` and `GitIgnore` objects) rather than global state or formal DI containers, which is idiomatic for Go.
*   **Data-Centric Design:** The `scanner` package is designed to be a pure data producer, and the `render` package is a pure data consumer. All communication between layers is via the DTOs defined in `scanner/types.go`.
*   **Error Handling:** Errors are typically handled immediately by printing to `os.Stderr` and exiting with a non-zero status code, as is common for CLI tools.

### Testing Requirements
*   **Unit Testing:** Critical logic within the `scanner` package (e.g., `IsExportedName`, file filtering, token estimation heuristics) requires robust unit tests to ensure correctness across different languages and configurations.
*   **Integration Testing:** End-to-end tests are necessary to verify the entire pipeline, from CLI flag parsing to the final output (both TUI and JSON), especially for complex modes like `--deps` and `--diff`.
*   **Grammar Testing:** The Tree-sitter parsing logic must be tested against various language code snippets to ensure accurate extraction of `FuncInfo` and `TypeInfo` at different `DetailLevel` settings.

### Performance Considerations
*   **Grammar Loading:** The application relies on dynamically loading Tree-sitter grammars via `purego`. The performance of `scanner.NewGrammarLoader()` is critical, and grammar availability is a prerequisite for deep analysis.
*   **File Traversal:** The use of `.gitignore` rules via `scanner.LoadGitignore` is essential for pruning the file system traversal and maintaining performance on large repositories.
*   **Token Estimation:** The current token estimation (`Tokens` field in `FileInfo`) is based on a simple character-per-token heuristic. For more accurate performance analysis, this should be replaced or augmented with language-aware token counting.

## Known Issues and Limitations

*   **Tree-sitter Grammar Management:** The application requires pre-built Tree-sitter grammars to be available in a specific location. If grammars are missing, the `--deps` mode will fail, requiring manual setup by the user.
*   **Brittle Dependency Parsing:** The logic for reading external dependencies (e.g., `go.mod`, `package.json`) is implemented manually within `scanner/deps.go`. This logic is brittle and may break if package manager file formats change or if complex features (like conditional dependencies) are introduced.
*   **Tight TUI Coupling:** The `render` package is heavily coupled to the `charmbracelet/bubbletea` ecosystem. Changing the terminal rendering strategy would require a significant rewrite of the presentation layer.
*   **Incomplete Features:** The `AnalyzeImpact` function is inferred but its full implementation and accuracy, especially for complex dependency chains, may be a source of technical debt or an area for future enhancement.

## Additional Documentation
The following internal documents provide deeper insight into the project's design and future plans:

*   [Enhanced Code Analysis Plan](/development-docs/0001-enhanced-code-analysis-plan.md)
*   [Token Heuristics and Symbol Search Plan](/development-docs/0002-token-heuristics-symbol-search-plan.md)
*   [Project Overview (Internal)](/.serena/memories/project_overview.md)
*   *Note: Additional documentation on the specific implementation details of the Tree-sitter parsing logic would be highly beneficial.*
