# TASKS.md: Google Gemini API Integration

## Project Briefing

**Objective:** Integrate Google Gemini API as a new LLM provider in codemap, enabling users to leverage Gemini models for code analysis, explanation, summarization, and semantic search via embeddings.

**Scope:**
- Implement `GeminiClient` adhering to `analyze.LLMClient` interface
- Add configuration support for `ProviderGemini` with API key and base URL
- Integrate in factory functions (`NewClient`, `NewEmbeddingClient`)
- Update documentation (.env.example, README.md)

**Non-Scope:**
- Streaming responses
- Vertex AI (OAuth/GCP) - only Google AI Studio (API Key)
- Multimodal inputs (images, audio)

---

## Implementation Checklist

### Phase 1: Configuration (`config/config.go`)

- [x] 1.1 Add `ProviderGemini` to provider enum
- [x] 1.2 Add `GeminiAPIKey`, `GeminiBaseURL` fields to `LLMConfig`
- [x] 1.3 Update `applyEnvOverrides()` for `GEMINI_API_KEY`, `GEMINI_BASE_URL`
- [x] 1.4 Update `Validate()` to validate `GeminiAPIKey`

### Phase 2: GeminiClient Implementation (`analyze/gemini.go`)

- [x] 2.1 Create `GeminiClient` struct and `NewGeminiClient()` constructor
- [x] 2.2 Define request/response types (8 structs)
  - [x] `geminiPart`
  - [x] `geminiContent`
  - [x] `geminiGenerationConfig`
  - [x] `geminiRequest`
  - [x] `geminiResponse`
  - [x] `geminiErrorResponse`
  - [x] `geminiEmbedRequest`
  - [x] `geminiEmbedResponse`
- [x] 2.3 Implement `Name()` and `Ping()` methods
- [x] 2.4 Implement `Models()` method (static list)
- [x] 2.5 Implement `Complete()` with:
  - [x] Message conversion (system → systemInstruction, assistant → model)
  - [x] Retry logic with exponential backoff
  - [x] Finish reason mapping
  - [x] Token usage extraction
- [x] 2.6 Implement `Embed()` method

### Phase 3: Factory Integration (`analyze/factory.go`)

- [x] 3.1 Add `ProviderGemini` case in `NewClient()`
- [x] 3.2 Verify `NewEmbeddingClient()` works with Gemini (no changes needed)

### Phase 4: Documentation & Testing

- [x] 4.1 Update `.env.example` with Gemini configuration section
- [x] 4.2 Update `README.md` to document Gemini provider
- [x] 4.3 Build and verification
  - [x] `go build -o codemap .` compiles without errors
  - [x] `go vet ./...` passes without warnings
  - [x] `go fmt ./...` produces no changes
- [x] 4.4 Validation test (without API key)
  - [x] Verify error message: "gemini_api_key required for gemini provider"
- [x] 4.5 Integration tests (with API key)
  - [x] Ping API successful
  - [x] Complete generates text correctly
  - [x] `--embed` generates embeddings with Gemini (768 dimensions)

---

## Validation & Testing

### Build Verification
```bash
go build -o codemap .    # ✅ Compiles without errors
go vet ./...             # ✅ Passes without warnings
go fmt ./...             # ✅ Produces no changes
```

### Validation Test (No API Key)
```bash
# Verified via direct test - returns:
# "config validation failed: gemini_api_key required for gemini provider (or set GEMINI_API_KEY env var)"
```

### Integration Test (With API Key)
```bash
export GEMINI_API_KEY="your-key-here"
export CODEMAP_LLM_PROVIDER="gemini"
export CODEMAP_EMBEDDING_MODEL="text-embedding-004"

# Test embeddings - ✅ Verified working (768 dimensions)
./codemap --embed .
```

---

## Files Changed

| File | Type | Status |
|------|------|--------|
| `config/config.go` | Modified | ✅ Complete |
| `analyze/gemini.go` | New | ✅ Complete |
| `analyze/factory.go` | Modified | ✅ Complete |
| `.env.example` | Modified | ✅ Complete |
| `README.md` | Modified | ✅ Complete |

---

## Additional Changes Made During Implementation

### Bug Fixes
1. **Retry Logic Bug (analyze/gemini.go):** Fixed issue where HTTP response body was closed prematurely for 4xx errors, preventing proper error message extraction

### Model Updates
1. **Default Model Updated:** Changed from `gemini-2.0-flash-exp` to `gemini-2.0-flash` (stable release)
2. **Models() Updated:** Updated available models list to reflect current API (gemini-2.5-flash, gemini-2.5-pro, gemini-2.0-flash, gemini-2.0-flash-exp)

### Configuration Enhancements
1. **Embedding Environment Variables:** Added `CODEMAP_EMBEDDING_MODEL` and `CODEMAP_EMBEDDING_PROVIDER` env var overrides in `config/config.go`
2. **Updated .env.example:** Added documentation for embedding model configuration per provider

---

## Notes

- Implementation follows the existing adapter pattern from OpenAI/Anthropic/Ollama clients
- Gemini embedding endpoint does NOT return token usage (documented, returns zero)
- System messages are converted to Gemini's `systemInstruction` field
- Role mapping: `assistant` → `model`
- Finish reason mapping: `STOP` → `stop`, `MAX_TOKENS` → `length`, `SAFETY`/`RECITATION` → `content_filter`
- When using Gemini for embeddings, set `CODEMAP_EMBEDDING_MODEL=text-embedding-004`

---

## Summary

The Google Gemini API integration is **complete and verified**. All tests pass:
- ✅ Build compiles without errors
- ✅ Static analysis (go vet) passes
- ✅ Code formatted correctly
- ✅ API validation works correctly
- ✅ Complete API generates text successfully
- ✅ Embed API generates 768-dimension vectors
