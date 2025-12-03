# Task Completion Checklist

## Before Committing Changes

### Required
- [ ] Run `go fmt ./...` to format code
- [ ] Build succeeds: `go build -o codemap .`
- [ ] Test changes manually with `./codemap .` on a real project

### If Modifying --deps Mode
- [ ] Build MCP server: `go build -o codemap-mcp ./mcp/`
- [ ] Test with grammars available: `./codemap --deps .`
- [ ] Verify JSON output: `./codemap --deps --json .`

### If Adding a New Language
- [ ] Grammar added to `.github/workflows/release.yml`
- [ ] Query file created in `scanner/queries/<lang>.scm`
- [ ] Extension mapping added in `scanner/grammar.go`

### If Modifying MCP Server
- [ ] Build: `go build -o codemap-mcp ./mcp/`
- [ ] All 8 tools still work (status, list_projects, get_structure, get_dependencies, get_diff, find_file, get_importers, get_symbol)

## Testing Modes
```bash
./codemap .                    # Basic tree (shows token counts)
./codemap --deps .             # Dependency mode (shows line numbers)
./codemap --deps --api .       # API surface view (compact)
./codemap --deps --detail 1 .  # Dependencies with signatures
./codemap --diff               # Diff mode
./codemap --skyline .          # Skyline mode
./codemap --skyline --animate  # Animated skyline
```

## Note
This project currently has no automated tests (`go test ./...` shows no test files). Manual testing against real projects is the primary verification method.
