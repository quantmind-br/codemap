# TASKS: Codemap GraphRAG Implementation

## Project Briefing

**Objective**: Transform `codemap` from a structural file-tree mapper into a **GraphRAG (Graph-based Retrieval Augmented Generation)** system enabling LLM agents to "understand" codebases through pre-computed summaries, call graphs, and semantic search.

**Current State**: CLI tool with tree view, dependency flow, diff mode, skyline visualization, and MCP server with 8 tools.

**Target State**: Full GraphRAG system with knowledge graph persistence, LLM-powered explanations, and hybrid semantic/structural search.

**Critical Constraints**:
1. **Portability**: Cross-compile support (Linux/Mac/Windows). Avoid CGO.
2. **Performance**: Fast CLI startup (<100ms) and indexing.
3. **Token Efficiency**: Optimize all outputs for LLM context windows.

---

## Phase 1: Knowledge Graph Foundation

### 1.1 Core Data Structures (`graph/` package)

- [ ] **1.1.1** Create `graph/` package directory structure
- [ ] **1.1.2** Define `NodeKind` enum (File, Package, Function, Type, Method, Variable)
- [ ] **1.1.3** Define `EdgeKind` enum (Imports, Calls, Defines, Contains, References)
- [ ] **1.1.4** Implement `NodeID` generation using deterministic `sha256(path + symbol)`
- [ ] **1.1.5** Define `Node` struct with fields: ID, Kind, Name, Path, Line, Signature, DocString
- [ ] **1.1.6** Define `Edge` struct with fields: From, To, Kind, Weight, Location
- [ ] **1.1.7** Implement `CodeGraph` struct with map-based indexes (`nodesByID`, `edgesByFrom`, `edgesByTo`)
- [ ] **1.1.8** Implement `AddNode()`, `AddEdge()`, `GetNode()`, `GetEdges()` methods
- [ ] **1.1.9** Implement `SaveBinary(path)` using `encoding/gob`
- [ ] **1.1.10** Implement `LoadBinary(path)` using `encoding/gob`
- [ ] **1.1.11** Benchmark persistence for target <100ms with 100k nodes

### 1.2 Scanner Enhancements (`scanner/`)

- [ ] **1.2.1** Refactor `walker.go` to decouple file scanning from data generation
- [ ] **1.2.2** Create `scanner/calls.go` for call expression extraction
- [ ] **1.2.3** Update `queries/go.scm` to capture `call_expression` nodes
- [ ] **1.2.4** Update `queries/python.scm` to capture `call` nodes
- [ ] **1.2.5** Update `queries/javascript.scm` to capture `call_expression` nodes
- [ ] **1.2.6** Update `queries/typescript.scm` to capture `call_expression` nodes
- [ ] **1.2.7** Update remaining language queries (rust, java, c, cpp, etc.)
- [ ] **1.2.8** Implement `ExtractCalls(file, source)` function returning call sites
- [ ] **1.2.9** Implement `SyntacticCallGraph` extractor aggregating all files
- [ ] **1.2.10** Implement `ImportGraphFilter` to eliminate calls to non-imported packages
- [ ] **1.2.11** Implement `ArityFilter` to eliminate calls where arg count mismatches

### 1.3 Graph Builder (`graph/`)

- [ ] **1.3.1** Create `graph/builder.go` for graph construction logic
- [ ] **1.3.2** Implement `BuildFromScanner(root string)` orchestrating full graph build
- [ ] **1.3.3** Convert `FileAnalysis` results into `Node` objects
- [ ] **1.3.4** Convert import relationships into `Edge` objects (EdgeKind: Imports)
- [ ] **1.3.5** Convert call relationships into `Edge` objects (EdgeKind: Calls)
- [ ] **1.3.6** Implement `IsGraphStale(root)` using file modification times
- [ ] **1.3.7** Implement incremental update logic for changed files only
- [ ] **1.3.8** Add progress reporting callback for large codebases

### 1.4 CLI Commands

- [ ] **1.4.1** Implement `codemap index` command in `main.go`
- [ ] **1.4.2** Add `--force` flag to rebuild graph even if fresh
- [ ] **1.4.3** Add `--output` flag to specify graph file location (default: `.codemap/graph.gob`)
- [ ] **1.4.4** Implement `codemap query --from <symbol>` showing outgoing edges
- [ ] **1.4.5** Implement `codemap query --to <symbol>` showing incoming edges
- [ ] **1.4.6** Implement `codemap query --from <A> --to <B>` for path tracing
- [ ] **1.4.7** Add `--json` output format for query results
- [ ] **1.4.8** Add `--depth` flag to limit traversal depth

### 1.5 MCP Tools

- [ ] **1.5.1** Update `get_structure` to use graph if `.codemap/graph.gob` exists
- [ ] **1.5.2** Implement `trace_path` MCP tool (find connections between symbols)
- [ ] **1.5.3** Implement `get_callers` MCP tool (who calls this symbol)
- [ ] **1.5.4** Implement `get_callees` MCP tool (what does this symbol call)

### 1.6 Testing

- [ ] **1.6.1** Create `scanner/testdata/corpus/` directory
- [ ] **1.6.2** Add Go test files with known call patterns
- [ ] **1.6.3** Add Python test files with known call patterns
- [ ] **1.6.4** Add TypeScript test files with known call patterns
- [ ] **1.6.5** Implement snapshot tests comparing generated graph to JSON baseline
- [ ] **1.6.6** Manual verification: run against codemap itself and 2-3 real projects

