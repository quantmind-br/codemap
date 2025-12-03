# TASKS.md - Enhanced Code Analysis Features

## Project Briefing

**Objective:** Implement advanced code analysis features for `codemap` including:
1. Full function signatures (parameters and return types)
2. Type extraction (structs, classes, interfaces, traits, enums)
3. API Surface mode for compact public API visualization

**Design Principles:**
- Backward compatibility - existing output unchanged by default
- Token efficiency - new details are opt-in via `--detail` flag
- Semantic consistency - types normalized across languages

**Key Files Modified:**
- `scanner/types.go` - New data structures
- `scanner/grammar.go` - Enhanced analysis logic
- `scanner/queries/*.scm` - Updated tree-sitter queries
- `render/depgraph.go` - Updated rendering
- `render/api.go` - New API surface view
- `main.go` - New CLI flags
- `mcp/main.go` - MCP server updates

---

## Phase 1: Base Data Structures

### 1.1 New Types in scanner/types.go
- [x] Add `DetailLevel` type and constants (DetailNone=0, DetailSignature=1, DetailFull=2)
- [x] Add `FuncInfo` struct (Name, Signature, Receiver, IsExported)
- [x] Add `TypeKind` type and constants (struct, class, interface, trait, enum, alias, protocol)
- [x] Add `TypeInfo` struct (Name, Kind, Fields, Methods, IsExported)
- [x] Update `FileAnalysis` struct (Functions: []FuncInfo, add Types: []TypeInfo)
- [x] Add `DetailLevel` field to `DepsProject` struct

### 1.2 JSON Backward Compatibility
- [x] Implement custom `MarshalJSON` for `FuncInfo` (serialize as string when no signature)
- [x] Implement custom `UnmarshalJSON` for backward compatibility
- [x] Ensure existing JSON consumers still work

### 1.3 CLI Flags in main.go
- [x] Add `--detail` flag (int, 0-2)
- [x] Add `--api` flag (bool)
- [x] Update help text with new flags
- [x] Wire flags through to `runDepsMode`

### 1.4 Scanner Integration
- [x] Update `ScanForDeps` to accept `DetailLevel` parameter
- [x] Update `AnalyzeFile` to accept `DetailLevel` parameter
- [x] Propagate detail level through the analysis pipeline

### 1.5 Validation
- [x] Build succeeds: `go build -o codemap .`
- [x] `./codemap --deps .` still works (backward compatibility)
- [x] `./codemap --help` shows new flags

---

## Phase 2: Go + Python Queries

### 2.1 Go Query (scanner/queries/go.scm)
- [x] Rewrite query with granular captures:
  - [x] `@func.name`, `@func.params` for functions
  - [x] `@func.receiver` for methods
  - [x] `@type.name`, `@type.struct`, `@type.interface`
- [x] Keep existing `@import` captures

### 2.2 Python Query (scanner/queries/python.scm)
- [x] Rewrite query with granular captures:
  - [x] `@func.name`, `@func.params` for functions
  - [x] `@type.name`, `@type.class`
- [x] Keep existing `@import` captures

### 2.3 Analysis Logic in scanner/grammar.go
- [x] Add `funcCapture` struct for collecting function components
- [x] Add `typeCapture` struct for collecting type components
- [x] Implement `handleFuncCapture` function
- [x] Implement `handleTypeCapture` function
- [x] Implement `buildSignature` function for Go and Python
- [x] Implement `IsExportedName` function for Go and Python
- [x] Add deduplication helpers: `dedupeFuncs`, `dedupeTypes`

### 2.4 Validation
- [x] Build succeeds
- [x] Test Go file with `--detail 0` (names only)
- [x] Test Go file with `--detail 1` (shows signatures)
- [x] JSON output is valid and backward compatible

---

## Phase 3: TypeScript + JavaScript + Rust Queries

