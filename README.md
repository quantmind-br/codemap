# Codebase Analysis Tool (codemap)

## Project Overview

The `codemap` project is a sophisticated code intelligence tool built in Go. It is designed to provide deep, structural, and semantic analysis of a codebase by combining high-performance static analysis with the reasoning capabilities of Large Language Models (LLMs).

**Purpose and Main Functionality:**
The core purpose of `codemap` is to transform raw source code into a queryable **Knowledge Graph**. This graph, augmented with vector embeddings, serves as the foundation for a Retrieval-Augmented Generation (RAG) pipeline, enabling AI agents and developers to ask complex questions about the codebase structure, dependencies, and functionality.

**Key Features and Capabilities:**

*   **Multi-Language Static Analysis:** Uses Tree-sitter to parse various languages and extract symbols, types, and dependencies.
*   **Knowledge Graph Persistence:** Builds and stores a persistent graph of the codebase (Nodes, Edges, Vectors) for fast, repeated querying.
*   **LLM-Powered Insights:** Provides natural language explanations for code symbols and summaries for modules using external LLMs (OpenAI, Gemini, Anthropic, Ollama).
*   **Semantic Search:** Supports hybrid search capabilities using vector embeddings and graph traversal.
*   **Code Flow Tracing:** Can trace the shortest path of function calls between two symbols in the graph.
*   **Dual Interface:** Operates as a Command Line Interface (CLI) tool for developers and as a **Model Context Protocol (MCP) Server** for AI agents.
*   **Rich Visualization:** Offers terminal visualizations, including file trees, dependency graphs, and a code "skyline" view.

**Likely Intended Use Cases:**

*   **AI Agent Tooling:** Serving as a core tool for LLM orchestrators to gain deep, structured context about a repository.
*   **Developer Onboarding:** Quickly generating summaries and explanations for unfamiliar code modules.
*   **Automated Documentation:** Generating up-to-date structural and dependency documentation.
*   **Code Review Context:** Providing impact analysis for code changes (`--diff` mode).

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

The `codemap` application follows a clear, layered architecture centered around the Retrieval-Augmented Generation (RAG) pattern.

**High-Level Architecture Overview**
The system is structured as a pipeline: Data Acquisition -> Knowledge Layer -> Intelligence Layer -> Presentation Layer. The core state is managed by the persistent Knowledge Graph.

**Technology Stack and Frameworks**

| Category | Technology | Purpose |
| :--- | :--- | :--- |
| **Core Language** | Go | Primary development language. |
| **Static Analysis** | Tree-sitter | High-performance, multi-language parsing and AST generation. |
| **Persistence** | `encoding/gob` | Efficient serialization of the Knowledge Graph to disk. |
| **CLI/TUI** | `charmbracelet/bubbletea` | Framework for building rich, interactive terminal user interfaces. |
| **AI Integration** | Model Context Protocol (MCP) | Standardized protocol for communication with LLM orchestrators. |

**Key Design Patterns**

*   **Factory Pattern:** Used in the `analyze` package to instantiate the correct LLM client (OpenAI, Gemini, etc.) based on configuration, abstracting provider-specific details.
*   **Strategy Pattern:** Different LLM providers are interchangeable strategies conforming to a common `LLMClient` interface.
*   **Repository Pattern:** The `graph` package acts as a repository, abstracting the persistence and querying logic for the code knowledge graph.

**Component Relationships**

The diagram below illustrates the primary data and control flow between the core internal packages.

```mermaid
graph TD
    subgraph 0. Orchestration
        A[main: CLI Entry Point & Router]
    end

    subgraph 1. Presentation Layer (Output)
        R[render: Visualization & Formatting]
    end

    subgraph 2. Intelligence Layer (LLM Orchestration)
        F[analyze: LLM Clients, RAG, Embeddings]
    end

    subgraph 3. Knowledge Layer (Data Core)
        E[graph: Knowledge Graph Store & Query]
        D[cache: Persistence & Caching]
    end

    subgraph 4. Data Acquisition Layer (Input)
        C[scanner: Tree-sitter Parsing & Git]
        B[config: Configuration]
    end

    A --> B
    A --> C
    A --> D
    A --> E
    A --> F
    A --> R

    C -- Reads Settings --> B
    E -- Consumes DTOs --> C
    E -- Persists Data --> D
    F -- Loads Settings --> B
    F -- Uses Cache --> D
    F -- Queries Context --> E
    R -- Consumes Data --> C
    R -- Consumes Data --> E
```

## C4 Model Architecture

### Context Diagram (Level 1)

<details>
<summary>C4 Context Diagram: System and External Relationships</summary>

