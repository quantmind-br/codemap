# Dependency Analysis

## Internal Dependencies Map

The project is structured into several highly cohesive Go packages, with the `main` package acting as the central orchestrator and dependency assembler.

| Package | Description | Key Dependencies |
| :--- | :--- | :--- |
| **main** | CLI entry point, flag parsing, and mode execution. | `config`, `scanner`, `cache`, `graph`, `analyze`, `render` |
| **scanner** | Code parsing, file system traversal, and Git integration. | `config`, `tree-sitter/go-tree-sitter` (External), `go-gitignore` (External) |
| **graph** | Knowledge Graph construction, storage, and querying (RAG backend). | `scanner` (consumes parsed data), `cache` |
| **analyze** | LLM client management, RAG logic, and embedding generation. | `config`, `graph` (for context retrieval), `cache`, LLM Clients (internal/external) |
| **cache** | Persistence layer for graph data (`.codemap/graph.gob`) and LLM responses. | Standard library (`encoding/gob`, `os`) |
| **config** | Application configuration loading (e.g., API keys, model names). | `gopkg.in/yaml.v3` (External) |
| **render** | Terminal output and visualization (tree, skyline, dependency graph). | `scanner`, `graph`, `charmbracelet/bubbletea` (External) |

**Key Dependency Flow:**
1.  **Initialization:** `main` loads `config` and initializes `cache`.
2.  **Scanning:** `main` calls `scanner` to traverse the codebase and parse files.
3.  **Indexing:** `main` calls `graph` to build the knowledge graph from `scanner` output, persisting via `cache`.
4.  **LLM/RAG:** `main` calls `analyze` for `explain`, `summarize`, or `search` modes. `analyze` uses `graph` for context retrieval.
5.  **Output:** `main` calls `render` to display results from `scanner` or `graph`.

## External Libraries Analysis

The project is a Go application that relies heavily on two main categories of external dependencies: **CLI/TUI** and **Code Analysis/LLM**.

| Dependency | Version | Category | Purpose |
| :--- | :--- | :--- | :--- |
| `github.com/tree-sitter/go-tree-sitter` | `v0.25.0` | Code Analysis | Go bindings for the core Tree-sitter parsing engine, used by the `scanner` package. |
| `github.com/ebitengine/purego` | `v0.9.1` | Code Analysis | Used for dynamic C function calls, likely to interface with the compiled Tree-sitter C grammars. |
| `github.com/charmbracelet/bubbletea` | `v1.3.10` | CLI/TUI | Framework for building the interactive terminal user interface, especially for the `--skyline` and default tree views. |
| `github.com/modelcontextprotocol/go-sdk` | `v1.1.0` | LLM/RAG | SDK for integrating with the Model Context Protocol, standardizing LLM interactions. |
| `github.com/sabhiram/go-gitignore` | `v0.0.0-...` | Utility | Parsing and matching rules from `.gitignore` files to exclude paths from scanning. |
| `gopkg.in/yaml.v3` | `v3.0.1` | Utility | Configuration file parsing. |
| `golang.org/x/term` | `v0.37.0` | Utility | Low-level terminal manipulation. |

**Indirect Dependencies (TUI Ecosystem):**
A large number of indirect dependencies (e.g., `github.com/charmbracelet/lipgloss`, `muesli/termenv`, `rivo/uniseg`) are pulled in by `bubbletea` to provide rich, cross-platform terminal rendering.

## Service Integrations

The `analyze` package is the dedicated integration layer for external AI services, abstracting the communication behind a common interface (`analyze.Client`).

| Service/Integration Point | Internal Package | Protocol/Client | Purpose |
| :--- | :--- | :--- | :--- |
| **OpenAI API** | `analyze/openai.go` | HTTP/Go SDK (Inferred) | General-purpose LLM for `explain`, `summarize`, and `search`. |
| **Google Gemini API** | `analyze/gemini.go` | HTTP/Go SDK (Inferred) | Alternative LLM provider integration. |
| **Anthropic API** | `analyze/anthropic.go` | HTTP/Go SDK (Inferred) | Alternative LLM provider integration. |
| **Ollama** | `analyze/ollama.go` | HTTP/Go SDK (Inferred) | Integration for local/self-hosted LLMs. |
| **Model Context Protocol (MCP)** | `analyze` | `modelcontextprotocol/go-sdk` | Standardizes the RAG pipeline and context formatting for LLM calls. |
| **Git** | `scanner/git.go` | OS Command Execution (Inferred) | Used to determine changed files (`--diff` mode) and repository root. |

