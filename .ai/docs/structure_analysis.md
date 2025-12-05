The project at `.` is a sophisticated code analysis tool written in Go. Its primary function is to scan a codebase, build a structural graph representation, and leverage Large Language Models (LLMs) for deeper, context-aware analysis. The architecture is modular, separating concerns into scanning, graph management, LLM interaction, and rendering. The heavy reliance on `tree-sitter` within the `/scanner` package is a key architectural decision, providing robust, language-agnostic parsing capabilities.

# Code Structure Analysis
## Architectural Overview
The codebase follows a layered, modular architecture, primarily centered around a **Code Graph** data model. It can be described as a **Pipeline Architecture** with distinct stages:

1.  **Configuration & Initialization:** Handled by the `/config` package and the main entry point (`main.go`).
2.  **Scanning & Parsing (Source Layer):** The `/scanner` package uses `tree-sitter` to perform Abstract Syntax Tree (AST) parsing, extracting symbols, calls, and dependencies from source code files across multiple languages.
3.  **Graph Construction (Core Layer):** The `/graph` package takes the raw data from the scanner and transforms it into a persistent, queryable graph structure (likely a Knowledge Graph).
4.  **Analysis & Intelligence (Service Layer):** The `/analyze` package integrates with various LLM providers (OpenAI, Anthropic, Ollama) to perform advanced analysis on the code graph and source code snippets, using structured prompts and token management.
5.  **Presentation (Output Layer):** The `/render` package is responsible for visualizing and outputting the analysis results in different formats, such as dependency graphs, tree views, and API summaries.

This separation ensures that the core logic (scanning and graph building) is decoupled from external services (LLMs) and presentation logic (rendering).

## Core Components
| Component | Directory | Purpose & Responsibility |
| :--- | :--- | :--- |
| **Scanner** | `/scanner` | The primary code ingestion engine. Responsible for traversing the file system, parsing source files using `tree-sitter` grammars, and extracting structural information (symbols, calls, dependencies). It abstracts the complexity of language-specific parsing. |
| **Code Graph** | `/graph` | The central data repository and domain model. It manages the persistence (`store.go`), construction (`builder.go`), and querying (`query.go`) of the code's structural map. It defines the core data types for nodes and edges (`types.go`). |
| **Analyzer** | `/analyze` | The intelligence layer. It orchestrates interactions with external LLMs, manages API clients, handles prompt engineering (`prompts.go`), and performs token counting (`tokens.go`) to analyze code snippets and graph data. |
| **Configuration** | `/config` | Manages application settings, including LLM API keys, model names, and project-specific configurations. It provides a single source of truth for runtime parameters. |
| **Renderer** | `/render` | The output formatter. It takes the processed data from the graph and analysis layers and formats it for human consumption or further processing (e.g., dependency graphs, tree structures). |
| **Cache** | `/cache` | Provides a mechanism for storing and retrieving expensive computation results, likely for parsed ASTs or graph segments, to speed up subsequent runs. |

## Service Definitions
The project defines services primarily through the organization of its packages, where each package acts as a service boundary.

| Service/Module | Key Files | Responsibility |
| :--- | :--- | :--- |
| **LLM Client Service** | `/analyze/client.go`, `/analyze/openai.go`, `/analyze/anthropic.go`, `/analyze/ollama.go` | Provides a unified interface for communicating with different LLM providers. Each file implements the specific API calls for its respective provider, abstracting the LLM interaction details from the core analysis logic. |
| **Graph Store Service** | `/graph/store.go` | Handles the serialization and deserialization of the code graph to and from persistent storage (e.g., `graph.gob`). It ensures the graph state can be saved and loaded efficiently. |
| **Grammar Management Service** | `/scanner/grammar.go` | Manages the loading and initialization of the various `tree-sitter` language grammars, which are essential for the scanner's operation. |
| **Source Retrieval Service** | `/analyze/retriever.go`, `/analyze/source.go` | Responsible for fetching and preparing source code snippets based on file paths and line numbers, often in preparation for LLM analysis. |

## Interface Contracts
While the full set of interfaces requires deeper code inspection, the structure suggests the use of interfaces for abstraction, particularly in the `/analyze` package to support multiple LLM providers.

| Potential Interface | Location | Likely Methods/Purpose |
| :--- | :--- | :--- |
| **`LLMClient`** | `/analyze/client.go` | Defines the contract for interacting with any LLM (e.g., `AnalyzeCode(prompt string) (response string, error)`). Implemented by `OpenAIClient`, `AnthropicClient`, etc. |
| **`GraphStore`** | `/graph/store.go` | Defines methods for persisting and loading the graph (e.g., `Load(path string) (*Graph, error)`, `Save(path string) error`). |
| **`CodeScanner`** | `/scanner` | Defines the contract for scanning a directory and returning a structured representation of the code (e.g., `Scan(dir string) (*ScanResult, error)`). |
| **`Renderer`** | `/render` | Defines methods for outputting data in a specific format (e.g., `RenderTree(data *Graph)`, `RenderDepGraph(data *Graph)`). |

