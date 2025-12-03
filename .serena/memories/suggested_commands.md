# Suggested Commands

## Build Commands

```bash
# Build CLI binary
go build -o codemap .

# Build MCP server
go build -o codemap-mcp ./mcp/

# Build tree-sitter grammars (one-time, requires clang/gcc)
make grammars

# Clean artifacts
make clean
```

## Run Commands

```bash
# Basic tree view
./codemap .
./codemap /path/to/project

# With make (handles build automatically)
make run                           # Tree view of current dir
make run DIR=/path/to/project      # Tree view of specific path

# Dependency flow mode
./codemap --deps .
make run DEPS=1                    # Via make

# Diff mode (changed files vs branch)
./codemap --diff                   # vs main
./codemap --diff --ref develop     # vs develop branch

# Skyline visualization
./codemap --skyline .
./codemap --skyline --animate .
make run SKYLINE=1 ANIMATE=1

# JSON output (for programmatic use)
./codemap --json .
./codemap --deps --json .
```

## Installation Commands

```bash
# Install to ~/.local/bin (default)
make install

# Install MCP server to ~/.local/bin
make install-mcp

# Uninstall
make uninstall
```

## Development Workflow

```bash
# Format code
go fmt ./...

# Build and test locally
go build -o codemap . && ./codemap .

# Test MCP server
go build -o codemap-mcp ./mcp/

# Run with debug output
./codemap --debug .
```

## System Commands (Linux)

```bash
# Standard utilities available
git, ls, cd, grep, find, cat, head, tail
```

## Environment Variables

- `CODEMAP_GRAMMAR_DIR` - Override grammar library location for --deps mode
