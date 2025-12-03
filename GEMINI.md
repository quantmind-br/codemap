# codemap: Codebase Analysis and Visualization Tool

## Project Context
**codemap** is a Go-based CLI tool designed to analyze, map, and visualize codebase structures. It bridges the gap between raw source code and high-level architectural understanding, specifically optimized for:
1.  **Developer Insight:** Providing visual maps (File Tree, Dependency Graph, City Skyline) of complex projects.
2.  **AI Context:** Generating structured, token-efficient JSON outputs to feed Large Language Models (LLMs) via the Model Context Protocol (MCP).

## Development Workflow

### Prerequisites
- **Go**: 1.24+
- **C/C++ Compiler**: Required for building Tree-sitter grammars (gcc/clang).

### Key Commands
The project uses a `Makefile` for common tasks:

| Task | Command | Description |
| :--- | :--- | :--- |
| **Build CLI** | `make build` | Compiles the `codemap` binary. |
| **Build MCP** | `make build-mcp` | Compiles the MCP server binary (`codemap-mcp`). |
| **Setup Grammars** | `make grammars` | **Required for --deps mode.** Downloads and builds Tree-sitter libraries. |
| **Run (Tree)** | `make run DIR=.` | Runs the tool in default tree mode on the current directory. |
| **Run (Deps)** | `make deps DIR=.` | Runs dependency analysis (requires grammars). |
| **Clean** | `make clean` | Removes binaries and grammar artifacts. |

### Manual Execution
```bash
# Basic file tree structure
./codemap .

# Deep dependency analysis (requires make grammars first)
./codemap --deps .

# Skyline visualization with animation
./codemap --skyline --animate .

# JSON output for LLM consumption
./codemap --json .
```

## Architecture

The application follows a strict unidirectional pipeline:
`Orchestration (main)` → `Acquisition (scanner)` → `Presentation (render)`

### Core Packages
*   **`scanner/`**: The engine room.
    *   **`walker.go`**: Handles file system traversal and Git integration (respecting `.gitignore`).
    *   **`deps.go`**: Uses **Tree-sitter** to parse code and extract symbols (functions, types) and imports.
    *   **`types.go`**: Defines the domain models (`Project`, `DepsProject`, `FileInfo`). **This is the contract between scanner and render.**
*   **`render/`**: The presentation layer.
    *   **`tree.go`**: Renders the standard file tree.
    *   **`skyline.go`**: Renders the 3D ASCII city visualization.
    *   **`depgraph.go`**: Renders the dependency flow.
*   **`mcp/`**: Model Context Protocol server implementation.
    *   Exposes analysis capabilities as tools (`get_structure`, `get_dependencies`, etc.) to LLM agents.

## Roadmap & Future Features (from PLAN.md)
The project is evolving from a structural mapper to a semantic understanding system ("GraphRAG").

*   **Knowledge Graph**: Moving to a graph-based data model (Nodes/Edges) to track calls and complex relationships.
*   **Hybrid Retrieval**: Combining structural search (graph) with semantic search (embeddings).
*   **LLM Integration**: Adding features to "explain" symbols or "trace" paths using local LLMs.

## Conventions
*   **Style**: Standard Go formatting (`go fmt`).
*   **Error Handling**: Fail fast with clear error messages to stderr.
*   **Dependencies**: Explicit dependency injection is preferred over global state.
*   **Grammars**: Tree-sitter grammars are loaded dynamically via `purego`. New languages require adding a `.scm` query file in `scanner/queries/` and updating `release.yml`.
