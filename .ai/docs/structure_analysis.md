# Code Structure Analysis
## Architectural Overview
The project, "codemap," is a sophisticated code intelligence tool built primarily in Go, designed to analyze source code by combining static analysis with large language model (LLM) capabilities through a Retrieval-Augmented Generation (RAG) architecture.

The system is organized into a clear, layered structure:

1.  **Data Acquisition Layer (`scanner`):** Responsible for parsing source code files using Tree-sitter to build Abstract Syntax Trees (ASTs) and extract raw structural data (symbols, calls, dependencies).
2.  **Knowledge Layer (`graph`):** The core data management component. It transforms raw structural data into a persistent, queryable knowledge graph, including vector embeddings for semantic search. This layer acts as the system's central memory.
3.  **Intelligence Layer (`analyze`):** Orchestrates the LLM interaction. It handles prompt engineering, context retrieval (RAG) from the Knowledge Layer, communication with various LLM providers (OpenAI, Gemini, Anthropic), and caching.
4.  **Presentation Layer (`render`):** Formats and visualizes the analysis results, providing output in various forms like dependency graphs, code structure trees, or structured API responses.

This architecture promotes modularity, allowing the static analysis engine, the knowledge graph, and the LLM integration to evolve independently.

## Core Components

| Component | Directory | Responsibility |
| :--- | :--- | :--- |
| **Scanner** | `/scanner` | Performs multi-language static analysis using Tree-sitter. Extracts symbols, function calls, dependencies, and types from source files. |
| **Graph Store** | `/graph` | Manages the code knowledge graph. Handles graph construction, persistence (`store.go`), querying, and vector indexing (`vectors.go`) for semantic retrieval. |
| **Analysis Engine** | `/analyze` | The LLM orchestration layer. Manages API clients for various LLMs, handles token counting, embedding generation, and implements the core retrieval logic. |
| **Configuration** | `/config` | Centralized management of application settings, including LLM provider selection, API keys, and operational parameters. |
| **Caching** | `/cache` | Provides a persistent key-value store to cache expensive results, such as LLM responses and generated embeddings, optimizing performance and cost. |
| **Renderer** | `/render` | Responsible for output formatting and visualization, including generating dependency graphs (`depgraph.go`) and structural views (`tree.go`, `skyline.go`). |

## Service Definitions

*   **LLM Client Service:** A set of concrete implementations (`anthropic.go`, `gemini.go`, `openai.go`, `ollama.go`) that adhere to a common interface (`client.go`). Each implementation handles the specific API communication, request formatting, and response parsing for its respective LLM provider.
*   **Retriever Service:** Defined in `/analyze/retriever.go`. This service implements the core RAG functionality. It takes a user query, uses the embedding service to vectorize it, queries the Graph Store's vector index for relevant code context, and prepares the final context for the LLM prompt.
*   **Grammar Management Service:** Implied by the `/scanner/.grammar-build` directory and `scanner/grammar.go`. This service manages the compilation and loading of Tree-sitter language parsers, ensuring the Scanner can process various languages.
*   **Graph Persistence Service:** Defined in `/graph/store.go`. Handles the serialization and deserialization of the entire code graph to and from disk (using `.gob` files), ensuring state persistence between runs.

## Interface Contracts

The architecture relies on interfaces to maintain separation of concerns, particularly in the LLM and data layers.

*   **`LLMClient` (inferred from `/analyze/client.go` and `mock.go`):**
    *   Contract for sending prompts to an LLM and receiving a response.
    *   Key methods: `Generate(prompt string) (string, error)`, `GetTokenCount(text string) (int, error)`.
*   **`LLMFactory` (inferred from `/analyze/factory.go`):**
    *   Contract for instantiating specific LLM clients based on configuration.
    *   Key method: `NewClient(provider string, config *Config) (LLMClient, error)`.
*   **`Embedder` (inferred from `/analyze/embed.go`):**
    *   Contract for generating vector embeddings for text chunks.
    *   Key method: `CreateEmbedding(text string) ([]float32, error)`.
*   **`GraphStore` (inferred from `/graph/store.go`):**
    *   Contract for managing the persistent graph data.
    *   Key methods: `Load() (*Graph, error)`, `Save(*Graph) error`, `Query(q *Query) (*Results, error)`.

## Design Patterns Identified

