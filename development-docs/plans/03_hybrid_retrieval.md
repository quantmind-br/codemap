# Phase 3: Hybrid Retrieval & Search

**Goal**: Enable natural language search over the codebase.
**Focus**: Embeddings, Vector Search, and merging Structural + Semantic results.
**Prerequisite**: Phase 2 completed.

## 1. Vector Storage (`graph/vectors.go`)

- [ ] **Design Decision**: Use **pure Go** vector logic or a lightweight library (e.g., `github.com/philippgille/gokv`) instead of CGO SQLite extensions to maintain portability.
- [ ] **Storage**: Implement `VectorIndex` interface.
    - Simple flat file with `gob` serialization is likely sufficient for <50k vectors.
- [ ] **Search**: Implement Cosine Similarity search in memory.

## 2. Embedding Pipeline

- [ ] **Batch Processing**: Implement `EmbedNodes(nodes)` to batch calls to the embedding model.
- [ ] **Text Representation**: Define `NodeToText(node)` strategy (Signature + Docstring + Summary).

## 3. Hybrid Search Engine (`analyze/retriever.go`)

- [ ] **Algorithm**:
    1.  Get `top_k` from Vector Search (Semantic).
    2.  Get `top_k` from Graph Query (Name matching/fuzzy search).
    3.  Rank fusion (Reciprocal Rank Fusion or weighted score).
- [ ] **Context Expansion**: Retrieve connected nodes (Callers/Callees) for the top results.

## 4. CLI & MCP Tools

- [ ] **Command**: `codemap search "natural language query"`
- [ ] **MCP Tool**: `semantic_search` (The "Killer Feature" for agents).

## 5. Performance Tuning

- [ ] **Profile**: Ensure embedding generation doesn't block the UI/CLI.
- [ ] **Progress Bars**: Add TUI feedback for long-running embedding tasks.