## Design Patterns Identified
1.  **Adapter Pattern:** Clearly used in the `/analyze` package. `client.go` likely defines a common interface, and files like `openai.go` and `anthropic.go` act as adapters to fit the specific vendor APIs into this common interface.
2.  **Factory Pattern:** Suggested by `/analyze/factory.go`, which is likely responsible for creating the correct `LLMClient` implementation based on configuration (e.g., which LLM provider is selected).
3.  **Repository Pattern:** The `/graph` package, particularly `store.go` and `query.go`, acts as a repository for the code graph, abstracting the data storage and retrieval logic from the business logic.
4.  **Builder Pattern:** Suggested by `/graph/builder.go`, which is responsible for the complex, step-by-step construction of the `Graph` object from the raw data provided by the scanner.

## Component Relationships
1.  **`/config` -> All Components:** The `config` package is a dependency for almost all other packages, providing necessary runtime parameters (API keys, model names, paths).
2.  **`/scanner` -> `/graph`:** The `scanner` is a producer of raw structural data (symbols, calls, dependencies), which is consumed by the `graph/builder` to construct the core `Graph` model.
3.  **`/graph` -> `/analyze`:** The `analyze` package queries the `graph` for structural context before generating prompts for the LLMs.
4.  **`/analyze` -> External LLMs:** The `analyze` package is the sole intermediary between the application and external AI services.
5.  **`/graph` & `/analyze` -> `/render`:** The `render` package consumes the final, processed data from both the `graph` (for structural views) and potentially the `analyze` package (for LLM-generated summaries) to produce the final output.
6.  **`/scanner` -> `tree-sitter` (External Dependency):** The `scanner` package is tightly coupled with the `tree-sitter` C libraries, which are managed and built within the `/scanner/.grammar-build` directory.

## Key Methods & Functions
Based on the file names, the following methods are critical to the application's capabilities:

| Package | File | Key Function/Method (Inferred) | Capability |
| :--- | :--- | :--- | :--- |
| `/scanner` | `walker.go` | `Walk(sourceCode []byte, language string) (*ScanResult, error)` | Core function for traversing the AST and extracting structural data. |
| `/scanner` | `symbol.go` | `ExtractSymbols(node *tree_sitter.Node)` | Identifies and catalogs defined entities (functions, classes, variables) in a file. |
| `/graph` | `builder.go` | `NewGraphBuilder().Build(scanResults []*ScanResult)` | Constructs the complete, interconnected code graph from all scanned files. |
| `/graph` | `query.go` | `QueryDependencies(symbolID string)` | Retrieves the upstream and downstream dependencies for a given code entity. |
| `/analyze` | `client.go` | `Analyze(prompt string, context *Graph)` | Sends a structured prompt and relevant code context to an LLM for analysis. |
| `/analyze` | `tokens.go` | `CountTokens(text string)` | Manages token limits, crucial for cost and performance of LLM interactions. |
| `/render` | `depgraph.go` | `RenderDependencyGraph(graph *Graph)` | Generates a visual or textual representation of the system's dependency structure. |
| `/render` | `tree.go` | `RenderFileTree(graph *Graph)` | Generates a hierarchical, structural view of the codebase. |

## Available Documentation
The project includes a significant amount of internal and external documentation.

| Document Path | Evaluation |
| :--- | :--- |
| `/.ai/docs/api_analysis.md` | **High Relevance:** Likely details the LLM API interaction and data formats. |
| `/.ai/docs/data_flow_analysis.md` | **High Relevance:** Crucial for understanding how data moves between the scanner, graph, and analyzer. |
| `/.ai/docs/dependency_analysis.md` | **High Relevance:** Explains the logic behind dependency extraction in the `/scanner` and `/graph` packages. |
| `/.ai/docs/request_flow_analysis.md` | **High Relevance:** Describes the end-to-end process, especially the flow from user request to final output. |
| `/.ai/docs/structure_analysis.md` | **High Relevance:** A pre-existing structural analysis, which should be compared against this current analysis for completeness. |
| `/development-docs/0001-enhanced-code-analysis-plan.md` | **High Relevance:** Provides insight into planned or recent architectural changes and feature goals. |
| `/development-docs/plans/01_knowledge_graph.md` | **High Relevance:** Details the design and implementation strategy for the core `/graph` component. |
| `/.serena/memories/project_overview.md` | **Medium Relevance:** Provides a general summary, useful for quick context. |
| `README.md`, `CONTRIBUTING.md`, `LICENSE` | **Standard Relevance:** Provides external-facing project information. |

**Documentation Quality Assessment:** The presence of detailed, numbered development plans (`/development-docs`) and specific AI-focused analysis documents (`/.ai/docs`) suggests a high-quality, well-documented project, particularly concerning its core architectural components (graph, analysis, dependencies). The documentation is highly relevant to understanding the "what" and "why" of the codebase.