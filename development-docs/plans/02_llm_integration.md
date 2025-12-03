# Phase 2: Semantic Intelligence (LLM)

**Goal**: Add semantic understanding and explanation capabilities.
**Focus**: LLM client integration, summarization pipeline, and configuration.
**Prerequisite**: Phase 1 completed.

## 1. Configuration System (`config/`)

- [ ] **Config Manager**: Implement loading from `~/.config/codemap/config.yaml`.
- [ ] **Secrets**: Handle API keys safely (Ollama URL, OpenAI Key).

## 2. LLM Client Layer (`analyze/`)

- [ ] **Interfaces**: Define `LLMClient` (Complete, Embed).
- [ ] **Ollama**: Implement client for local inference (preferred default).
- [ ] **OpenAI/Anthropic**: Implement clients for cloud inference.

## 3. Summarization Engine

- [ ] **Code Reader**: Implement `ReadSymbolSource(node)` to extract raw code efficiently.
- [ ] **Prompt Engineering**: Create robust prompts for "Explain this symbol" and "Summarize this module".
- [ ] **Caching Strategy**:
    - Implement `cache/` system storing JSON responses.
    - **Critical**: Use `ContentHash` of the symbol's source code as the cache invalidation key.

## 4. CLI & MCP Tools

- [ ] **Command**: `codemap explain <symbol>` (CLI wrapper for explanation).
- [ ] **MCP Tool**: `explain_symbol` (Returns structured summary + code context).
- [ ] **MCP Tool**: `summarize_module` (Aggregates summaries of files in a folder).

## 5. Testing

- [ ] **Mock LLM**: Create a mock client that returns fixed strings to test the pipeline without spending tokens/time.
