This project, **codemap**, is a **Command Line Interface (CLI) application** written in **Go**. It does not expose a traditional network-based API (like REST, GraphQL, or gRPC). Instead, its primary "API" is its command-line interface, which provides various analysis and visualization tools for a codebase.

The project's core functionality involves consuming external APIs from various **Large Language Model (LLM) providers** (OpenAI, Anthropic, Gemini, Ollama) to perform advanced code analysis, explanation, and semantic search.

The following documentation focuses on the CLI interface as the primary served API and the LLM integrations as the external dependencies.

# API Documentation

## APIs Served by This Project

The primary interface is the command-line tool itself. The most "API-like" interactions are the LLM-powered modes, which are documented below.

### Endpoints (CLI Commands)

The CLI commands act as the functional endpoints of the application.

| Method | Path (Command) | Description |
| :--- | :--- | :--- |
| `POST` | `codemap --explain` | Explains a specific symbol (function, type, etc.) using an LLM. |
| `POST` | `codemap --summarize` | Summarizes a module or directory using an LLM. |
| `POST` | `codemap --search` | Performs a natural language semantic search over the codebase. |
| `GET` | `codemap --query` | Queries the internal knowledge graph for dependencies and paths. |
| `POST` | `codemap --index` | Builds or rebuilds the internal knowledge graph index. |
| `POST` | `codemap --embed` | Generates vector embeddings for the knowledge graph. |

#### `codemap --explain`

Explains a specific symbol in the codebase using a configured LLM.

*   **Method and Path**: `codemap --explain --symbol <name> [path]`
*   **Description**: Fetches the source code for the specified symbol, constructs a prompt, and sends it to the LLM for a detailed explanation.
*   **Request**:
    *   **Params**:
        *   `--symbol <name>` (Required): The name of the symbol (e.g., `main`, `NewClient`).
        *   `[path]` (Optional): The root directory of the codebase (defaults to `.`).
        *   `--model <name>` (Optional): Overrides the configured LLM model.
        *   `--no-cache` (Optional): Bypasses the LLM response cache.
        *   `--json` (Optional): Outputs the result in JSON format.
*   **Response (Success)**: A detailed, natural language explanation of the symbol's purpose and implementation.
*   **Response (Error)**: A message indicating the symbol was not found or an error occurred during the LLM request (e.g., API key missing, network error).
*   **Authentication**: None (CLI tool). Relies on environment variables for LLM API keys (see Authentication & Security).
*   **Examples**:
    ```bash
    codemap --explain --symbol runExplainMode .
    codemap --explain --symbol NewClient --model gemini-2.5-flash .
    ```

#### `codemap --summarize`

Summarizes the contents and purpose of a given module or directory using an LLM.

*   **Method and Path**: `codemap --summarize [path]`
*   **Description**: Gathers all relevant code from the specified path, chunks it if necessary, and requests a summary from the LLM.
*   **Request**:
    *   **Params**:
        *   `[path]` (Required): The path to the directory or file to summarize.
        *   `--model <name>` (Optional): Overrides the configured LLM model.
        *   `--no-cache` (Optional): Bypasses the LLM response cache.
        *   `--json` (Optional): Outputs the result in JSON format.
*   **Response (Success)**: A natural language summary of the module/directory.
*   **Response (Error)**: A message indicating a failure in scanning or LLM communication.
*   **Authentication**: None (CLI tool). Relies on environment variables for LLM API keys.
*   **Examples**:
    ```bash
    codemap --summarize ./analyze
    ```

#### `codemap --search`

Performs a semantic search over the codebase using vector embeddings and an LLM.

*   **Method and Path**: `codemap --search --q <query> [path]`
*   **Description**: Uses the knowledge graph's embeddings to find code snippets semantically related to the natural language query.
*   **Request**:
    *   **Params**:
        *   `--q <query>` (Required): The natural language search query (e.g., `"how do I parse the configuration file"`).
        *   `[path]` (Optional): The root directory of the codebase (defaults to `.`).
        *   `--limit <n>` (Optional): Number of results to return (default: 10).
        *   `--expand` (Optional): Includes callers/callees context for each result.
        *   `--json` (Optional): Outputs the result in JSON format.
*   **Response (Success)**: A list of relevant code symbols, their file paths, and surrounding context.
*   **Authentication**: None (CLI tool). Requires a configured embedding model (usually the same as the LLM model).
*   **Examples**:
    ```bash
    codemap --search --q "find all functions that call the OpenAI API" .
    ```

### Authentication & Security

The `codemap` CLI tool itself does not require authentication. However, its LLM-powered features rely on API keys for external services.

*   **Mechanism**: API Keys via Environment Variables.
*   **Configuration**: The application reads configuration from a `config.Config` struct, which is populated from environment variables.
    *   **OpenAI**: Requires `OPENAI_API_KEY`.
    *   **Anthropic**: Requires `ANTHROPIC_API_KEY`.
    *   **Gemini**: Requires `GEMINI_API_KEY`.
    *   **Ollama**: Does not require an API key, as it is typically self-hosted.

### Rate Limiting & Constraints

*   **Internal**: No explicit internal rate limiting is implemented in the CLI tool.
*   **External**: The tool is subject to the rate limits imposed by the external LLM providers (OpenAI, Anthropic, Gemini). Users should ensure their usage adheres to the quotas of their respective API accounts.

