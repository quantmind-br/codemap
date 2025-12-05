The `codemap` project is a data-intensive application focused on analyzing source code and building a knowledge graph, which is then used to interact with Large Language Models (LLMs). The data flow is characterized by a pipeline: **Source Code Scanning -> Graph Persistence -> LLM Interaction/Retrieval -> Output Rendering**.

# Data Flow Analysis

## Data Models Overview

The core data models define the structure of the analyzed code and the application's configuration.

| Model Name | Location | Description | Key Fields |
| :--- | :--- | :--- | :--- |
| **`Config`** | `config/config.go` | Application configuration, loaded from YAML and environment variables. | `LLM` (`LLMConfig`), `Cache` (`CacheConfig`), `Debug` (bool). |
| **`LLMConfig`** | `config/config.go` | Settings for LLM providers (OpenAI, Anthropic, Ollama). | `Provider`, `Model`, `OpenAIAPIKey`, `EmbeddingModel`, `Temperature`, `MaxTokens`. |
| **`CacheConfig`** | `config/config.go` | Settings for the local file cache. | `Enabled` (bool), `Dir` (string), `TTLDays` (int). |
| **`Symbol`** | `scanner/types.go` | Represents a code entity (function, struct, variable, etc.). The fundamental unit of analysis. | `ID` (string), `Name` (string), `Type` (string), `Kind` (string), `File` (string), `StartLine`, `EndLine`, `Docstring` (string). |
| **`Call`** | `scanner/types.go` | Represents a function call or method invocation. | `ID` (string), `CallerID` (string), `CalleeID` (string), `File` (string), `Line` (int). |
| **`Dependency`** | `scanner/types.go` | Represents a package or module import/dependency. | `ID` (string), `SourceFile` (string), `TargetPackage` (string). |
| **`Graph`** | `graph/types.go` | The main data structure for the knowledge graph, holding all extracted data. | `Symbols` (map[string]*`Symbol`), `Calls` (map[string]*`Call`), `Dependencies` (map[string]*`Dependency`), `Vectors` (map[string][]float32). |
| **`EmbeddingRequest`** | `analyze/client.go` | Data structure for requesting embeddings from an LLM provider. | `Model` (string), `Input` ([]string). |
| **`EmbeddingResponse`** | `analyze/client.go` | Data structure for receiving embeddings from an LLM provider. | `Data` ([]`EmbeddingData`), `Usage` (`Usage`). |

## Data Transformation Map

Data undergoes several transformations as it moves through the system:

| Stage | Input Data | Output Data | Transformation/Mechanism |
| :--- | :--- | :--- | :--- |
| **Configuration Loading** | YAML/Environment Variables (Text) | `Config` (Go Struct) | `yaml.Unmarshal` (Deserialization), Environment Variable Overrides, `DefaultConfig` (Initialization). |
| **Code Scanning** | Source Code (Text) | `Symbol`, `Call`, `Dependency` (Go Structs) | Tree-sitter parsing (`scanner/walker.go`), AST traversal, Query matching (`scanner/queries/*.scm`), Data extraction and normalization. |
| **Graph Building** | `Symbol`, `Call`, `Dependency` (Go Structs) | `Graph` (Go Struct) | Aggregation and indexing of individual entities into maps keyed by ID. |
| **Graph Persistence** | `Graph` (Go Struct) | `graph.gob` (Binary File) | `gob` encoding/decoding (`graph/store.go`). |
| **Embedding Generation** | `Symbol` Docstrings/Code Snippets (Text) | Vector Embeddings ([]float32) | LLM API call (`analyze/client.go`), JSON serialization/deserialization of `EmbeddingRequest`/`EmbeddingResponse`. |
| **LLM Prompting** | `Graph` data, User Query (Text) | LLM Prompt (Text) | Contextualization and formatting using templates (`analyze/prompts.go`). |
| **LLM Response Parsing** | LLM Response (JSON/Text) | Structured Analysis (Text/Go Structs) | JSON parsing or text extraction from the LLM's output. |
| **Output Rendering** | `Graph` data (Go Structs) | Console Output (Text/ASCII/Colors) | Data traversal and formatting logic in the `render` package (e.g., `render/tree.go`, `render/depgraph.go`). |

## Storage Interactions

The application uses two primary storage mechanisms: a file-based cache and a persistent graph store.

### 1. Graph Persistence (Knowledge Graph)

*   **Mechanism:** The entire knowledge graph, represented by the `graph.Graph` struct, is persisted to a single file, typically `.codemap/graph.gob`.
*   **Technology:** Go's built-in `encoding/gob` package is used for serialization and deserialization. This is a binary format optimized for Go data structures.
*   **Interaction:**
    *   **Write:** `graph.Store.Save(g *Graph)` writes the `Graph` struct to the file.
    *   **Read:** `graph.Store.Load()` reads the file and decodes it back into a `Graph` struct.
*   **Data Stored:** `Symbols`, `Calls`, `Dependencies`, and `Vectors` (embeddings).

