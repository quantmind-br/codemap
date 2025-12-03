# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Build CLI
go build -o codemap .

# Build MCP server
go build -o codemap-mcp ./mcp/

# Run with make
make run                          # Default tree view
make run DIR=/path DEPS=1         # Dependency mode
make run SKYLINE=1 ANIMATE=1      # Animated skyline

# Build tree-sitter grammars (one-time setup for --deps mode)
make grammars

# Development
go fmt ./...                      # Format code
go vet ./...                      # Lint code
make clean                        # Remove artifacts
make install                      # Install to ~/.local/bin
```

## Architecture

codemap is a Go CLI that generates token-efficient codebase visualizations for LLMs.

**Core Pipeline:**
1. **scanner/** - File system traversal and dependency analysis
   - `walker.go` - `ScanFiles()` walks directories respecting .gitignore, `ScanForDeps()` for dependency mode
   - `deps.go` - Tree-sitter based import/function extraction for 16 languages
   - `grammar.go` + `grammar_unix.go`/`grammar_windows.go` - Platform-specific tree-sitter loading
   - `queries/` - Tree-sitter query files (.scm) for each supported language

2. **render/** - Output visualization
   - `tree.go` - `Tree()` generates main file tree view
   - `depgraph.go` - `Depgraph()` generates dependency flow visualization
   - `skyline.go` - ASCII cityscape visualization with optional animation
   - `colors.go` - Terminal color handling per language

3. **main.go** - CLI entry point with flag parsing (`--deps`, `--diff`, `--skyline`, `--animate`, `--ref`)

4. **mcp/** - MCP server exposing 7 tools: `status`, `list_projects`, `get_structure`, `get_dependencies`, `get_diff`, `find_file`, `get_importers`

**Key Dependencies:**
- `tree-sitter/go-tree-sitter` - AST parsing for dependency analysis
- `ebitengine/purego` - Dynamic loading of tree-sitter grammars
- `charmbracelet/bubbletea` - Terminal animation for skyline mode

**Grammar Location:** Tree-sitter `.so`/`.dylib` files must exist in `scanner/grammars/` or be specified via `CODEMAP_GRAMMAR_DIR` env var.

**Testing:**
There are no automated tests. Changes must be verified manually by running `codemap` against real projects:
- `./codemap .` (Basic tree)
- `./codemap --deps .` (Dependency mode)
- `./codemap --diff` (Diff mode)