## Dependency Injection Patterns

The project uses a pragmatic, explicit dependency passing approach typical of idiomatic Go, avoiding a heavy DI framework.

1.  **Explicit Construction and Passing:** Dependencies are instantiated in `main.go` and passed as arguments to the functions that need them (e.g., `runIndexMode` receives the `gitignore` matcher and the root path).
2.  **Factory Pattern for Abstraction:** The `analyze` package uses a factory pattern (`analyze/factory.go`) to create the correct LLM client (`analyze.Client` interface implementation) based on the `--model` flag or configuration. This decouples the core RAG logic from the specific LLM provider implementation.
3.  **Global Configuration:** The `config` package likely provides a singleton or globally accessible configuration structure, which is then used by other packages (`analyze`, `scanner`) to initialize their components (e.g., setting API keys or model names).

## Module Coupling Assessment

The project exhibits a layered architecture with clear, unidirectional dependencies, leading to high cohesion within modules and manageable coupling between them.

| Relationship | Type of Coupling | Assessment | Rationale |
| :--- | :--- | :--- | :--- |
| **main -> All Packages** | Control/Data Coupling | Necessary Orchestration | `main` is the entry point, responsible for wiring and controlling the flow between all core components. |
| **graph -> scanner** | Data Coupling | Tight, but Justified | `graph` relies entirely on the data structures (`File`, `Symbol`, `Call`) produced by `scanner`. This is the core data pipeline. |
| **analyze -> graph** | Data/Functional Coupling | Tight, but Justified | The RAG features in `analyze` require direct access to the `graph.Store` for vector search and traversal. |
| **render -> scanner/graph** | Data Coupling | Loose/Data-Driven | `render` consumes data structures from `scanner` (file tree) and `graph` (dependency edges) to visualize them, but does not modify them. |
| **scanner -> External Grammars** | Configuration/Build Coupling | High | The `scanner` package is tightly coupled to the Tree-sitter C grammars (in `/scanner/.grammar-build/`) via the build process (`build-grammars.sh`) and `purego` bindings. |

**Overall Cohesion:** High. Each package has a single, well-defined responsibility (parsing, graphing, LLM interaction, rendering).
**Overall Coupling:** Moderate. The core data flow (`scanner` -> `graph` -> `analyze`) creates a necessary chain of dependencies, but the use of interfaces (in `analyze`) helps manage external service coupling.

## Dependency Graph

```mermaid
graph TD
    subgraph Core Application
        A[main]
        B(config)
        C(scanner)
        D(cache)
        E(graph)
        F(analyze)
        G(render)
    end

    subgraph External Libraries
        H[tree-sitter/go-tree-sitter]
        I[ebitengine/purego]
        J[charmbracelet/bubbletea]
        K[go-gitignore]
        L[gopkg.in/yaml.v3]
        M[modelcontextprotocol/go-sdk]
    end

    subgraph External Services
        N(LLM Clients: OpenAI, Gemini, Anthropic, Ollama)
    end

    % Internal Dependencies
    A --> B
    A --> C
    A --> D
    A --> E
    A --> F
    A --> G

    C --> B
    E --> C
    E --> D
    F --> B
    F --> D
    F --> E
    G --> C
    G --> E

    % External Dependencies
    C --> H
    C --> I
    C --> K
    B --> L
    G --> J
    F --> M
    F --> N
```

## Potential Dependency Issues

1.  **Tree-sitter C Bindings Complexity:** The `scanner` package's reliance on `github.com/ebitengine/purego` and the extensive, manually managed C grammar repositories (`/scanner/.grammar-build/`) introduces significant complexity and platform-specific build dependencies. This is a necessary evil for high-performance parsing but is the most fragile part of the dependency chain.
2.  **Tight Coupling in RAG Pipeline:** The `analyze` package is tightly coupled to the `graph` package's internal structure for RAG operations. While logical, changes to the `graph.Store` interface or data model will directly impact all LLM-related features.
3.  **LLM Client Proliferation:** The `analyze` package contains separate implementations for Anthropic, Gemini, OpenAI, and Ollama. While abstracted by a factory, maintaining four distinct API clients increases the surface area for external dependency management (e.g., different SDKs, authentication methods). The use of the MCP SDK helps mitigate this by standardizing the *output* format.
4.  **TUI Dependency Overhead:** The `charmbracelet/bubbletea` and its ecosystem bring in a large number of indirect dependencies. While providing a great user experience, this adds significant bulk to the final binary for features like `--skyline` that are not core to the analysis logic.