```mermaid
%% C4 Context Diagram
C4Context
    title Context Diagram for Codebase Analysis Tool

    Person(developer, "Developer/User", "Interacts via CLI for analysis and visualization.")
    System(codemap, "Codebase Analysis Tool (codemap)", "Analyzes source code, builds a knowledge graph, and provides RAG-powered insights.")
    System(llm_orchestrator, "LLM Orchestrator/Agent", "Consumes the MCP Server API for code context and analysis tools.")
    System_Ext(external_llms, "External LLM APIs", "OpenAI, Anthropic, Gemini, etc. Used for generation and embedding.")
    System_Ext(source_code, "Source Code Repository", "The codebase being analyzed (Go, Python, etc.).")

    developer --> codemap "Uses CLI to run analysis modes"
    llm_orchestrator --> codemap "Calls MCP Server API (stdio/IPC)"
    codemap --> external_llms "Calls for LLM generation and embeddings (HTTP/S)"
    codemap --> source_code "Reads and parses files (File I/O)"
```
</details>

### Container Diagram (Level 2)

<details>
<summary>C4 Container Diagram: High-Level Technical Building Blocks</summary>

```mermaid
%% C4 Container Diagram
C4Container
    title Container Diagram for Codebase Analysis Tool

    System_Boundary(codemap_system, "Codebase Analysis Tool")
        Container(cli, "CLI Application (codemap)", "Go Executable", "Handles command-line arguments, orchestrates analysis, and provides TUI output.")
        Container(mcp_server, "MCP Server", "Go Executable (stdio)", "Exposes analysis tools via the Model Context Protocol (MCP) for AI agents.")
        Container(knowledge_graph, "Knowledge Graph Store", "File System (.codemap/graph.gob)", "Persistent storage for the code graph (Nodes, Edges, Vectors).")
        Container(analysis_cache, "Analysis Cache", "File System (.codemap/cache)", "Stores cached LLM responses and analysis results.")
    System_Boundary

    System_Ext(external_llms, "External LLM APIs", "OpenAI, Anthropic, Gemini, Ollama")
    System_Ext(source_code, "Source Code Repository", "Files on disk")

    cli --> knowledge_graph "Reads/Writes Graph Data (gob)"
    cli --> analysis_cache "Reads/Writes Cached Results (JSON/Bytes)"
    cli --> source_code "Scans and Parses Code"
    cli --> external_llms "Calls LLM APIs (via analyze package)"

    mcp_server --> knowledge_graph "Reads/Writes Graph Data (gob)"
    mcp_server --> analysis_cache "Reads/Writes Cached Results (JSON/Bytes)"
    mcp_server --> source_code "Scans and Parses Code"
    mcp_server --> external_llms "Calls LLM APIs (via analyze package)"

    knowledge_graph --> source_code "Built from parsed code"
```
</details>

## Repository Structure

The repository is organized by functional package, reflecting the layered architecture.

| Directory/File | Purpose |
| :--- | :--- |
| `/main.go` | Application entry point, command-line argument parsing, and mode routing. |
| `/scanner` | Code parsing, file system traversal, Git integration, and symbol extraction using Tree-sitter. |
| `/graph` | Knowledge Graph construction, persistence, querying, and vector indexing (the RAG backend). |
| `/analyze` | LLM orchestration, client management, embedding generation, and RAG context retrieval logic. |
| `/render` | Output formatting and visualization for the terminal (trees, graphs, skyline). |
| `/config` | Centralized management of application settings and LLM configuration. |
| `/cache` | Persistent key-value store for caching expensive analysis results (LLM responses, embeddings). |
| `/.codemap` | Default directory created by the tool to store persistent state (graph, vectors, cache). |

## Dependencies and Integration

### Internal and External Service Dependencies

The application relies on several external services, primarily for its AI-powered features.

| Service/Integration Point | Internal Package | Protocol/Client | Purpose |
| :--- | :--- | :--- | :--- |
| **OpenAI API** | `analyze/openai.go` | HTTP/Go SDK | General-purpose LLM for `explain`, `summarize`, and `search`. |
| **Google Gemini API** | `analyze/gemini.go` | HTTP/Go SDK | Alternative LLM provider integration. |
| **Anthropic API** | `analyze/anthropic.go` | HTTP/Go SDK | Alternative LLM provider integration. |
| **Ollama** | `analyze/ollama.go` | HTTP/Go SDK | Integration for local/self-hosted LLMs. |
| **Model Context Protocol (MCP)** | `analyze` | `modelcontextprotocol/go-sdk` | Standardizes the RAG pipeline and context formatting for LLM calls. |
| **Git** | `scanner/git.go` | OS Command Execution (Inferred) | Used to determine changed files (`--diff` mode) and repository root. |

### Integration Patterns