## External API Dependencies

The project consumes APIs from multiple Large Language Model (LLM) providers. The integration is managed through the `analyze` package, which defines a common `LLMClient` interface.

### Services Consumed

#### 1. OpenAI API

*   **Service Name & Purpose**: OpenAI. Used for general-purpose code explanation, summarization, and semantic search.
*   **Base URL/Configuration**: `https://api.openai.com/v1/`
    *   The base URL is configured within the `analyze/openai.go` client.
    *   Configuration is loaded via `config.Config`.
*   **Endpoints Used**:
    *   **`POST /v1/chat/completions`**: Used for the `Explain` and `Summarize` operations.
    *   **`POST /v1/embeddings`**: Used by the embedding client for the `Embed` operation.
*   **Authentication Method**: Bearer Token (`Authorization: Bearer <OPENAI_API_KEY>`). The key is read from the environment variable `OPENAI_API_KEY`.
*   **Error Handling**:
    *   The client handles HTTP errors and returns a structured error message.
    *   Specific error codes (e.g., 401 for invalid key, 429 for rate limit) are likely handled by the underlying Go HTTP client, but the `analyze` package primarily returns a generic `fmt.Errorf` on failure.
*   **Retry/Circuit Breaker Configuration**: No explicit retry or circuit breaker logic is visible in the client implementation. It relies on a single HTTP request.

#### 2. Anthropic API

*   **Service Name & Purpose**: Anthropic (Claude). Used for general-purpose code explanation, summarization, and semantic search.
*   **Base URL/Configuration**: `https://api.anthropic.com/v1/`
    *   The base URL is configured within the `analyze/anthropic.go` client.
    *   Configuration is loaded via `config.Config`.
*   **Endpoints Used**:
    *   **`POST /v1/messages`**: Used for the `Explain` and `Summarize` operations.
*   **Authentication Method**: API Key via a custom header (`x-api-key: <ANTHROPIC_API_KEY>`). The key is read from the environment variable `ANTHROPIC_API_KEY`.
*   **Error Handling**: Similar to the OpenAI client, it handles HTTP errors and returns a structured error.
*   **Retry/Circuit Breaker Configuration**: No explicit retry or circuit breaker logic is visible.

#### 3. Google Gemini API

*   **Service Name & Purpose**: Google Gemini. Used for general-purpose code explanation, summarization, and semantic search.
*   **Base URL/Configuration**: Uses the Google GenAI SDK, which abstracts the base URL.
    *   Configuration is loaded via `config.Config`.
*   **Endpoints Used**:
    *   The client uses the `google/generativeai` Go SDK's `GenerateContent` method.
*   **Authentication Method**: API Key, typically passed to the SDK client initialization. The key is read from the environment variable `GEMINI_API_KEY`.
*   **Error Handling**: Relies on the error handling provided by the Google GenAI SDK.
*   **Retry/Circuit Breaker Configuration**: Relies on the SDK's internal resilience mechanisms.

#### 4. Ollama API

*   **Service Name & Purpose**: Ollama. Used for local, self-hosted LLM operations.
*   **Base URL/Configuration**: Configurable base URL, defaults to `http://localhost:11434`.
    *   The base URL is configured within the `analyze/ollama.go` client and can be overridden via configuration.
*   **Endpoints Used**:
    *   **`POST /api/generate`**: Used for the `Explain` and `Summarize` operations.
    *   **`POST /api/embeddings`**: Used by the embedding client for the `Embed` operation.
*   **Authentication Method**: None (typically used in a trusted, local environment).
*   **Error Handling**: Standard HTTP error handling.
*   **Retry/Circuit Breaker Configuration**: No explicit retry or circuit breaker logic is visible.

### Integration Patterns

*   **Client Abstraction**: The project uses a common interface (`analyze.LLMClient`) to abstract different LLM providers, allowing the core logic (`runExplainMode`, `runSummarizeMode`, etc.) to be provider-agnostic.
*   **Configuration-Driven**: The specific LLM client (OpenAI, Anthropic, Gemini, or Ollama) is instantiated by an `analyze.ClientFactory` based on the model name specified in the configuration or via the `--model` flag.
*   **Caching**: A local file-based cache (`cache.Cache`) is used to store LLM responses, preventing redundant API calls for the same prompt/symbol explanation. This improves performance and reduces external API costs. The cache can be bypassed with the `--no-cache` flag.

## Available Documentation

The project includes internal documentation and development plans, but no formal, external API specification (like an OpenAPI/Swagger file) is present, which is expected for a CLI tool.

| Path | Description | Quality Evaluation |
| :--- | :--- | :--- |
| `./README.md` | Project overview and basic usage instructions for the CLI. | **Good**: Provides the entry point for understanding the tool's functionality. |
| `./.ai/docs/api_analysis.md` | Pre-existing analysis document. | **Unknown**: Could contain valuable insights into the internal API structure. |
| `./development-docs/` | Contains various development plans (`0001-enhanced-code-analysis-plan.md`, `0004-gemini-integration-plan.md`, etc.). | **High**: Excellent for understanding planned and implemented features, especially LLM integrations. |
| `./main.go` | The source code for the CLI entry point, which defines all available commands and flags. | **Definitive**: The ultimate source of truth for the CLI "API" contract. |