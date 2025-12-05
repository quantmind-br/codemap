# Code Structure Analysis
## Architectural Overview

The project, named `codemap`, is a sophisticated **Code Analysis and Knowledge Graph System** built in Go. Its architecture is primarily **Layered**, separating concerns into distinct packages for scanning, data storage, intelligence, and presentation. It also exhibits characteristics of a **Hexagonal Architecture** by isolating the core analysis logic from external services (LLMs) through clear interfaces.

The system's core function is to ingest source code, transform it into a structured knowledge graph, and leverage Large Language Models (LLMs) for deep, context-aware analysis.

**Key Architectural Layers:**

1.  **Infrastructure Layer (`/scanner`):** Responsible for parsing source code using Tree-sitter grammars, generating an Abstract Syntax Tree (AST), and extracting raw structural data (symbols, calls, dependencies).
2.  **Data Layer (`/graph`, `/cache`):** Manages the persistence and retrieval of the extracted data, forming a knowledge graph and associated vector embeddings for semantic search.
3.  **Application/Intelligence Layer (`/analyze`):** Contains the business logic for interacting with external LLM services, managing prompts, generating embeddings, and performing Retrieval-Augmented Generation (RAG).
4.  **Presentation Layer (`/render`):** Handles the visualization and formatting of the analysis results, such as dependency graphs and file trees.

## Core Components

| Component | Package | Purpose & Responsibility |
| :--- | :--- | :--- |
| **Code Scanner** | `/scanner` | The primary data ingestion component. It orchestrates the use of language-specific Tree-sitter parsers to perform deep syntactic analysis and extract structural metadata like function calls, type definitions, and dependencies. |
| **Knowledge Graph Store** | `/graph` | The central data repository. It defines the structure for the code graph (nodes, edges, properties) and manages the storage (`store.go`) and querying (`query.go`) of this data, including vector embeddings (`vectors.go`). |
| **LLM Analysis Engine** | `/analyze` | Abstracts all interactions with external AI services. It includes client implementations for various providers (OpenAI, Gemini, Anthropic) and manages the RAG pipeline via the `retriever.go` component. |
| **Configuration** | `/config` | Manages application-wide settings, environment variables, and command-line parameters, ensuring all components are initialized with the correct operational context. |
| **Output Renderer** | `/render` | Responsible for transforming complex data structures (like the knowledge graph) into human-readable or machine-consumable formats, including visual representations of dependency graphs (`depgraph.go`) and file structures (`tree.go`). |

## Service Definitions

1.  **`analyze.Client` Service:**
    *   **Definition:** An interface (implemented by `anthropic.go`, `gemini.go`, `openai.go`, `ollama.go`) that standardizes communication with different LLM providers.
    *   **Responsibility:** Sending prompts, receiving analysis results, and handling provider-specific authentication and rate limiting.
2.  **`analyze.Retriever` Service:**
    *   **Definition:** A service that bridges the LLM analysis with the Knowledge Graph.
    *   **Responsibility:** Performing vector similarity searches against the graph's embeddings to retrieve relevant code context (source chunks) that are then injected into the LLM prompt.
3.  **`graph.Builder` Service:**
    *   **Definition:** The component responsible for constructing the graph data model.
    *   **Responsibility:** Taking the raw structural data from the `/scanner` package and mapping it into the defined graph schema (nodes and relationships).
4.  **`scanner.Walker` Service:**
    *   **Definition:** The core traversal logic for the AST.
    *   **Responsibility:** Iterating over the Tree-sitter AST nodes and applying language-specific queries (`scanner/queries/*.scm`) to identify and extract symbols, calls, and dependencies.

## Interface Contracts

The use of interfaces is crucial for maintaining flexibility, particularly in the `/analyze` package.

1.  **`analyze.Client` Interface (in `analyze/client.go`):**
    *   **Contract:** Defines methods for core LLM operations, such as `GenerateResponse(prompt string, context []Source) string` and `GetTokenCount(text string) int`. This allows the application to swap LLM providers seamlessly.
2.  **`graph.Store` Interface (in `graph/store.go`):**
    *   **Contract:** Defines methods for data persistence, such as `Load()` and `Save()`, abstracting the serialization format (which appears to be `gob`).
3.  **`scanner.Grammar` Interface (in `scanner/grammar.go`):**
    *   **Contract:** Defines methods for loading and accessing the compiled Tree-sitter parsers for different languages, ensuring the scanner logic is decoupled from the specifics of each language's grammar.

## Design Patterns Identified

