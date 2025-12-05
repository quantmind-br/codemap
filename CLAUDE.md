# CLAUDE.md: Claude Code Configuration

Welcome, Claude. This document provides persistent context for working with the `codemap` repository.

## Project Overview
`codemap` is a sophisticated command-line interface (CLI) tool written in Go. Its core function is to transform a codebase into a structural **Knowledge Graph** using `tree-sitter` for deep parsing. This graph is then leveraged for intelligent, context-aware analysis via **Retrieval-Augmented Generation (RAG)** with various LLMs (Anthropic, OpenAI, Ollama, Gemini). The architecture is a highly decoupled, configuration-driven pipeline, exhibiting characteristics of a Layered and Hexagonal design.

## Build & Run Commands

The project is a CLI tool. All interactions are via the main executable.

```bash
# 1. Build the main CLI executable
go build -o codemap .

# 2. Build tree-sitter grammars (CRITICAL SETUP)
# This compiles the C grammars using purego and is a necessary one-time setup.
# MUST be re-run if any file in scanner/queries/*.scm is modified.
make grammars

# 3. Common Operational Modes

# Index Mode: Builds or updates the knowledge graph index (.codemap/graph.gob)
./codemap --index .
./codemap --index --force . # Force a full rebuild

# LLM Explain Mode: Uses LLM to explain a specific symbol, retrieving context from the graph
./codemap --explain --symbol "graph.NewBuilder" --model claude-3-opus .

# LLM Search Mode: Performs semantic search using vector embeddings
./codemap --search --q "where is the LLM client initialized" .

# Dependency Mode: Generates a dependency flow map (functions, types, imports)
./codemap --deps .

# Default Mode: Generates a file tree view with token/size estimates
./codemap .

# JSON Output: Use the --json flag for machine-readable output in any mode
./codemap --deps . --json
```

## Architecture Overview

The system operates as a data processing pipeline, with clear separation of concerns across layers.

### 1. Core Components and Responsibilities
| Component (Package) | Purpose and Responsibility | Design Pattern |
| :--- | :--- | :--- |
| **`scanner`** | **Infrastructure/Parsing.** Orchestrates `tree-sitter` to generate ASTs and extract raw structural data (Symbols, Calls, Dependencies) using language-specific `.scm` queries. | Data Mapper |
| **`graph`** | **Data Layer/Repository.** The central domain layer. Manages `Node`/`Edge` creation, persistence (`.codemap/graph.gob`), and vector store management (`vectors.go`). | Repository Pattern |
| **`analyze`** | **Application/Intelligence.** Abstracts all LLM interactions. Handles prompt construction, token counting, embedding generation, and the RAG pipeline via `retriever.go`. | Strategy/Adapter/Factory Pattern |
| **`render`** | **Presentation Layer.** Formats analyzed data for user consumption (TUI via `bubbletea`, dependency graphs, skyline views). | Presentation Layer |
| **`config`** | **Configuration Root.** Loads and validates settings from YAML and environment variables, ensuring correct LLM credentials and file paths. | Configuration Root |

### 2. LLM Integration Strategy (Hexagonal Core)
The `/analyze` package isolates the core analysis logic from external LLM services.

*   **Interface Contract:** The `analyze.Client` interface (in `analyze/client.go`) defines the contract for all LLM operations (`GenerateResponse`, `GetTokenCount`).
*   **Factory Pattern:** `analyze/factory.go` instantiates the correct concrete client (`anthropic.go`, `openai.go`, `ollama.go`, `gemini.go`) based on the `LLM.Provider` configuration.
*   **RAG Flow:** The `analyze.Retriever` service performs vector similarity searches against the `graph/vectors.go` store to retrieve relevant code context, which is then injected into the LLM prompt. The logic in `/analyze/tokens.go` is crucial for managing context window limits.

## Development Conventions & Gotchas

### Go Style and Error Handling
*   **Language:** Standard Go (Golang). Ensure all code passes `go fmt` and `go vet`.
*   **Error Handling:** Errors are checked immediately. Critical failures (e.g., missing configuration, failed graph load) print a descriptive message to `os.Stderr` and exit with `os.Exit(1)`.
*   **Dependencies:** Dependencies are manually wired in `main.go` (Constructor Injection). Avoid introducing a heavy DI container.

### Data Persistence and State Management
*   **Primary State:** The `graph.Store` is the single source of truth. It is persisted to `.codemap/graph.gob` using **Gob encoding** for fast serialization of complex Go objects, often wrapped in Gzip compression.
*   **Deterministic IDs:** Graph nodes use a stable, deterministic ID generation scheme (`graph.GenerateNodeID`) based on file path and symbol name. This is crucial for incremental indexing.
*   **Caching:** LLM responses are cached in the local file system (`.codemap/cache`). When testing new prompt engineering or model changes, always use the `--no-cache` flag to force a fresh API call and bypass the `cache` package.

### Tree-sitter Gotcha (CRITICAL)
The `scanner` package relies on compiled C libraries for `tree-sitter` grammars, managed via `purego`. If you modify any grammar query file (`scanner/queries/*.scm`), you **must** run `make grammars` to recompile the shared libraries before the `scanner` package will reflect the changes. The application will fail with a "missing grammar" error if this step is skipped.

### Configuration Priority
The `config` package prioritizes settings in this order:
1. Environment Variables (e.g., `OPENAI_API_KEY`)
2. Project-level YAML (`.codemap/config.yaml`)
3. User-level YAML (`~/.config/codemap/config.yaml`)
4. Hardcoded defaults.

## Key External Dependencies
*   `github.com/tree-sitter/go-tree-sitter`: Core AST parsing engine.
*   `github.com/ebitengine/purego`: Enables C interop for Tree-sitter without Cgo.
*   `github.com/charmbracelet/bubbletea`: TUI framework used by the `/render` package.
*   `github.com/modelcontextprotocol/go-sdk`: Integration for standardized LLM interaction.