---

## Phase 2: Semantic Intelligence (LLM Integration)

### 2.1 Configuration System (`config/`)

- [ ] **2.1.1** Create `config/` package directory
- [ ] **2.1.2** Define `Config` struct with LLM settings
- [ ] **2.1.3** Implement loading from `~/.config/codemap/config.yaml`
- [ ] **2.1.4** Implement loading from `.codemap/config.yaml` (project-level override)
- [ ] **2.1.5** Handle API keys safely (env vars, file permissions)
- [ ] **2.1.6** Add validation for required config fields

### 2.2 LLM Client Layer (`analyze/`)

- [ ] **2.2.1** Create `analyze/` package directory
- [ ] **2.2.2** Define `LLMClient` interface with `Complete(prompt)` and `Embed(text)` methods
- [ ] **2.2.3** Implement `OllamaClient` for local inference (default)
- [ ] **2.2.4** Implement `OpenAIClient` for cloud inference
- [ ] **2.2.5** Implement `AnthropicClient` for Claude API
- [ ] **2.2.6** Add retry logic and timeout handling
- [ ] **2.2.7** Add token counting utilities

### 2.3 Summarization Engine

- [ ] **2.3.1** Implement `ReadSymbolSource(node)` to extract raw code from file
- [ ] **2.3.2** Design prompt template for "Explain this function"
- [ ] **2.3.3** Design prompt template for "Summarize this module"
- [ ] **2.3.4** Create `cache/` package for response caching
- [ ] **2.3.5** Implement `ContentHash` calculation for cache invalidation
- [ ] **2.3.6** Implement JSON-based cache storage in `.codemap/cache/`
- [ ] **2.3.7** Add cache hit/miss statistics

### 2.4 CLI Commands

- [ ] **2.4.1** Implement `codemap explain <symbol>` command
- [ ] **2.4.2** Add `--model` flag to select LLM model
- [ ] **2.4.3** Add `--no-cache` flag to bypass cache
- [ ] **2.4.4** Implement `codemap summarize <path>` for module summaries

### 2.5 MCP Tools

- [ ] **2.5.1** Implement `explain_symbol` MCP tool
- [ ] **2.5.2** Implement `summarize_module` MCP tool
- [ ] **2.5.3** Add structured output with code context

### 2.6 Testing

- [ ] **2.6.1** Create mock LLM client returning fixed strings
- [ ] **2.6.2** Test caching with content hash validation
- [ ] **2.6.3** Manual verification with Ollama and OpenAI

---

## Phase 3: Hybrid Retrieval & Search

### 3.1 Vector Storage (`graph/vectors.go`)

- [ ] **3.1.1** Define `VectorIndex` interface
- [ ] **3.1.2** Implement in-memory vector storage using `gob` serialization
- [ ] **3.1.3** Implement Cosine Similarity search
- [ ] **3.1.4** Optimize for <50k vectors target
- [ ] **3.1.5** Benchmark search performance

### 3.2 Embedding Pipeline

- [ ] **3.2.1** Implement `NodeToText(node)` strategy (Signature + DocString + Summary)
- [ ] **3.2.2** Implement `EmbedNodes(nodes)` with batching
- [ ] **3.2.3** Add progress bar for embedding generation
- [ ] **3.2.4** Store embeddings alongside graph in `.codemap/vectors.gob`

### 3.3 Hybrid Search Engine (`analyze/retriever.go`)

- [ ] **3.3.1** Implement vector search (top-k semantic matches)
- [ ] **3.3.2** Implement graph search (name matching/fuzzy)
- [ ] **3.3.3** Implement Reciprocal Rank Fusion for result merging
- [ ] **3.3.4** Add context expansion (retrieve callers/callees of top results)
- [ ] **3.3.5** Tune weights for optimal relevance

### 3.4 CLI Commands

- [ ] **3.4.1** Implement `codemap search "natural language query"`
- [ ] **3.4.2** Add `--limit` flag for result count
- [ ] **3.4.3** Add `--expand` flag for context expansion
- [ ] **3.4.4** Add `--json` output format

### 3.5 MCP Tools

- [ ] **3.5.1** Implement `semantic_search` MCP tool
- [ ] **3.5.2** Add structured response with relevance scores
- [ ] **3.5.3** Include code snippets in results

### 3.6 Performance & Polish

- [ ] **3.6.1** Profile embedding generation performance
- [ ] **3.6.2** Add TUI progress bars for long operations
- [ ] **3.6.3** Optimize memory usage for large codebases
- [ ] **3.6.4** Documentation and README updates

---

## Validation & Testing

- [ ] **V.1** Run `go fmt ./...` and `go vet ./...`
- [ ] **V.2** Test all new commands manually
- [ ] **V.3** Test MCP tools via Claude Desktop
- [ ] **V.4** Verify cross-platform build (`GOOS=darwin`, `GOOS=windows`)
- [ ] **V.5** Performance benchmarks meet targets (<100ms startup)

---

## Progress Tracking

| Phase | Total Tasks | Completed | Progress |
|-------|-------------|-----------|----------|
| Phase 1 | 45 | 0 | 0% |
| Phase 2 | 27 | 0 | 0% |
| Phase 3 | 22 | 0 | 0% |
| Validation | 5 | 0 | 0% |
| **Total** | **99** | **0** | **0%** |

---

## Notes & Deviations

*(To be filled during implementation)*
