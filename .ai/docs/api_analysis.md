The project at `.` is a **Command Line Interface (CLI) tool** written in **Go (Golang)**. Its primary function is to analyze, visualize, and generate context for codebases, with a strong focus on integration with Large Language Models (LLMs).

The project **does not expose a traditional HTTP/REST API**. The primary interface for developers is the command line tool itself, which is documented below under "APIs Served by This Project" (referring to the CLI commands).

The project **consumes** several external APIs, primarily for its LLM-powered features.

# API Documentation

## APIs Served by This Project

Since this is a CLI application, the "API" consists of the command-line interface and its flags. The tool's main entry point is `main.go`, which uses the standard `flag` package for command parsing.

### Endpoints (CLI Commands)

| Method | Path | Description |
| :--- | :--- | :--- |
| `codemap [path]` | `(default)` | Generates a file tree view with token estimates and file sizes. |
| `codemap --deps [path]` | `deps` | Generates a dependency flow map (functions, types, and imports). |
| `codemap --skyline [path]` | `skyline` | Generates a city skyline visualization of the codebase. |
| `codemap --diff [path]` | `diff` | Shows only files changed compared to a specified Git reference. |
| `codemap --index [path]` | `index` | Builds the internal knowledge graph index (`.codemap/graph.gob`). |
| `codemap --query [path]` | `query` | Queries the knowledge graph for symbol relationships. |
| `codemap --explain [path]` | `explain` | Uses an LLM to explain a specific symbol. |
| `codemap --summarize [path]` | `summarize` | Uses an LLM to summarize a module or directory. |
| `codemap --search [path]` | `search` | Performs semantic search on the codebase using natural language. |
| `codemap --embed [path]` | `embed` | Generates vector embeddings for the knowledge graph. |

#### Endpoint Details

**1. `codemap --deps` (Dependency Graph Mode)**

*   **Method and Path:** `codemap --deps [path]`
*   **Description:** Scans the codebase using Tree-sitter grammars to build a dependency graph of functions, types, and imports.
*   **Request:**
    *   **Params:**
        *   `--detail <level>` (int): Detail level for symbols (0=names, 1=signatures, 2=full).
        *   `--api` (bool): If true, filters the output to show only the public API surface (compact view).
        *   `--json` (bool): If true, outputs the result as JSON.
*   **Response (Success):**
    *   **Format (Default):** Rendered dependency graph in the terminal.
    *   **Format (`--json`):** A JSON object of type `scanner.DepsProject` containing `Root`, `Mode`, `Files` (list of `scanner.FileAnalysis`), `ExternalDeps`, `DiffRef`, and `DetailLevel`.
*   **Authentication:** None (local file system access).

**2. `codemap --index` (Knowledge Graph Indexing)**

*   **Method and Path:** `codemap --index [path]`
*   **Description:** Parses the entire codebase to build a serialized knowledge graph (`graph.CodeGraph`) stored at `.codemap/graph.gob`.
*   **Request:**
    *   **Params:**
        *   `--force` (bool): Forces a full rebuild, bypassing incremental update checks.
        *   `--output <path>` (string): Custom path for the graph file.
        *   `--json` (bool): Outputs indexing status as JSON.
*   **Response (Success):**
    *   **Format (Default):** Console output confirming index status (up-to-date, incremental update, or full rebuild) and statistics (nodes, edges).
    *   **Format (`--json`):** A JSON object with `status`, `path`, `nodes`, `edges`, and `indexed_at`.
*   **Authentication:** None.

**3. `codemap --query` (Knowledge Graph Query)**

*   **Method and Path:** `codemap --query [path]`
*   **Description:** Traverses the knowledge graph to find relationships between symbols.
*   **Request:**
    *   **Params:**
        *   `--from <symbol>` (string): Starting symbol for the trace (outgoing edges).
        *   `--to <symbol>` (string): Target symbol for pathfinding (incoming edges).
        *   `--depth <n>` (int): Maximum traversal depth (default: 5).
        *   `--json` (bool): Outputs results as JSON.
*   **Response (Success):**
    *   **Format (Default):** Rendered graph traversal path in the terminal.
    *   **Format (`--json`):** A JSON object representing the query results (e.g., list of paths/nodes).
*   **Authentication:** None.

**4. `codemap --explain` (LLM Symbol Explanation)**

*   **Method and Path:** `codemap --explain [path]`
*   **Description:** Uses a configured LLM to generate an explanation for a specific code symbol.
*   **Request:**
    *   **Params:**
        *   `--symbol <name>` (string, **required**): The name of the symbol to explain.
        *   `--model <name>` (string): Overrides the configured LLM model.
        *   `--no-cache` (bool): Bypasses the local cache for the LLM request.
        *   `--json` (bool): Outputs the result as JSON.
