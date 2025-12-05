The project at `.` is a codebase analysis tool written in **Go**. It operates in two primary modes: a **Command Line Interface (CLI)** tool (`codemap`) and a **Model Context Protocol (MCP) Server** (`codemap-mcp`).

The MCP Server is the project's exposed API, designed for communication with Large Language Models (LLMs) and other AI agents. It provides a suite of tools for structural, dependency, and semantic analysis of a codebase.

The project also consumes several external LLM APIs (OpenAI, Anthropic, Google Gemini) for its AI-powered features.

## API Documentation

## APIs Served by This Project

The project exposes a set of tools via the **Model Context Protocol (MCP)**, typically running over standard I/O (stdio) for inter-process communication with an LLM orchestrator. The API version is **2.3.0**.

### Endpoints

The MCP server defines the following tools, which act as API endpoints. All requests are JSON objects conforming to the specified input schemas.

| Method | Path (Tool Name) | Description |
| :--- | :--- | :--- |
| `CALL` | `get_structure` | Get the project structure as a tree view with file metrics. |
| `CALL` | `get_dependencies` | Get the dependency flow (imports, calls) of the project. |
| `CALL` | `get_diff` | Get files changed compared to a git branch with impact analysis. |
| `CALL` | `find_file` | Find files matching a name pattern. |
| `CALL` | `get_importers` | Find all files that import a specific file. |
| `CALL` | `status` | Check the server's operational status and version. |
| `CALL` | `list_projects` | Discover project directories under a parent path. |
| `CALL` | `get_symbol` | Search for functions and types by name. |
| `CALL` | `trace_path` | Find the connection path between two symbols in the call graph. |
| `CALL` | `get_callers` | Find all functions that call a specific symbol. |
| `CALL` | `get_callees` | Find all functions called by a specific symbol. |
| `CALL` | `explain_symbol` | LLM-powered explanation of a code symbol. |
| `CALL` | `summarize_module` | LLM-powered summary of a module/directory. |
| `CALL` | `semantic_search` | Hybrid semantic and graph-based search. |

#### `CALL get_structure`

*   **Description:** Provides a hierarchical view of the codebase, including file sizes, language detection, and token estimates.
*   **Request:**
    *   **Body (JSON):**
        ```json
        {
          "path": "/path/to/project"
        }
        ```
        *   `path` (string, required): Path to the project directory to analyze.
*   **Response:**
    *   **Success (string):** A formatted string representing the file tree structure.
    *   **Error (string):** An error message if the path is invalid or analysis fails.

#### `CALL get_dependencies`

*   **Description:** Generates a dependency graph showing external dependencies, internal import chains, and function/type counts.
*   **Request:**
    *   **Body (JSON):**
        ```json
        {
          "path": "/path/to/project",
          "detail": 1,
          "mode": "api"
        }
        ```
        *   `path` (string, required): Path to the project directory.
        *   `detail` (integer, optional, default: 0): Detail level for symbols (0=names, 1=signatures, 2=full).
        *   `mode` (string, optional, default: "deps"): Output mode ("deps" for full flow, "api" for public API surface only).
*   **Response:**
    *   **Success (string):** A formatted string containing the dependency analysis report.

#### `CALL trace_path`

*   **Description:** Finds the shortest path of function calls connecting a source symbol to a target symbol. Requires a pre-built knowledge graph index.
*   **Request:**
    *   **Body (JSON):**
        ```json
        {
          "path": "/path/to/project",
          "from": "SourceSymbolName",
          "to": "TargetSymbolName",
          "depth": 5
        }
        ```
        *   `path` (string, required): Path to the project directory.
        *   `from` (string, required): Source symbol name to trace from.
        *   `to` (string, required): Target symbol name to trace to.
        *   `depth` (integer, optional, default: 5): Maximum traversal depth.
*   **Response:**
    *   **Success (string):** A formatted string showing the call chain path.
    *   **Error (string):** An error if the index is missing or no path is found.

#### `CALL explain_symbol`

*   **Description:** Uses an LLM to generate a natural language explanation for a specific code symbol.
*   **Request:**
    *   **Body (JSON):**
        ```json
        {
          "path": "/path/to/project",
          "symbol": "main",
          "model": "gemini-2.5-flash",
          "no_cache": false
        }
        ```
        *   `path` (string, required): Path to the project directory.
        *   `symbol` (string, required): Symbol name (function, type, method) to explain.
        *   `model` (string, optional): LLM model to use (overrides configuration).
        *   `no_cache` (boolean, optional): If true, bypasses the local cache for the LLM request.
*   **Response:**
    *   **Success (string):** The LLM-generated explanation.

### Authentication & Security