*   **Factory Pattern:** Explicitly used in `analyze/factory.go` to instantiate the correct `analyze.Client` implementation (e.g., `NewOpenAIClient`, `NewGeminiClient`) based on configuration, adhering to the principle of dependency inversion.
*   **Repository Pattern:** The `/graph` package, particularly `store.go`, acts as a repository, abstracting the data access logic for the knowledge graph from the business logic in `/analyze`.
*   **Adapter Pattern:** The concrete LLM client implementations (e.g., `anthropic.go`) act as adapters, translating the generic `analyze.Client` interface calls into the specific API requests required by the external LLM service.
*   **Strategy Pattern:** The different LLM clients represent different strategies for fulfilling the analysis task, selectable at runtime via the Factory.

## Component Relationships

The system operates as a pipeline:

1.  **Configuration Dependency:** `/main.go` and `/mcp/main.go` depend on `/config` for initialization parameters.
2.  **Data Flow (Scanning to Graph):** `/scanner` (Producer) -> `/graph/builder.go` (Consumer). The scanner generates the raw data structure which the graph builder processes.
3.  **Data Flow (Graph to Analysis):** `/analyze/retriever.go` depends on `/graph/vectors.go` and `/graph/query.go` to fetch context for LLM prompts.
4.  **Control Flow (Analysis):** `/analyze` depends on external LLM APIs (via the `Client` interface) and uses `/analyze/tokens.go` for cost management.
5.  **Output Flow:** `/render` depends on `/graph/query.go` to retrieve the structural data needed for visualization (e.g., dependency maps).
6.  **Infrastructure Dependency:** `/scanner` is heavily dependent on the pre-built Tree-sitter grammars located in the extensive `/.grammar-build` directory.

## Key Methods & Functions

| Component | File | Key Function/Method (Inferred) | Responsibility |
| :--- | :--- | :--- | :--- |
| **Entry Point** | `main.go` | `main()` | Initializes configuration, sets up the scanner and graph, and executes the primary analysis workflow. |
| **Scanning** | `scanner/walker.go` | `Walk(sourceCode, language)` | Initiates the AST traversal for a given file and language, driving the data extraction process. |
| **Graphing** | `graph/vectors.go` | `Embed(data)` | Calculates and stores the vector embedding for a piece of code or structural element, enabling semantic search. |
| **Analysis** | `analyze/retriever.go` | `Search(query)` | Executes a vector search to find the most semantically relevant code snippets from the knowledge graph. |
| **Analysis** | `analyze/client.go` | `CallLLM(prompt, model)` | The core method for sending a request to an external LLM and handling the response. |
| **Rendering** | `render/depgraph.go` | `Draw(graph)` | Generates a visual representation of the component dependencies stored in the knowledge graph. |

## Available Documentation Include document paths and evaluate documentation quality.

The project is exceptionally well-documented, featuring both standard project documentation and extensive internal planning/analysis documents, suggesting a high degree of architectural foresight and transparency.

| Document Path | Evaluation & Content |
| :--- | :--- |
| `/.ai/docs/` | **High Quality (Structured Analysis):** Contains five detailed analysis documents (`api_analysis.md`, `data_flow_analysis.md`, `dependency_analysis.md`, `request_flow_analysis.md`, `structure_analysis.md`). These documents provide a deep, pre-existing understanding of the system's internal mechanics and relationships, which is invaluable for any developer or AI agent. |
| `/development-docs/` | **High Quality (Strategic Planning):** Contains detailed plans for major feature implementations (e.g., `0001-enhanced-code-analysis-plan.md`, `0003-graphrag-implementation-plan.md`). This documentation explains the *intent* and *evolution* of the codebase, covering topics like token heuristics, symbol search, and GraphRAG implementation. |
| `/development-docs/plans/` | **High Quality (Feature Blueprints):** Specific plans for core features like `01_knowledge_graph.md`, `02_llm_integration.md`, and `03_hybrid_retrieval.md`. These serve as architectural blueprints for the system's most critical components. |
| `/.serena/memories/` | **Contextual (AI-Specific):** Contains internal notes (`project_overview.md`, `mcp_integration.md`) used by an AI assistant (Serena). Useful for understanding the project's high-level goals and specific integration points. |
| `/` (Root) | **Standard Project Docs:** Includes `README.md`, `CONTRIBUTING.md`, and specific LLM notes (`CLAUDE.md`, `GEMINI.md`). These cover setup, contribution guidelines, and basic usage. |
| `/scanner/queries/` | **Technical (Implementation Detail):** Contains Tree-sitter query files (`*.scm`). These are the direct specification of *what* structural elements are extracted from each supported language's AST. |

**Overall Documentation Quality:** Excellent. The presence of both strategic planning documents and detailed, structured analysis reports (in `/.ai/docs/`) provides a comprehensive view of the project's architecture, history, and current state.