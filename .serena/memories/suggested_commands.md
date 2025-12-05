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

## Run Commands - Basic Modes

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

## Run Commands - Knowledge Graph

```bash
# Build the knowledge graph index
./codemap --index .
./codemap --index --force .        # Force full rebuild

# Query the graph
./codemap --query --from main .              # What does main call?
./codemap --query --to Scanner .             # What calls Scanner?
./codemap --query --from A --to B .          # Find path from A to B
./codemap --query --from main --depth 3 .    # Limit traversal depth
```

## Run Commands - LLM Analysis

```bash
# Generate embeddings for semantic search
./codemap --embed .
./codemap --embed --force .        # Force re-embedding

# Explain a symbol using LLM
./codemap --explain --symbol main .
./codemap --explain --symbol "NewBuilder" --model gpt-4o .
./codemap --explain --symbol main --no-cache .

# Summarize a module/directory
./codemap --summarize ./scanner
./codemap --summarize ./graph --model claude-3-5-sonnet-20241022

# Semantic search
./codemap --search --q "parse configuration" .
./codemap --search --q "error handling" --limit 20 .
./codemap --search --q "database operations" --expand .
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

# Lint code
go vet ./...

# Build and test locally
go build -o codemap . && ./codemap .

# Test MCP server
go build -o codemap-mcp ./mcp/

# Run with debug output
./codemap --debug .
```

## Configuration

### Option 1: Using .env file (recommended)

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
# Edit .env with your settings
```

Example `.env`:
```bash
CODEMAP_LLM_PROVIDER=openai
CODEMAP_LLM_MODEL=gpt-4o-mini
OPENAI_API_KEY=sk-...
```

### Option 2: YAML config file

LLM settings can also be configured in `~/.config/codemap/config.yaml`:

```yaml
llm:
  provider: openai    # or anthropic, ollama
  model: gpt-4o-mini
cache:
  enabled: true
  ttl_days: 7
```

### Configuration Priority (highest to lowest)
1. Shell environment variables
2. Project `.env` file
3. User `.env` file (`~/.config/codemap/.env`)
4. Project config (`.codemap/config.yaml`)
5. User config (`~/.config/codemap/config.yaml`)
6. Default values

## Environment Variables

- `CODEMAP_LLM_PROVIDER` - LLM provider (openai/anthropic/ollama)
- `CODEMAP_LLM_MODEL` - LLM model name
- `OPENAI_API_KEY` - OpenAI API key
- `OPENAI_BASE_URL` - OpenAI base URL (for Azure or compatible APIs)
- `ANTHROPIC_API_KEY` - Anthropic API key
- `OLLAMA_HOST` or `CODEMAP_OLLAMA_URL` - Ollama server URL
- `CODEMAP_GRAMMAR_DIR` - Override grammar library location
- `CODEMAP_DEBUG` - Enable debug output (1 or true)