### 2. Caching Mechanism

*   **Mechanism:** A simple file-based cache is implemented in the `cache` package. It stores LLM responses to avoid redundant API calls.
*   **Configuration:** Controlled by `config.CacheConfig` (`Enabled`, `Dir`, `TTLDays`).
*   **Data Flow:**
    1.  A cache key is generated from the LLM request (e.g., prompt, model, temperature).
    2.  The `cache.Cache` object checks for the key in its directory (`.codemap/cache` by default).
    3.  If a hit, the cached response (likely raw LLM output) is returned.
    4.  If a miss, the LLM API is called, and the response is written to a file in the cache directory before being returned.
*   **Persistence Pattern:** Write-through/Read-through caching pattern.

## Validation Mechanisms

Data validation primarily occurs at the configuration and input stages.

### 1. Configuration Validation

*   **Location:** `config.Config.Validate()` in `config/config.go`.
*   **Logic:** Checks for mandatory fields based on the selected LLM provider:
    *   If `ProviderOllama`, checks if `OllamaURL` is set.
    *   If `ProviderOpenAI`, checks if `OpenAIAPIKey` is set (or relies on environment variable override).
    *   If `ProviderAnthropic`, checks if `AnthropicAPIKey` is set (or relies on environment variable override).
    *   Ensures `Model` is set.
    *   Performs range checks on numeric fields like `Timeout` and `MaxRetries` (must be non-negative).

### 2. Data Integrity (Implicit)

*   **Scanner:** The `scanner` package relies on the robustness of the Tree-sitter parsers. Data integrity is implicitly maintained by ensuring that extracted `Symbol`, `Call`, and `Dependency` structs are correctly populated from the Abstract Syntax Tree (AST).
*   **Graph:** The `graph` package uses string IDs to maintain relationships between entities (e.g., `Call.CallerID` references a `Symbol.ID`). While there is no explicit foreign key validation, the graph structure itself enforces relationships.

## State Management Analysis

The application's state is managed through configuration, the in-memory graph, and the file-based cache.

| State Component | Storage Location | Management Strategy |
| :--- | :--- | :--- |
| **Application Settings** | `config.Config` (In-memory struct) | Loaded once at startup (`config.Load()`) from multiple sources (defaults, user config, project config, env vars) and treated as immutable for the application's runtime. |
| **Knowledge Graph** | `graph.Graph` (In-memory struct) | **Primary State.** Loaded from `.codemap/graph.gob` at the start of analysis/querying. It is updated during the scanning phase and persisted back to disk upon completion. |
| **LLM Responses** | `cache.Cache` (File system) | **Secondary State.** Managed by a TTL (Time-To-Live) mechanism defined in `CacheConfig.TTLDays`. Responses are stored as files, providing persistence across runs. |
| **LLM Client State** | `analyze.Client` (In-memory struct) | Manages transient state like rate limiting (`RequestsPerMin`) and retry logic (`MaxRetries`), ensuring controlled interaction with external APIs. |

## Serialization Processes

The project uses two main serialization formats:

### 1. YAML (Configuration)
*   **Purpose:** Storing human-readable configuration.
*   **Mechanism:** `gopkg.in/yaml.v3`.
*   **Process:** YAML text is deserialized into `config.Config` structs.

### 2. Gob (Graph Persistence)
*   **Purpose:** Efficiently storing the complex, interconnected `graph.Graph` data structure.
*   **Mechanism:** `encoding/gob`.
*   **Process:** The `Graph` struct is serialized into a binary `.gob` file for fast loading and saving.

### 3. JSON (LLM Communication)
*   **Purpose:** Standard communication format for external LLM APIs (OpenAI, Anthropic, Ollama).
*   **Mechanism:** `encoding/json`.
*   **Process:**
    *   **Outbound:** Go structs like `EmbeddingRequest` are serialized to JSON for the API request body.
    *   **Inbound:** JSON responses from the LLM are deserialized into structs like `EmbeddingResponse` or parsed for the final text output.

## Data Lifecycle Diagrams

### 1. Code Analysis and Graph Building Lifecycle

```mermaid
graph TD
    A[Source Code Files] --> B(Scanner Package);
    B --> C{Tree-sitter Parsing};
    C --> D[Extracted Entities: Symbol, Call, Dependency];
    D --> E(Graph Builder);
    E --> F[In-Memory Graph (graph.Graph)];
    F --> G{Graph Store (gob)};
    G --> H[.codemap/graph.gob];
    H --> F;
```

### 2. LLM Interaction and Caching Lifecycle

```mermaid
graph TD
    A[User Query] --> B(Analyze Package);
    B --> C{Cache Check (cache.Cache)};
    C -- Cache Hit --> D[Return Cached Response];
    C -- Cache Miss --> E(LLM Client);
    E --> F{LLM API Request (JSON)};
    F --> G[LLM Response (JSON)];
    G --> H(Cache Write);
    H --> I[Cache Directory];
    G --> J[Parse Response];
    J --> D;
```