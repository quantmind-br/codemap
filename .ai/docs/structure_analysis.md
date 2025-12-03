# Code Structure Analysis
## Architectural Overview
The codebase implements a command-line interface (CLI) tool, `codemap`, designed to analyze and visualize the structure of a codebase. It follows a clear **Layered Architecture** or **Pipeline Pattern**, separating concerns into three main layers:

1.  **Control/Orchestration Layer (Root/`main` package):** Handles command-line argument parsing, mode selection (tree, skyline, deps, diff), and orchestrates the flow of execution.
2.  **Data Acquisition/Analysis Layer (`scanner` package):** The core domain logic. It is responsible for file system traversal, git integration (diffing, ignore rules), and language-specific code parsing (using tree-sitter grammars) to extract structural information (functions, types, imports).
3.  **Presentation Layer (`render` package):** Responsible for formatting and outputting the analysis results in various visualization modes (terminal tree, skyline, dependency graph, or raw JSON).

The architecture is highly modular, with the `scanner` package producing well-defined data structures (`Project`, `DepsProject`) that serve as the contract for the `render` package.

## Core Components
The system is built around two primary Go packages: `scanner` and `render`.

### 1. `scanner` Package
**Purpose:** The analysis engine of the application. It abstracts away the complexities of file system interaction, git operations, and source code parsing.
**Key Files/Modules:**
*   `types.go`: Defines the canonical data structures (`FileInfo`, `FuncInfo`, `TypeInfo`, `FileAnalysis`, `Project`, `DepsProject`) that model the codebase structure. This is the core data contract.
*   `walker.go` (Inferred): Responsible for file system traversal (`ScanFiles`).
*   `git.go` (Inferred): Handles git-related operations, specifically loading `.gitignore` rules (`LoadGitignore`) and calculating file differences (`GitDiffInfo`).
*   `grammar.go` (Inferred): Manages the loading and availability of language grammars (tree-sitter) required for deep code analysis.
*   `deps.go` (Inferred): Contains the logic for dependency analysis (`ScanForDeps`), which uses the loaded grammars to parse code and extract symbols and imports.

### 2. `render` Package
**Purpose:** The visualization and output layer. It consumes the structured data from the `scanner` and presents it to the user in a readable format.
**Key Files/Modules:**
*   `tree.go`: Implements the default hierarchical file tree view (`render.Tree`).
*   `skyline.go`: Implements the "city skyline" visualization, likely mapping file size/complexity to building height (`render.Skyline`).
*   `depgraph.go`: Implements the dependency graph visualization (`render.Depgraph`).
*   `api.go`: Implements the public API surface view (`render.APIView`).
*   `colors.go`: Utility functions for terminal color formatting.

## Service Definitions
The application's capabilities are defined by the following high-level services:

| Service/Function | Package | Responsibility |
| :--- | :--- | :--- |
| `main()` | `main` | Application entry point. Parses CLI flags and delegates to the appropriate scanner and renderer functions based on the selected mode. |
| `runDepsMode()` | `main` | Specialized orchestration for dependency analysis mode (`--deps`). Ensures grammars are loaded and calls the dependency scanner and renderer. |
| `ScanFiles(root, gitignore)` | `scanner` | Traverses the file system starting at `root`, respecting `.gitignore` rules, and collects basic `FileInfo` for all relevant files. |
| `GitDiffInfo(root, ref)` | `scanner` | Interacts with Git to determine which files have changed relative to a specified reference branch (`ref`), enabling the `--diff` mode. |
| `ScanForDeps(root, gitignore, loader, detail)` | `scanner` | Performs deep code analysis using tree-sitter grammars to extract functions, types, and imports, returning a list of `FileAnalysis` objects. |
| `AnalyzeImpact(root, files)` | `scanner` | (Inferred) Calculates the structural impact of changed files, likely by analyzing dependencies or file relationships. |
| `Tree(project)` | `render` | Renders the `Project` data structure as a hierarchical file tree in the terminal. |
| `Skyline(project, animate)` | `render` | Renders the `Project` data structure as a visual skyline representation. |
| `Depgraph(depsProject)` | `render` | Renders the `DepsProject` data structure as a dependency flow map. |
| `APIView(depsProject)` | `render` | Renders a compact view of the public API surface extracted from the `DepsProject`. |

## Interface Contracts
While no explicit Go interfaces are visible in the provided files, the structural contracts are defined by the data transfer objects (DTOs) in `scanner/types.go`. These structs act as the implicit interfaces between the `scanner` and `render` packages.

| Contract (Struct) | Purpose | Used by |
| :--- | :--- | :--- |
| `FileInfo` | Basic file metadata (path, size, diff stats). | `scanner` (producer), `Project` (container). |
| `Project` | Root data model for Tree and Skyline modes. | `scanner` (producer), `render.Tree`, `render.Skyline` (consumers). |
| `FuncInfo` | Detailed information about a function or method. | `scanner` (producer), `FileAnalysis` (container). |
| `TypeInfo` | Detailed information about a type definition (struct, class, interface). | `scanner` (producer), `FileAnalysis` (container). |
| `FileAnalysis` | Aggregated structural data for a single file (functions, types, imports). | `scanner` (producer), `DepsProject` (container). |
| `DepsProject` | Root data model for Dependency Graph mode. | `scanner` (producer), `render.Depgraph`, `render.APIView` (consumers). |
| `DetailLevel` (Enum) | Defines the depth of analysis (0=None, 1=Signature, 2=Full). | `main` (input), `scanner` (logic). |

