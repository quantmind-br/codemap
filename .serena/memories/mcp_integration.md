# MCP Integration Guide

## Overview
codemap provides a Model Context Protocol (MCP) server that allows Claude Desktop and other MCP-compliant clients to interact with the local filesystem and analyze codebases.

## Installation

### 1. Build the Server
```bash
go build -o codemap-mcp ./mcp/
```

### 2. Configure Claude Desktop
Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

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

## Available Tools

| Tool | Description | Input Parameters |
|------|-------------|------------------|
| `status` | Verify connection and filesystem access | None |
| `list_projects` | Discover projects in a directory | `path` (string), `pattern` (optional string) |
| `get_structure` | Project tree view with file sizes | `path` (string) |
| `get_dependencies` | Dependency flow analysis | `path` (string), `detail` (0-2), `mode` ("api" for compact view) |
| `get_diff` | Changed files vs branch | `path` (string), `ref` (default "main") |
| `find_file` | Find files by name pattern | `path` (string), `pattern` (string) |
| `get_importers` | Find what imports a file | `path` (string), `file` (relative path) |
| `get_symbol` | Search for symbols by name | `path` (string), `name` (string), `kind` (optional: "function"/"type"), `file` (optional filter) |

## Development
- Source code: `mcp/main.go`
- The server uses `github.com/modelcontextprotocol/go-sdk`
- To test changes: `go build -o codemap-mcp ./mcp/` and restart Claude Desktop
- The server runs on stdio transport
- Output is captured and ANSI codes are stripped to ensure clean JSON responses
