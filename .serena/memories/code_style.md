# Code Style and Conventions

## General Philosophy
- Keep it simple - this is a CLI tool, not a framework
- Minimal external dependencies

## Go Style
- Run `go fmt` before committing
- Standard Go naming conventions (camelCase for private, PascalCase for exported)
- No explicit type annotations where inference is clear
- Error handling: return errors up the call stack, print to stderr at CLI level

## File Organization
- Main entry points in root (`main.go`) and `mcp/main.go`
- Core logic organized by responsibility: `scanner/`, `render/`
- Platform-specific code uses build tags: `grammar_unix.go`, `grammar_windows.go`

## Adding New Languages
To add a new language for `--deps` mode:
1. Add grammar to `.github/workflows/release.yml` GRAMMARS env var
2. Create `scanner/queries/<lang>.scm` with tree-sitter queries
3. Add extension mapping in `scanner/grammar.go` extToLang map

## Tree-sitter Queries
Located in `scanner/queries/` with `.scm` extension:
- Capture `@function` for function definitions
- Capture `@import` for import statements
- Use tree-sitter playground to find correct node types

## Output Conventions
- Colorized terminal output using ANSI codes (see `render/colors.go`)
- JSON output available with `--json` flag for all modes
- Stderr for errors/warnings, stdout for results