## Design Patterns Identified
*   **Command Line Interface (CLI) Pattern:** The `main` package uses the standard `flag` package to define and parse arguments, controlling the application's behavior based on user input.
*   **Data Transfer Object (DTO) Pattern:** The structs in `scanner/types.go` (`Project`, `DepsProject`, `FileInfo`, etc.) are pure data containers used to pass information between the `scanner` and `render` layers, ensuring loose coupling.
*   **Strategy Pattern (Implicit):** The application's behavior is determined by the selected mode (e.g., `--skyline`, `--deps`). The `main` function acts as the context, selecting the appropriate "strategy" function from the `render` package (`render.Tree`, `render.Skyline`, `render.Depgraph`) to execute the final output.
*   **Adapter Pattern (Implicit):** The `scanner` package uses `tree-sitter` (a C library, inferred from `scanner/.grammar-build`) to parse various languages. The Go code acts as an adapter, translating the language-agnostic AST output from tree-sitter into the application's canonical Go DTOs (`FuncInfo`, `TypeInfo`).

## Component Relationships
The flow of control and data is strictly unidirectional:

1.  **`main`** initiates the process.
2.  **`main`** calls **`scanner`** functions (`ScanFiles` or `ScanForDeps`).
3.  **`scanner`** performs analysis and returns a root data structure (`Project` or `DepsProject`).
4.  **`main`** passes the returned data structure to the appropriate **`render`** function (`Tree`, `Skyline`, `Depgraph`, or `APIView`).
5.  **`render`** outputs the final result to `os.Stdout`.

**Key Dependencies:**
*   `main` depends on `scanner` and `render`.
*   `scanner` depends on external libraries for git interaction (`GitDiffInfo`) and grammar parsing (tree-sitter, managed via `scanner/.grammar-build`).
*   `render` depends only on the data structures defined in `scanner/types.go`.

## Key Methods & Functions
| Method/Function | Location | Capability |
| :--- | :--- | :--- |
| `main.main()` | `main.go` | **Orchestration & Configuration:** Defines all CLI flags and executes the core logic based on the chosen mode. |
| `scanner.ScanForDeps()` | `scanner/deps.go` (Inferred) | **Deep Analysis:** The primary function for extracting structural code elements (symbols, types, imports) using language grammars. |
| `scanner.IsExportedName()` | `scanner/types.go` | **Language Abstraction:** Provides a normalized way to determine symbol visibility (public/private) across different language conventions (Go, Python, etc.). |
| `render.Skyline()` | `render/skyline.go` | **Visualization:** Transforms file size and complexity data into a visual, spatial representation. |
| `render.Depgraph()` | `render/depgraph.go` | **Relationship Mapping:** Presents the extracted functions, types, and imports as a flow map, highlighting dependencies. |
| `scanner.LoadGitignore()` | `scanner/git.go` (Inferred) | **Filtering:** Ensures that the analysis respects standard repository exclusion rules, crucial for performance and relevance. |

## Available Documentation
The project includes a dedicated documentation directory, indicating a commitment to internal and external documentation.

| Document Path | Evaluation |
| :--- | :--- |
| `/.ai/docs/api_analysis.md` | **High Value:** Likely details the output format and structure of the API surface analysis (`--api` mode). |
| `/.ai/docs/data_flow_analysis.md` | **High Value:** Should describe how data moves through the system, particularly between `scanner` and `render`. |
| `/.ai/docs/dependency_analysis.md` | **High Value:** Crucial for understanding the logic behind `ScanForDeps` and how dependencies are resolved. |
| `/.ai/docs/request_flow_analysis.md` | **High Value:** Likely describes the execution path from CLI input to final output. |
| `/.ai/docs/structure_analysis.md` | **High Value:** A previous structural analysis, which can be used for comparison or deeper insight. |
| `/development-docs/0001-enhanced-code-analysis-plan.md` | **High Value (Planning):** Outlines the strategy for improving the code analysis capabilities, possibly related to the `--detail` and `--api` flags. |
| `/development-docs/0002-token-heuristics-symbol-search-plan.md` | **High Value (Technical):** Details the implementation plan for token estimation (`EstimateTokens` in `scanner/types.go`) and symbol search, indicating future or current advanced features. |
| `/.serena/memories/project_overview.md` | **Contextual Value:** Provides a high-level summary of the project, useful for quick orientation. |

**Documentation Quality Assessment:** The presence of detailed, numbered development plans (`0001-`, `0002-`) and dedicated AI analysis documents (`/.ai/docs/`) suggests a high quality of internal documentation, focusing on both architectural structure and specific technical features like token estimation and dependency analysis. These documents are essential for understanding the "why" behind the current implementation and recent feature additions.