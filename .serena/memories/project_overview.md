# Project Overview

## Purpose
codemap is a Go CLI tool that generates token-efficient codebase visualizations for LLMs. It creates "brain maps" - compact, structured representations of project architecture that AI assistants can instantly understand without burning tokens on exploration.

## Tech Stack
- **Language**: Go 1.24.0
- **Key Dependencies**:
  - `tree-sitter/go-tree-sitter` - AST parsing for dependency analysis
  - `ebitengine/purego` - Dynamic loading of tree-sitter grammar libraries
  - `charmbracelet/bubbletea` - Terminal UI for animated skyline mode
  - `sabhiram/go-gitignore` - Gitignore pattern matching
  - `modelcontextprotocol/go-sdk` - MCP server implementation

## Main Features
1. **Tree View** (default) - Hierarchical file tree with token estimation and smart flattening
2. **Dependency Flow** (`--deps`) - Import/function relationships with line numbers across 16 languages
3. **Diff Mode** (`--diff`) - Changed files with impact analysis vs a branch
4. **Skyline Mode** (`--skyline`) - ASCII cityscape visualization
5. **MCP Server** (`mcp/`) - 8 tools for Claude Desktop integration
6. **List Projects** (MCP) - Discover projects in a directory

## Architecture

```
main.go           # CLI entry point, flag parsing
├── scanner/      # File traversal & analysis
│   ├── walker.go      # ScanFiles(), ScanForDeps()
│   ├── deps.go        # Tree-sitter import/function extraction
│   ├── symbol.go      # Symbol search across codebase
│   ├── git.go         # Git diff operations
│   ├── grammar.go     # Grammar loading logic
│   └── queries/       # Tree-sitter .scm queries per language
├── render/       # Output generation
│   ├── tree.go        # Tree() - main file tree view
│   ├── depgraph.go    # Depgraph() - dependency flow
│   ├── skyline.go     # Skyline visualization
│   └── colors.go      # Terminal colors per language
└── mcp/          # MCP server for Claude integration
    └── main.go        # 8 tools: status, list_projects, get_structure, get_dependencies, get_diff, find_file, get_importers, get_symbol
```

## Supported Languages (for --deps)
Go, Python, JavaScript, TypeScript, Rust, Ruby, C, C++, Java, Swift, Kotlin, C#, PHP, Dart, R, Bash

## Grammar Requirements
Tree-sitter grammars must be available for `--deps` mode:
- Pre-built in `scanner/grammars/` directory
- Or via `CODEMAP_GRAMMAR_DIR` environment variable
- Built locally with `make grammars` (requires clang/gcc)