1.  **Factory Pattern:** Explicitly used in the `/analyze` package (`factory.go`) to decouple the application from concrete LLM client implementations. This allows easy switching or addition of new LLM providers.
2.  **Strategy Pattern:** Applied to the LLM interaction. Different LLM providers (OpenAI, Gemini, etc.) are interchangeable strategies that conform to the `LLMClient` interface.
3.  **Repository Pattern:** The `/graph` package acts as a repository for the code knowledge. It abstracts the complex data structure (the graph) and its persistence details from the consuming analysis logic.
4.  **Model-View-Controller (MVC) / Layered Architecture:** The structure loosely follows a layered pattern:
    *   **Model:** The `/graph` package (data and business logic).
    *   **Controller:** The `/analyze` package (orchestration and external service calls).
    *   **View:** The `/render` package (presentation logic).

## Component Relationships

The system operates as a pipeline:

1.  **Initialization:** `main.go` loads settings from `/config` and initializes the `Scanner` and `Graph Store`.
2.  **Data Ingestion:** The `Scanner` uses its language grammars and walkers to process source files. The extracted structural data is fed to the `Graph Builder` (`/graph/builder.go`).
3.  **Knowledge Base Creation:** The `Graph Builder` constructs the knowledge graph, and the `Embedder` (`/analyze/embed.go`) generates vectors for graph nodes, which are stored by the `Graph Store`.
4.  **Analysis Execution:** The `Analysis Engine` (`/analyze`) receives a task. It uses the `Retriever` to query the `Graph Store` (using vector search) to gather relevant code context.
5.  **LLM Interaction:** The `Analysis Engine` uses the `LLMClient` (created by the `LLMFactory`) to send the prompt and context to the external LLM, caching the result via `/cache`.
6.  **Output:** The final analysis result is passed to the `Renderer` for display.

## Key Methods & Functions

| Method/Function (Inferred) | Location | Purpose |
| :--- | :--- | :--- |
| `main.main()` | `/main.go` | Application entry point; handles command-line arguments and orchestrates the entire analysis workflow. |
| `scanner.Walk()` | `/scanner/walker.go` | Recursively traverses the AST of a file, applying Tree-sitter queries to extract structural elements (symbols, calls). |
| `graph.BuildGraph()` | `/graph/builder.go` | The core logic for transforming raw scanner output into a structured, interconnected knowledge graph. |
| `analyze.RetrieveContext()` | `/analyze/retriever.go` | Implements the RAG retrieval step: vectorizes the query and fetches the most semantically relevant code nodes from the graph. |
| `analyze.CreateEmbedding()` | `/analyze/embed.go` | Calls the embedding model API to convert text (code snippets, documentation) into high-dimensional vectors. |
| `client.Generate()` | `/analyze/client.go` | Sends the final, context-augmented prompt to the selected LLM provider for generation. |
| `render.RenderDepGraph()` | `/render/depgraph.go` | Visualizes the dependency relationships extracted from the graph, likely for TUI or graphical output. |

## Available Documentation

The repository is well-documented with internal planning and existing analysis reports, which are highly valuable for understanding the project's intent and current state.

| Document Path | Purpose | Quality Evaluation |
| :--- | :--- | :--- |
| `/.ai/docs/api_analysis.md` | Analysis of external API usage, likely focusing on LLM and embedding service endpoints. | High. Provides concrete details on external dependencies. |
| `/.ai/docs/data_flow_analysis.md` | Traces the movement of data through the system (e.g., file content -> AST -> Graph Node -> Vector -> LLM). | High. Essential for understanding the pipeline execution. |
| `/.ai/docs/structure_analysis.md` | Previous structural analysis of the codebase. | High. Useful for validating and comparing against the current architectural state. |
| `/development-docs/0003-graphrag-implementation-plan.md` | Detailed plan for implementing the Graph RAG architecture. | Excellent. Explicitly confirms the core architectural pattern and goals of the system. |
| `/development-docs/0004-gemini-integration-plan.md` | Plan for integrating the Gemini LLM. | Excellent. Confirms the use of the Factory/Strategy pattern for LLM clients. |
| `/.serena/memories/project_overview.md` | High-level summary of the project's purpose and components. | Moderate. Provides quick context for new developers or AI agents. |
| `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` | Documentation on specific LLM integrations and agent capabilities. | High. Details the functional capabilities of the Intelligence Layer. |

The documentation quality is generally high, with the development plans being particularly insightful as they articulate the "why" behind the current component structure (e.g., the commitment to a RAG-based knowledge graph).