*   **LLM Client Abstraction:** The `analyze` package uses a common `Client` interface and a **Factory Pattern** to abstract communication with all external LLM providers, ensuring the core RAG logic remains provider-agnostic.
*   **Resilience:** The system implements a **caching layer** (`/cache`) to prevent redundant external API calls. For transient errors (e.g., network issues, rate limits), the `analyze` package implements **retry logic** with a maximum of **3 attempts** and an exponential backoff strategy.

## API Documentation

The project exposes its analysis capabilities via the **Model Context Protocol (MCP) Server**, designed for inter-process communication with LLM orchestrators. The API version is **2.3.0**.

All requests are made as `CALL` methods with JSON payloads.

| Tool Name | Description | Key Request Parameters |
| :--- | :--- | :--- |
| `get_structure` | Provides a hierarchical file tree view of the codebase, including file sizes, language, and token estimates. | `path` (string, required) |
| `get_dependencies` | Generates a dependency graph report showing external dependencies and internal import chains. | `path` (string, required), `detail` (int, optional), `mode` (string, optional) |
| `trace_path` | Finds the shortest path of function calls connecting a source symbol to a target symbol. Requires a pre-built knowledge graph index. | `path`, `from`, `to` (strings, required), `depth` (int, optional) |
| `explain_symbol` | Uses an LLM to generate a natural language explanation for a specific code symbol (function, type, method). | `path`, `symbol` (strings, required), `model`, `no_cache` (optional) |
| `summarize_module` | Uses an LLM to generate a summary of a module or directory based on its contents. | `path` (string, required), `model`, `no_cache` (optional) |
| `semantic_search` | Performs a hybrid semantic and graph-based search across the codebase. | `path` (string, required), `query` (string, required - inferred) |
| `get_callers` | Finds all functions that call a specific symbol. | `path`, `symbol` (strings, required) |
| `status` | Checks the server's operational status and version. | None |

**Authentication & Security:**
The MCP server does not implement user authentication. It is designed to run in a secure, sandboxed environment where access control is managed by the orchestrating agent. Authentication for external LLM services is handled internally via API keys loaded from environment variables or configuration files.

## Development Notes

### Project-Specific Conventions

*   **Dependency Passing:** The project favors explicit dependency passing (passing structs and interfaces as function arguments) over global state or heavy dependency injection frameworks, which is idiomatic Go.
*   **Data Persistence:** Internal data structures (like the Knowledge Graph) are serialized using `encoding/gob` for performance, while external data (like cache entries) often use JSON.
*   **Configuration:** Configuration is centralized in the `config` package and loaded once at startup, providing a consistent state for all components.

### Testing Requirements

*   **Unit Testing:** Critical components like `scanner` (parsing logic), `graph` (querying and indexing), and `analyze` (LLM client interfaces) should have robust unit tests, particularly for error handling and data transformation.
*   **Integration Testing:** The RAG pipeline (`scanner` -> `graph` -> `analyze` -> LLM) requires integration tests to ensure context retrieval is accurate and LLM clients communicate correctly with external APIs.

### Performance Considerations

*   **Static Analysis:** The use of Tree-sitter with C bindings via `purego` is a deliberate choice to maximize parsing speed, which is the primary bottleneck for large codebases.
*   **Caching:** The `cache` layer is essential for performance and cost management, preventing redundant computation and external API calls.
*   **Graph Serialization:** Using `encoding/gob` for the graph store ensures that the knowledge base can be loaded and saved quickly, minimizing startup time for query and analysis modes.

## Known Issues and Limitations

*   **Tree-sitter C Bindings Complexity:** The reliance on `github.com/ebitengine/purego` and manually managed C grammars introduces significant build complexity and platform-specific dependencies. This is the most fragile part of the build chain.
*   **Tight Coupling in RAG Pipeline:** The `analyze` package is tightly coupled to the internal data structures of the `graph` package. Any significant refactoring of the graph model will require corresponding updates to the RAG retrieval logic.
*   **LLM Client Maintenance:** The need to maintain separate client implementations for four different LLM providers (OpenAI, Anthropic, Gemini, Ollama) increases the surface area for external dependency management and API changes.
*   **TUI Dependency Overhead:** The `charmbracelet/bubbletea` dependency, while providing a great user experience, adds significant binary size for features that are not core to the analysis logic.

## Additional Documentation

*   [Development Plans]: Contains detailed architectural decisions, such as the Graph RAG implementation plan and specific LLM integration strategies.
*   [LLM Integration Guides]: Specific documentation detailing the configuration and usage of individual LLM providers (e.g., Gemini, Claude).
*   [Model Context Protocol (MCP) Specification]: External documentation detailing the full protocol specification for AI agent integration. (Note: This link is inferred as necessary for the MCP server).
*   [CONTRIBUTING.md]: (Inferred) Guidelines for contributing to the codebase.