### 3.1 TypeScript Query (scanner/queries/typescript.scm)
- [x] Rewrite query with granular captures for:
  - [x] Functions with parameters
  - [x] Arrow functions
  - [x] Method definitions
  - [x] Interfaces, classes, type aliases, enums

### 3.2 JavaScript Query (scanner/queries/javascript.scm)
- [x] Rewrite query with granular captures

### 3.3 Rust Query (scanner/queries/rust.scm)
- [x] Rewrite query with granular captures for:
  - [x] Functions with parameters
  - [x] Structs, enums, traits

---

## Phase 4: Java + C# Queries

### 4.1 Java Query (scanner/queries/java.scm)
- [x] Rewrite query with granular captures:
  - [x] Methods and constructors with params
  - [x] Classes, interfaces, enums

### 4.2 C# Query (scanner/queries/c_sharp.scm)
- [x] Rewrite query with granular captures:
  - [x] Methods and constructors
  - [x] Classes, interfaces, structs, enums

---

## Phase 5: Rendering Updates

### 5.1 Update render/depgraph.go
- [x] Update summary to show type counts

### 5.2 Create render/api.go
- [x] Implement `APIView` function:
  - [x] Group files by package/directory
  - [x] Filter to exported types and functions only
  - [x] Group methods by their receiver type
  - [x] Render compact view with type icons
- [x] Implement `typeIcon` helper for ASCII type icons ([S], [C], [I], etc.)
- [x] Implement `extractTypeName` helper for receiver parsing
- [x] Add summary statistics (exported types/functions count)

### 5.3 Wire API Mode in main.go
- [x] Call `render.APIView` when `--api` flag is set

---

## Phase 6: MCP Server Updates

### 6.1 Update mcp/main.go
- [x] Add `DepsInput` type with `detail` parameter
- [x] Update `get_dependencies` tool to use new input type
- [x] Pass detail level to `ScanForDeps`
- [x] Update tool description for new parameter

### 6.2 Validation
- [x] Build MCP server: `go build -o codemap-mcp ./mcp/`

---

## Phase 7: Testing & Documentation

### 7.1 Manual Testing
- [x] Test on codemap itself with all flags
- [x] Test tree mode (basic view)
- [x] Test deps mode (detail=0)
- [x] Test deps mode with signatures (detail=1)
- [x] Test API surface mode

### 7.2 Final Validation
- [x] All existing modes still work (tree, deps, diff, skyline)
- [x] Build both binaries successfully
- [x] Run `go fmt ./...`

---

## Completion Summary

**Status:** [x] Completed

| Phase | Description | Status |
|-------|-------------|--------|
| 1 | Base Data Structures | [x] Complete |
| 2 | Go + Python Queries | [x] Complete |
| 3 | TypeScript + Rust Queries | [x] Complete |
| 4 | Java + C# Queries | [x] Complete |
| 5 | Rendering Updates | [x] Complete |
| 6 | MCP Server Updates | [x] Complete |
| 7 | Testing & Documentation | [x] Complete |

### Implementation Notes

1. **Function Signatures**: Function parameters are captured via `@func.params` and methods include receivers via `@func.receiver`. Signatures are built using the `buildSignature` function in `grammar.go`.

2. **Type Extraction**: Types are captured with pattern markers (`@type.struct`, `@type.class`, etc.) that set the `Kind` field in `TypeInfo`. Type fields are not extracted in this implementation to keep queries simple.

3. **Return Types**: Due to tree-sitter query syntax limitations with optional fields, function return types are not captured. Signatures show function name and parameters only.

4. **Backward Compatibility**: `FuncInfo.MarshalJSON()` serializes as plain string when no extended info is present, ensuring existing JSON consumers continue to work.

5. **Queries Updated**: Go, Python, TypeScript, JavaScript, Rust, Java, and C# queries have been updated with the new capture patterns. Other languages (C, C++, Swift, etc.) continue to use legacy `@function` captures.

---

*Created: 2024-12-03*
*Completed: 2024-12-03*
