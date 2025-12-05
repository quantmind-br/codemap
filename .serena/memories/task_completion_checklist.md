# Task Completion Checklist

## Before Committing Changes

### Required for All Changes
- [ ] Run `go fmt ./...` to format code
- [ ] Run `go vet ./...` to lint code
- [ ] Build succeeds: `go build -o codemap .`
- [ ] Test changes manually with `./codemap .` on a real project

### If Modifying --deps or --index Mode
- [ ] Test with grammars available: `./codemap --deps .`
- [ ] Verify JSON output: `./codemap --deps --json .`
- [ ] Test index build: `./codemap --index .`

### If Modifying graph/ Package
- [ ] Test index build: `./codemap --index --force .`
- [ ] Test query mode: `./codemap --query --from main .`
- [ ] Verify graph persistence (check .codemap/graph.gob exists)

### If Modifying analyze/ Package
- [ ] Test explain mode: `./codemap --explain --symbol main .`
- [ ] Test summarize mode: `./codemap --summarize ./scanner`
- [ ] Test with --no-cache flag to verify fresh API calls
- [ ] Test embed mode: `./codemap --embed .`
- [ ] Test search mode: `./codemap --search --q "test query" .`

### If Modifying cache/ Package
- [ ] Test that caching works (second request is faster)
- [ ] Test --no-cache bypasses cache
- [ ] Check cache files in ~/.config/codemap/cache/

### If Modifying config/ Package
- [ ] Test config loading from ~/.config/codemap/config.yaml
- [ ] Test environment variable overrides work
- [ ] Run `./codemap --explain --symbol main .` to verify LLM config loads

### If Adding a New Language
- [ ] Grammar added to `.github/workflows/release.yml`
- [ ] Query file created in `scanner/queries/<lang>.scm`
- [ ] Extension mapping added in `scanner/grammar.go`

### If Modifying MCP Server
- [ ] Build: `go build -o codemap-mcp ./mcp/`
- [ ] All 13 tools still work:
  - Basic: status, list_projects, get_structure, find_file
  - Deps: get_dependencies, get_diff, get_importers, get_symbol
  - Graph: trace_path, get_callers, get_callees
  - LLM: explain_symbol, summarize_module, semantic_search

## Testing Modes

```bash
# Basic modes
./codemap .                          # Tree view
./codemap --deps .                   # Dependency mode
./codemap --deps --api .             # API surface view
./codemap --deps --detail 1 .        # With signatures
./codemap --diff                     # Diff mode
./codemap --skyline .                # Skyline mode
./codemap --skyline --animate        # Animated skyline

# Graph modes
./codemap --index .                  # Build index
./codemap --query --from main .      # Query callers
./codemap --query --to Scanner .     # Query callees
./codemap --query --from A --to B .  # Trace path

# LLM modes (require config)
./codemap --explain --symbol main .  # Explain symbol
./codemap --summarize ./scanner      # Summarize module
./codemap --embed .                  # Generate embeddings
./codemap --search --q "query" .     # Semantic search
```

## Data Files to Check
- `.codemap/graph.gob` - Knowledge graph (after --index)
- `.codemap/vectors.gob` - Vector embeddings (after --embed)
- `~/.config/codemap/cache/` - LLM response cache

## Note
This project currently has no automated tests (`go test ./...` shows no test files). Manual testing against real projects is the primary verification method.