*   **Response (Success):**
    *   **Format (Default):** Markdown-formatted explanation from the LLM.
    *   **Format (`--json`):** A JSON object containing the explanation text.
*   **Authentication:** Requires configuration of API keys for external LLM services (see **External API Dependencies**).

### Authentication & Security

*   **Authentication:** The CLI tool itself requires no authentication as it operates on the local file system.
*   **Security:** Security considerations are focused on the **consumed external APIs**. API keys for services like OpenAI and Anthropic must be configured securely, typically via environment variables or a local configuration file (`config.go` suggests a configuration mechanism). The tool uses a local cache (`codemap/cache`) to reduce repeated external API calls.

### Rate Limiting & Constraints

*   **Internal:** No explicit internal rate limiting. Performance is constrained by local CPU/disk I/O and the complexity of Tree-sitter parsing.
*   **External:** Rate limiting is imposed by the consumed LLM services (OpenAI, Anthropic, etc.). The application relies on the underlying HTTP client and LLM SDKs to handle standard network errors, but no explicit retry/circuit breaker logic is immediately visible in the high-level `main.go` flow.

## External API Dependencies

The project relies heavily on external LLM providers for its advanced analysis features. The `codemap/analyze` package contains clients for these services.

### Services Consumed

| Service Name & Purpose | Base URL/Configuration | Endpoints Used | Authentication Method | Error Handling | Retry/Circuit Breaker Configuration |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **OpenAI API** (LLM Analysis) | Configured via `config.OpenAI.BaseURL` (default: `api.openai.com`) | `/v1/chat/completions`, `/v1/embeddings` | API Key (`config.OpenAI.APIKey`) | Standard SDK error handling. | Not explicitly configured in `main.go` or `config.go`. Relies on SDK defaults. |
| **Anthropic API** (LLM Analysis) | Configured via `config.Anthropic.BaseURL` (default: `api.anthropic.com`) | `/v1/messages` | API Key (`config.Anthropic.APIKey`) | Standard SDK error handling. | Not explicitly configured. Relies on SDK defaults. |
| **Ollama** (Local LLM) | Configured via `config.Ollama.BaseURL` (default: `http://localhost:11434`) | `/api/generate`, `/api/embeddings` | None (assumes local/internal network access) | Standard HTTP client error handling. | Not explicitly configured. |

#### Configuration Details (`config/config.go`)

The application uses a configuration structure (`config.Config`) to manage access to these services.

*   **API Keys:** Keys are loaded from environment variables or a configuration file.
    *   `OpenAI.APIKey`
    *   `Anthropic.APIKey`
*   **Model Selection:** The specific LLM model can be configured globally or overridden via the `--model` flag.
*   **Timeouts:** The `analyze.Client` likely uses a default HTTP client, which may have a configurable or default timeout, but this is not exposed as a top-level CLI flag.

### Integration Patterns

*   **Client Factory:** The `analyze/factory.go` file is responsible for creating the correct LLM client (`analyze.Client`) based on the configuration (`--model` flag or default settings).
*   **Caching:** The `codemap/cache` package is used to store results of expensive LLM requests (e.g., explanations, summaries) to prevent redundant API calls. This is bypassed with the `--no-cache` flag.
*   **Retrieval-Augmented Generation (RAG):** The `--search`, `--explain`, and `--summarize` modes utilize the internal knowledge graph (`codemap/graph`) and vector embeddings (`codemap/analyze/embed.go`) to retrieve relevant code context before making the external LLM API call.

## Available Documentation

| Path | Description | Quality Evaluation |
| :--- | :--- | :--- |
| `README.md` | Project overview, installation, and basic usage examples. | **High.** Provides a good starting point for users. |
| `main.go` (Help Output) | Comprehensive documentation of all CLI flags and modes. | **High.** Serves as the primary API reference for the CLI. |
| `/.ai/docs/api_analysis.md` | (Self-referential) This document, once generated. | N/A |
| `/.ai/docs/data_flow_analysis.md` | Internal analysis of data flow within the application. | **Medium.** Useful for understanding internal component interaction. |
| `/.ai/docs/dependency_analysis.md` | Internal analysis of code dependencies. | **Medium.** Useful for understanding internal component interaction. |
| `/development-docs/` | Contains several detailed plans for feature implementation (e.g., knowledge graph, LLM integration). | **High.** Excellent for understanding the *design* and *future direction* of the LLM integrations. |
| `/scanner/queries/*.scm` | Tree-sitter query files. | **Technical.** Essential for developers extending language support or understanding how code is parsed for dependency analysis. |

**Documentation Quality Evaluation:** The CLI interface is well-documented via the `--help` flag and `README.md`. The internal LLM integrations are complex but have good supporting design documentation in `/development-docs`. The lack of a formal OpenAPI/Swagger specification is expected, as there is no HTTP API.