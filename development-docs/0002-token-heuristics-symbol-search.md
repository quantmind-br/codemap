# TASKS.md - Token Heuristics, `get_symbol` Tool & MCP Improvements

> **Feature:** Token estimation, semantic symbol search, and MCP enhancements
> **Created:** 2025-12-03
> **Status:** Completed

---

## Project Briefing

This implementation adds three key capabilities to codemap for better LLM support:

1. **Token Estimation**: Display estimated token counts per file with warnings for large files (>8k tokens)
2. **Symbol Search (`get_symbol`)**: New MCP tool for precise function/type lookup by name
3. **MCP Enhancements**: Add API mode to `get_dependencies` and refactor path validation

### Key Technical Changes

| Component | Current | After |
|-----------|---------|-------|
| `FileInfo` | No token count | `Tokens int` field |
| `FuncInfo`/`TypeInfo` | No line numbers | `Line int` field |
| `funcCapture`/`typeCapture` | No line tracking | `line int` field |
| MCP handlers | 7 tools | 8 tools (`get_symbol`) |

---

## Phase 0: Line Capture Fix (BLOCKER)

> **Objective:** Enable line number capture for symbols - prerequisite for `get_symbol`

### Task 0.1: Add `line` field to `funcCapture` struct
- [x] **File:** `scanner/grammar.go:256-261`
- [x] Add `line int` field to `funcCapture` struct
- [x] Verify struct compiles

### Task 0.2: Add `line` field to `typeCapture` struct
- [x] **File:** `scanner/grammar.go:301-305`
- [x] Add `line int` field to `typeCapture` struct
- [x] Verify struct compiles

### Task 0.3: Add `Line` field to `FuncInfo` struct
- [x] **File:** `scanner/types.go:38-43`
- [x] Add `Line int json:"line,omitempty"` field
- [x] Verify struct compiles

### Task 0.4: Add `Line` field to `TypeInfo` struct
- [x] **File:** `scanner/types.go:87-93`
- [x] Add `Line int json:"line,omitempty"` field
- [x] Verify struct compiles

### Task 0.5: Modify `handleFuncCapture` to capture line numbers
- [x] **File:** `scanner/grammar.go:282-298`
- [x] Add `line int` parameter to `handleFuncCapture`
- [x] Set `fc.line` when capturing `func.name` (first significant capture)
- [x] Handle line in the switch statement

### Task 0.6: Modify `handleTypeCapture` to capture line numbers
- [x] **File:** `scanner/grammar.go:323-351`
- [x] Add `line int` parameter to `handleTypeCapture`
- [x] Set `tc.line` when capturing `type.name` (first significant capture)
- [x] Handle line in the switch statement

### Task 0.7: Update `AnalyzeFile` to pass line numbers
- [x] **File:** `scanner/grammar.go:183-253`
- [x] Extract line from `capture.Node.StartPosition().Row + 1` (1-indexed)
- [x] Pass line to `handleFuncCapture` calls
- [x] Pass line to `handleTypeCapture` calls

### Task 0.8: Update `funcCapture.Build` to include Line
- [x] **File:** `scanner/grammar.go:264-279`
- [x] Set `info.Line = fc.line` in Build method
- [x] Verify FuncInfo includes line in output

### Task 0.9: Update `typeCapture.Build` to include Line
- [x] **File:** `scanner/grammar.go:308-320`
- [x] Set `info.Line = tc.line` in Build method
- [x] Verify TypeInfo includes line in output

### Task 0.10: Validate Phase 0
- [x] Run `go build -o codemap .`
- [x] Run `./codemap --deps --json . | head -50` and verify lines appear
- [x] Verify line numbers are 1-indexed and correct

---

## Phase 1: Token Estimation (P0)

> **Objective:** Add token count visibility to help LLMs manage context windows

### Task 1.1: Add token estimation constants and helper
- [x] **File:** `scanner/types.go`
- [x] Add `const CharsPerToken = 3.5`
- [x] Add `const LargeFileTokens = 8000`
- [x] Add `func EstimateTokens(size int64) int`

### Task 1.2: Add `Tokens` field to `FileInfo`
- [x] **File:** `scanner/types.go:18-25`
- [x] Add `Tokens int json:"tokens,omitempty"` field

### Task 1.3: Calculate tokens in `ScanFiles`
- [x] **File:** `scanner/walker.go`
- [x] In `ScanFiles`, set `file.Tokens = EstimateTokens(file.Size)` for each file

### Task 1.4: Update `Tree` to display tokens per file
- [x] **File:** `render/tree.go`
- [x] Modify `printTreeNode` to show `[!]` warning for files > 8k tokens
- [x] Add warning indicator `[!]` for files > 8k tokens

### Task 1.5: Update header to show total tokens
- [x] **File:** `render/tree.go:106-195`
- [x] Calculate total tokens across all files
- [x] Add `Tokens: ~Xk` to stats line in header

### Task 1.6: Validate Phase 1
- [x] Run `go build -o codemap .`
- [x] Run `./codemap .` and verify tokens appear in output
- [x] Verify warning indicator appears for large files
- [x] Verify header shows total tokens

---

## Phase 2: Symbol Search - `get_symbol` (P0)

