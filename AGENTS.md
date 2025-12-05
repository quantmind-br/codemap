# AGENTS.md: Universal AI Agent Configuration

## Project Summary
`codemap` is a Go CLI tool that transforms a codebase into a persistent **Knowledge Graph** using `tree-sitter` for structural analysis. It uses **Retrieval-Augmented Generation (RAG)** via the `/analyze` package to provide context-aware analysis and semantic search using various LLMs.

## Build & Operational Commands

### Build Commands
| Action | Command | Notes |
| :--- | :--- | :--- |
| Build CLI | `go build -o codemap .` | Creates the main executable. |
| Build Grammars | `make grammars` | **CRITICAL:** Must be run once and after any change to `scanner/queries/*.scm`. |
| Format Code | `go fmt ./...` | Standard Go formatting. |

### Key Operational Modes
1.  **Index Graph:** `./codemap --index .` (Builds/updates `.codemap/graph.gob`)
2.  **LLM Explain:** `./codemap --explain --symbol <Name> .` (RAG-powered analysis)
3.  **Semantic Search:** `./codemap --search --q "query" .` (Vector search)
4.  **Trace Path:** `./codemap --query --from <SymA> --to <SymB> .` (Finds call path)

## Architecture Overview
The system is a Layered Pipeline centered on the `graph` package, utilizing interfaces for decoupling.

*   **Scanner (`/scanner`):** Data Acquisition Layer. Uses `tree-sitter` and `purego` for AST parsing.
*   **Graph (`/graph`):** Knowledge Layer. Central data repository, implementing the Repository Pattern for persistence and vector indexing.
*   **Analyze (`/analyze`):** Intelligence Layer. Abstracts LLM clients using Factory/Strategy patterns and implements the core RAG pipeline.
*   **Render (`/render`):** Presentation Layer. Handles TUI and visualization (e.g., dependency graphs).

## Key Conventions and Patterns

### Go Style and Error Handling
*   **Language:** Go (Golang). Adhere to standard Go idioms.
*   **Dependencies:** Dependencies are wired in `main.go` using **Constructor Injection**.
*   **Errors:** Critical errors must print to `os.Stderr` and exit with `os.Exit(1)`.

### Data Persistence
*   **Graph File:** The knowledge graph is stored in `.codemap/graph.gob`.
*   **Format:** Uses `encoding/gob` for efficient serialization of Go structs.
*   **IDs:** Graph nodes use deterministic IDs generated from file path and symbol name.

### LLM Integration
*   **Interface:** All LLM clients adhere to the `analyze.Client` interface.
*   **Authentication:** API keys (e.g., `ANTHROPIC_API_KEY`) are loaded primarily from **environment variables**.
*   **Resilience:** The `analyze` package implements a local file system cache and **retry logic (max 3 attempts)** for transient API errors.
*   **Tooling:** The project exposes its capabilities via the **Model Context Protocol (MCP)**, defining tools like `explain_symbol` and `trace_path`.