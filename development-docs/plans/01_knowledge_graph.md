# Phase 1: Knowledge Graph Foundation

**Goal**: Transform `codemap` from a file-tree viewer to a graph-aware analysis tool.
**Focus**: Structural relationships, call graph extraction, and persistence.
**No LLM dependency in this phase.**

## 1. Core Data Structures (`graph/`)

- [ ] **Define Types**: Create `Node`, `Edge`, `NodeKind`, `EdgeKind` structs.
    - *Correction*: Ensure `NodeID` generation is strictly deterministic (`sha256(path + symbol)`).
- [ ] **Graph Storage**: Implement `CodeGraph` struct with map-based indexes (`nodesByID`, `edgesByFrom`).
- [ ] **Persistence**:
    - Implement `SaveBinary(path)` / `LoadBinary(path)` using `encoding/gob` for fast startup.
    - *Note*: Skip SQLite for now if `gob` is fast enough (<100ms for 100k nodes).

## 2. Scanner Enhancements (`scanner/`)

- [ ] **Refactor `walker.go`**: Decouple file scanning from data generation.
- [ ] **Call Extraction (`calls.go`)**:
    - Update Tree-sitter queries (`queries/*.scm`) to capture `call_expression`.
    - Implement `SyntacticCallGraph` extractor.
- [ ] **Edge Validation**:
    - Implement `ImportGraphFilter`: Eliminate calls to packages not imported by the file.
    - Implement `ArityFilter`: Eliminate calls where arg count mismatches (high precision).

## 3. Graph Builder (`graph/`)

- [ ] **Builder Logic**: Convert `FileAnalysis` (from scanner) into `Node` and `Edge` objects.
- [ ] **Delta Analysis**: Implement `IsGraphStale(root)` using file modification times to avoid unnecessary re-indexing.

## 4. CLI & MCP Tools

- [ ] **Command**: `codemap index` (Builds `.codemap/graph.gob`).
- [ ] **Command**: `codemap query --from <symbol> --to <symbol>` (Path tracing).
- [ ] **MCP Tool**: `get_structure` (Update to use Graph if available).
- [ ] **MCP Tool**: `trace_path` (New tool for finding connections).

## 5. Testing Strategy

- [ ] **Golden Corpus**: Create `scanner/testdata/corpus/` with polyglot files (Go, Py, TS).
- [ ] **Snapshot Tests**: Verify that the generated graph nodes/edges match a JSON snapshot exactly.