> **Objective:** Enable precise semantic search for functions and types

### Task 2.1: Create `scanner/symbol.go` with types
- [x] Create new file `scanner/symbol.go`
- [x] Define `SymbolQuery` struct with `Name`, `Kind`, `File` fields
- [x] Define `SymbolMatch` struct with `Name`, `Kind`, `Signature`, `TypeKind`, `File`, `Line`, `Exported` fields

### Task 2.2: Implement `SearchSymbols` function
- [x] **File:** `scanner/symbol.go`
- [x] Implement search logic that filters `[]FileAnalysis`
- [x] Support case-insensitive substring matching on Name
- [x] Support filtering by Kind ("function", "type", "all")
- [x] Support filtering by File (optional)

### Task 2.3: Add `SymbolInput` struct to MCP
- [x] **File:** `mcp/main.go`
- [x] Add `SymbolInput` struct with `Path`, `Name`, `Kind`, `File` fields
- [x] Add JSON schema annotations

### Task 2.4: Implement `handleGetSymbol` handler
- [x] **File:** `mcp/main.go`
- [x] Validate path using existing pattern
- [x] Call `ScanForDeps` with appropriate detail level
- [x] Call `SearchSymbols` with query parameters
- [x] Format output as ASCII table with file:line references

### Task 2.5: Register `get_symbol` tool in MCP server
- [x] **File:** `mcp/main.go`
- [x] Add tool registration with description and schema
- [x] Update tool count comment

### Task 2.6: Validate Phase 2
- [x] Run `go build -o codemap-mcp ./mcp/`
- [x] Test `get_symbol` tool with various queries
- [x] Verify file:line format in output

---

## Phase 3: MCP Enhancements (P1)

> **Objective:** Add API mode and centralize path validation

### Task 3.1: Add `Mode` field to `DepsInput`
- [x] **File:** `mcp/main.go`
- [x] Add `Mode string json:"mode,omitempty"` to `DepsInput`
- [x] Document valid values: "deps" (default), "api"

### Task 3.2: Implement API mode in `handleGetDependencies`
- [x] **File:** `mcp/main.go`
- [x] Check if `input.Mode == "api"`
- [x] Call `render.APIView` instead of `render.Depgraph` when API mode
- [x] Ensure proper output capture

### Task 3.3: Create `validatePath` helper function
- [x] **File:** `mcp/main.go`
- [x] Create function: `func validatePath(path string) (string, error)`
- [x] Handle empty path, filepath.Abs, and os.Stat checks
- [x] Return absolute path or error

### Task 3.4: Refactor handlers to use `validatePath`
- [x] **File:** `mcp/main.go`
- [x] Update `handleGetStructure` to use `validatePath`
- [x] Update `handleGetDependencies` to use `validatePath`
- [x] Update `handleGetDiff` to use `validatePath`
- [x] Update `handleFindFile` to use `validatePath`
- [x] Update `handleGetImporters` to use `validatePath`
- [x] Update `handleListProjects` to use `validatePath`
- [x] Verify no duplicate `filepath.Abs` calls remain

### Task 3.5: Validate Phase 3
- [x] Run `go build -o codemap-mcp ./mcp/`
- [x] Test `get_dependencies` with `mode=api`
- [x] Verify path validation works correctly

---

## Final Validation & Testing

### Integration Tests
- [x] `./codemap .` - Verify tree with tokens
- [x] `./codemap --deps .` - Verify functions/types have line numbers
- [x] `./codemap --deps --json .` - Verify JSON includes `line` and `tokens` fields
- [x] MCP: `get_symbol(path=".", name="Scan")` - Verify results
- [x] MCP: `get_dependencies(path=".", mode="api")` - Verify API output

### Manual Verification Checklist
- [x] Line numbers are 1-indexed and match actual file lines
- [x] Token estimates use `~` prefix for approximation
- [x] Warning indicator `[!]` appears for files > 8k tokens
- [x] Header shows aggregated token count
- [x] `get_symbol` returns results with `file:line` format
- [x] API mode produces same output as `render.APIView`
- [x] No duplicate path validation code in MCP handlers

---

## Completion Notes

### Summary
- [x] All phases completed
- [x] All validation criteria passed

**Implementation completed on 2025-12-03**

### Files Modified
- `scanner/types.go` - Added `Tokens`, `Line` fields, token estimation constants and helper
- `scanner/grammar.go` - Added line capture in `funcCapture`, `typeCapture`, handlers, and Build methods
- `scanner/walker.go` - Added token calculation in `ScanFiles`
- `scanner/symbol.go` - New file with `SearchSymbols` function
- `render/tree.go` - Added token display in header and `[!]` warning for large files
- `mcp/main.go` - Added `get_symbol` tool, `validatePath` helper, API mode support

### Deviations from Plan
- Token display format: Changed from `(~Xk tokens)` after each file to `[!]` warning indicator only for files > 8k tokens (simpler, less cluttered output)
- The `validatePath` helper also handles `~/` expansion (bonus feature)

### Follow-up Items
- Consider adding token display per-file in verbose mode
- Test `get_symbol` with various languages to ensure line numbers are accurate
- Consider adding sorting options to `get_symbol` results
