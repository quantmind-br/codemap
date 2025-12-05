# Project Overview

## Purpose
codemap is a Go CLI tool that generates token-efficient codebase visualizations for LLMs. It creates "brain maps" - compact, structured representations of project architecture that AI assistants can instantly understand without burning tokens on exploration. It now includes a knowledge graph, LLM-powered analysis, and semantic search capabilities.

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
5. **Index Mode** (`--index`) - Build knowledge graph (.codemap/graph.gob)
6. **Query Mode** (`--query`) - Query the knowledge graph for call paths
7. **Explain Mode** (`--explain`) - LLM-powered symbol explanation
8. **Summarize Mode** (`--summarize`) - LLM-powered module/directory summaries
9. **Embed Mode** (`--embed`) - Generate vector embeddings for semantic search
10. **Search Mode** (`--search`) - Natural language semantic search using RRF fusion
11. **MCP Server** (`mcp/`) - 13 tools for Claude Desktop integration

## Architecture

```
main.go           # CLI entry point, flag parsing
├── scanner/      # File traversal & analysis
│   ├── walker.go      # ScanFiles(), ScanForDeps()
│   ├── deps.go        # Tree-sitter import/function extraction
│   ├── symbol.go      # Symbol search across codebase
│   ├── calls.go       # Call graph extraction
│   ├── git.go         # Git diff operations
│   ├── grammar.go     # Grammar loading logic
│   └── queries/       # Tree-sitter .scm queries per language
├── graph/        # Knowledge graph (NEW)
│   ├── types.go       # CodeGraph, Node, Edge, NodeKind, EdgeKind
│   ├── builder.go     # Builder pattern for graph construction
│   ├── store.go       # Binary persistence (.codemap/graph.gob)
│   ├── query.go       # Path finding, tree traversal, stats
│   └── vectors.go     # Vector index for embeddings
├── analyze/      # LLM integration (NEW)
│   ├── client.go      # LLMClient interface
│   ├── factory.go     # NewClient() factory
│   ├── openai.go      # OpenAI adapter
│   ├── anthropic.go   # Anthropic adapter
│   ├── ollama.go      # Ollama adapter
│   ├── embed.go       # Embedding generation
│   ├── prompts.go     # Prompt templates
│   ├── retriever.go   # RAG context retrieval
│   └── tokens.go      # Token counting/management
├── cache/        # LLM response caching (NEW)
│   └── cache.go       # Content-hash based file cache
├── config/       # Configuration management (NEW)
│   └── config.go      # YAML config + env vars for LLM settings
├── render/       # Output generation
│   ├── tree.go        # Tree() - main file tree view
│   ├── depgraph.go    # Depgraph() - dependency flow
│   ├── skyline.go     # Skyline visualization
│   └── colors.go      # Terminal colors per language
└── mcp/          # MCP server for Claude integration
    └── main.go        # 13 tools (see mcp_integration memory)
```

## Supported Languages (for --deps and --index)
Go, Python, JavaScript, TypeScript, Rust, Ruby, C, C++, Java, Swift, Kotlin, C#, PHP, Dart, R, Bash

## Grammar Requirements
Tree-sitter grammars must be available for `--deps` and `--index` modes:
- Pre-built in `scanner/grammars/` directory
- Or via `CODEMAP_GRAMMAR_DIR` environment variable
- Built locally with `make grammars` (requires clang/gcc)

## Data Files
- `.codemap/graph.gob` - Binary knowledge graph (built with `--index`)
- `.codemap/vectors.gob` - Vector embeddings (built with `--embed`)
- `~/.config/codemap/cache/` - LLM response cache
- `~/.config/codemap/config.yaml` - User configuration
