# MCP Integration Guide

## Overview
codemap provides a Model Context Protocol (MCP) server that allows Claude Desktop and other MCP-compliant clients to interact with the local filesystem and analyze codebases. Version 2.3.0 includes 13 tools.

## Installation

### 1. Build the Server
```bash
go build -o codemap-mcp ./mcp/
```

### 2. Configure Claude Desktop
Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `~/.config/Claude/claude_desktop_config.json` (Linux):

```json
{
  "mcpServers": {
    "codemap": {
      "command": "/absolute/path/to/codemap/codemap-mcp",
      "args": []
    }
  }
}
```

## Available Tools (13 total)

### Basic Tools
| Tool | Description | Input Parameters |
|------|-------------|------------------|
| `status` | Verify connection and filesystem access | None |
| `list_projects` | Discover projects in a directory | `path` (string), `pattern` (optional string) |
| `get_structure` | Project tree view with file sizes | `path` (string) |
| `find_file` | Find files by name pattern | `path` (string), `pattern` (string) |

### Dependency Analysis
| Tool | Description | Input Parameters |
|------|-------------|------------------|
| `get_dependencies` | Dependency flow analysis | `path`, `detail` (0-2), `mode` ("api" for compact) |
| `get_diff` | Changed files vs branch | `path`, `ref` (default "main") |
| `get_importers` | Find what imports a file | `path`, `file` (relative path) |
| `get_symbol` | Search for symbols by name | `path`, `name`, `kind` (optional), `file` (optional) |

### Graph Query Tools (require `--index` first)
| Tool | Description | Input Parameters |
|------|-------------|------------------|
| `trace_path` | Find path between two symbols | `path`, `from`, `to`, `depth` (optional) |
| `get_callers` | Find what calls a symbol | `path`, `symbol`, `depth` (optional) |
| `get_callees` | Find what a symbol calls | `path`, `symbol`, `depth` (optional) |

### LLM-Powered Tools (require config + optionally `--embed`)
| Tool | Description | Input Parameters |
|------|-------------|------------------|
| `explain_symbol` | LLM explanation of a symbol | `path`, `symbol`, `model` (optional), `no_cache` |
| `summarize_module` | LLM summary of a module/dir | `path`, `module`, `model` (optional), `no_cache` |
| `semantic_search` | Natural language code search | `path`, `query`, `limit` (optional), `expand` (optional) |

## Development
- Source code: `mcp/main.go`
- The server uses `github.com/modelcontextprotocol/go-sdk`
- To test changes: `go build -o codemap-mcp ./mcp/` and restart Claude Desktop
- The server runs on stdio transport
- Output is captured and ANSI codes are stripped to ensure clean JSON responses

## Prerequisites for Advanced Tools
1. **Graph tools** (`trace_path`, `get_callers`, `get_callees`): Run `codemap --index .` first
2. **LLM tools** (`explain_symbol`, `summarize_module`): Configure LLM provider in `~/.config/codemap/config.yaml`
3. **Semantic search**: Run `codemap --embed .` to generate vector embeddings