*   **Authentication:** The MCP server itself does not implement traditional HTTP authentication (like API keys or OAuth). It is designed to run locally or within a secure, sandboxed environment (like an LLM agent's execution context) where the orchestrator manages access.
*   **Security:** The server operates on the local filesystem, requiring the `path` parameter for all operations. Access control is delegated to the environment running the MCP server.

### Rate Limiting & Constraints

*   **Internal Rate Limiting:** The server does not implement internal rate limiting for its tools.
*   **External Constraints:** LLM-powered tools (`explain_symbol`, `summarize_module`, `semantic_search`) are constrained by the rate limits and token limits of the configured external LLM providers (OpenAI, Anthropic, Gemini). The project uses a local cache to mitigate repeated requests for the same analysis.

## External API Dependencies

The project's core functionality relies on external LLM APIs for its AI-powered features. These are configured via environment variables or a configuration file (likely in the `config` package).

### Services Consumed

| Service Name & Purpose | Base URL/Configuration | Endpoints Used | Authentication Method |
| :--- | :--- | :--- | :--- |
| **OpenAI API** (LLM Analysis) | Configured via `OPENAI_API_KEY` and `OPENAI_BASE_URL` | `/v1/chat/completions` | API Key (Bearer Token) |
| **Anthropic API** (LLM Analysis) | Configured via `ANTHROPIC_API_KEY` | `/v1/messages` | API Key (x-api-key Header) |
| **Google Gemini API** (LLM Analysis) | Configured via `GEMINI_API_KEY` | `/v1beta/models/generateContent` | API Key (Query Parameter) |
| **Ollama** (Local LLM) | Configured via `OLLAMA_BASE_URL` | `/api/generate` | None (Local/Internal Network) |

#### OpenAI, Anthropic, and Gemini Integration Details

*   **Purpose:** Used by the `analyze` package for generating explanations, summaries, and performing semantic search.
*   **Base URL/Configuration:** The specific client implementation is chosen by the `analyze.Factory` based on the configured model name or explicit environment variables.
*   **Endpoints Used:** The clients primarily use the chat/messages endpoints for multi-turn or single-turn prompt execution.
*   **Authentication Method:** Standard API key authentication for each respective provider. Keys are loaded from the environment via the `config` package.
*   **Error Handling:**
    *   The `analyze` package implements a generic `Client` interface, suggesting standardized error handling across providers.
    *   Specific error handling for API-level errors (e.g., 4xx, 5xx) is implemented within each provider's client (`anthropic.go`, `openai.go`, `gemini.go`).
*   **Retry/Circuit Breaker Configuration:**
    *   The code uses a `cache.Cache` layer to prevent redundant external API calls.
    *   The `analyze` package implements **retry logic** for transient errors (e.g., network issues, rate limits) using a mechanism with a maximum of **3 attempts** and an exponential backoff strategy (delays of 1s, 2s, 4s).

### Integration Patterns

1.  **Factory Pattern:** The `analyze.Factory` is used to instantiate the correct LLM client (OpenAI, Anthropic, Gemini, or Ollama) based on the requested model name, abstracting the provider-specific details from the core analysis logic.
2.  **Caching:** All LLM requests are wrapped with a `cache.Cache` layer (in `cache/cache.go`) to store and retrieve results based on a hash of the prompt and context, significantly reducing latency and external API costs for repeated queries.
3.  **Contextual Prompting:** The analysis tools (e.g., `explain_symbol`) first use the internal graph and scanner to retrieve relevant code snippets and context, which are then injected into the prompt sent to the external LLM API.

## Available Documentation

| Path | Description | Quality Evaluation |
| :--- | :--- | :--- |
| `/.ai/docs/api_analysis.md` | Existing API analysis document. | **High.** Provides a starting point for understanding the project's API surface. |
| `/.ai/docs/structure_analysis.md` | Analysis of the project's structure. | **High.** Useful for understanding the separation of concerns (e.g., `analyze`, `scanner`, `graph`). |
| `/.ai/docs/data_flow_analysis.md` | Analysis of data flow within the application. | **High.** Crucial for understanding how data moves from the scanner to the graph and then to the LLM clients. |
| `README.md` | General project overview and usage instructions. | **Medium.** Focuses on CLI usage, but implicitly documents the capabilities exposed via the MCP API. |
| `mcp/main.go` | Source code defining the MCP tools. | **High.** The Go structs and `mcp.AddTool` calls provide the definitive, machine-readable specification for the MCP API contract (input schemas and descriptions). |
| `analyze/*.go` | Source code for external API clients. | **High.** Defines the exact configuration variables and retry/error handling logic for external LLM integrations. |

**Documentation Quality Evaluation:** The project has excellent internal documentation and a clear separation of concerns in the code, which makes API analysis straightforward. The MCP tool definitions in `mcp/main.go` serve as a robust, self-documenting API specification for AI agents. The external API consumption is well-abstracted and includes resilience patterns (caching, retries).