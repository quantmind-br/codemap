# AGENTS.md: Universal AI Agent Configuration

## Project Summary
`codemap` is a Go CLI tool that transforms a codebase into a persistent **Knowledge Graph** using `tree-sitter` for structural analysis. It uses **Retrieval-Augmented Generation (RAG)** via the `/analyze` package to provide context-aware analysis and semantic search using various LLMs (OpenAI, Anthropic, Gemini, Ollama).

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
4.  **Dependency View:** `./codemap --deps .` (Structural dependency map)

## Architecture Overview
The system is a Layered Pipeline centered on the `graph` package.

*   **Scanner (`/scanner`):** Ingestion layer. Uses `tree-sitter` and `purego` for AST parsing and data extraction.
*   **Graph (`/graph`):** Central data repository. Implements the Repository Pattern for persistence (Gob/Gzip) and manages vector embeddings.
*   **Analyze (`/analyze`):** Intelligence layer. Uses Factory/Adapter patterns to abstract LLM clients and implements the RAG pipeline.
*   **Render (`/render`):** Presentation layer. Handles TUI and visualization using `charmbracelet/bubbletea`.

## Key Conventions and Patterns

### Go Style and Error Handling
*   **Language:** Go (Golang). Adhere to standard Go idioms.
*   **Errors:** Critical errors must print to `os.Stderr` and exit with `os.Exit(1)`.
*   **Dependencies:** Dependencies are manually wired in `main.go` (Constructor Injection).

### Data Persistence
*   **Graph File:** The knowledge graph is stored in `.codemap/graph.gob`.
*   **Format:** Uses `encoding/gob` for serialization, typically wrapped in `gzip`.
*   **IDs:** Graph nodes use deterministic IDs generated from file path and symbol name.

### LLM Integration
*   **Pattern:** The `analyze` package uses the **Factory Pattern** to instantiate the correct `analyze.Client` based on configuration.
*   **Authentication:** API keys (e.g., `OPENAI_API_KEY`) are loaded primarily from **environment variables**.
*   **Caching:** LLM responses are cached locally. Use `--no-cache` to force a fresh API call.

### Git Workflow
*   **Commits:** Use clear, imperative commit messages (e.g., `feat: Add Gemini client support`, `fix: Handle missing grammar error`).
*   **PRs:** Must include verification steps for the affected CLI mode(s).