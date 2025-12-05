# PLAN.md: Integração do Google Gemini API

## 1. Executive Summary

### 1.1. Objetivo
Integrar o **Google Gemini API** como um novo provider de LLM no sistema `codemap`, permitindo que usuários utilizem modelos Gemini para análise, explicação e sumarização de código, além de geração de embeddings para busca semântica.

### 1.2. Escopo

**Incluído:**
- Implementar `GeminiClient` aderindo à interface `analyze.LLMClient`
- Adicionar configuração para `ProviderGemini` com campos específicos
- Integrar nas factories `NewClient` e `NewEmbeddingClient`
- Suportar tanto completion quanto embeddings nativamente
- Documentação completa (README, .env.example)

**Não-Escopo:**
- Streaming responses (arquitetura atual não suporta)
- Suporte a Vertex AI (OAuth/GCP) - apenas Google AI Studio (API Key)
- Multimodal inputs (imagens, áudio) - apenas texto

### 1.3. Definição de Esforço

| Tag | Linhas de Código | Tempo Estimado | Complexidade |
|-----|------------------|----------------|--------------|
| **S** (Small) | < 30 linhas | < 30 min | Alterações pontuais, adição de constantes/campos |
| **M** (Medium) | 30-100 linhas | 30-60 min | Funções completas, structs, lógica moderada |
| **L** (Large) | > 100 linhas | > 1h | Implementações complexas com múltiplas dependências |

---

## 2. Current Situation Analysis

### 2.1. Arquitetura Existente

```
analyze/
├── client.go      # LLMClient interface, Request/Response structs
├── factory.go     # NewClient(), NewEmbeddingClient()
├── openai.go      # OpenAI adapter (~350 linhas)
├── anthropic.go   # Anthropic adapter (~280 linhas, sem embeddings)
├── ollama.go      # Ollama adapter (~300 linhas)
└── tokens.go      # Token estimation utilities

config/
└── config.go      # Provider enum, LLMConfig struct, Validate()
```

### 2.2. Interface LLMClient (analyze/client.go)

```go
type LLMClient interface {
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
    Models(ctx context.Context) ([]string, error)
    Ping(ctx context.Context) error
    Name() string
}
```

### 2.3. Structs de Request/Response Existentes

```go
type Message struct {
    Role    string // "system", "user", "assistant"
    Content string
}

type CompletionRequest struct {
    Messages    []Message
    Model       string
    Temperature float64
    MaxTokens   int
    Stop        []string
}

type CompletionResponse struct {
    Content      string
    Model        string
    FinishReason string  // "stop", "length", "content_filter", "other"
    Duration     time.Duration
    Usage        TokenUsage
}

type EmbeddingRequest struct {
    Text  string
    Model string
}

type EmbeddingResponse struct {
    Embedding []float64
    Model     string
    Duration  time.Duration
    Usage     TokenUsage
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

### 2.4. Providers Atuais - Comparativo

| Provider | Completion | Embeddings | Auth Header | Notas |
|----------|------------|------------|-------------|-------|
| OpenAI | ✅ | ✅ | `Authorization: Bearer` | Referência de implementação |
| Anthropic | ✅ | ❌ | `x-api-key` | Fallback para Ollama em embeddings |
| Ollama | ✅ | ✅ | Nenhum | Local, sem API key |
| **Gemini** | 🎯 | 🎯 | `x-goog-api-key` | **A implementar** |

### 2.5. Gaps Identificados

1. **Mapeamento de Mensagens:** Gemini usa formato diferente para system messages
2. **Roles Diferentes:** `assistant` → `model` no Gemini
3. **Estrutura de Content:** `content: string` → `parts: [{text: string}]`
4. **Finish Reasons:** Nomenclatura diferente (STOP vs stop)
5. **Token Usage em Embeddings:** Gemini não retorna contagem de tokens

---

## 3. Gemini API Technical Reference

### 3.1. Endpoints

| Operação | Endpoint | Método |
|----------|----------|--------|
| Generate | `https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent` | POST |
| Embed | `https://generativelanguage.googleapis.com/v1beta/models/{model}:embedContent` | POST |
| List Models | `https://generativelanguage.googleapis.com/v1beta/models` | GET |

### 3.2. Autenticação

**Header obrigatório:**
```
x-goog-api-key: $GEMINI_API_KEY
Content-Type: application/json
```

> ⚠️ **IMPORTANTE:** NÃO usar query parameter `?key=` para evitar vazamento de chaves em logs de servidor e histórico de navegador.

### 3.3. Modelos Disponíveis (Dezembro 2024)

**Completion:**
| Modelo | Context Window | Notas |
|--------|----------------|-------|
| `gemini-2.0-flash-exp` | 1M tokens | Mais recente, experimental |
| `gemini-1.5-flash` | 1M tokens | Estável, rápido |
| `gemini-1.5-flash-8b` | 1M tokens | Menor, mais econômico |
| `gemini-1.5-pro` | 2M tokens | Maior capacidade |

**Embeddings:**
| Modelo | Dimensões | Notas |
|--------|-----------|-------|
| `text-embedding-004` | 768 | Recomendado |
| `embedding-001` | 768 | Legacy |

### 3.4. Request Format - GenerateContent

```json
{
  "systemInstruction": {
    "parts": [{"text": "You are a code analysis assistant."}]
  },
  "contents": [
    {
      "role": "user",
      "parts": [{"text": "Explain this function..."}]
    },
    {
      "role": "model",
      "parts": [{"text": "This function does..."}]
    }
  ],
  "generationConfig": {
    "temperature": 0.1,
    "maxOutputTokens": 2048,
    "stopSequences": ["```"]
  }
}
```

**Diferenças Críticas vs OpenAI:**

| Aspecto | OpenAI | Gemini |
|---------|--------|--------|
| System message | `role: "system"` no array | Campo separado `systemInstruction` |
| Assistant role | `role: "assistant"` | `role: "model"` |
| Content | `content: string` | `parts: [{text: string}]` |
| Max tokens | `max_tokens` | `maxOutputTokens` |

### 3.5. Response Format - GenerateContent

```json
{
  "candidates": [{
    "content": {
      "parts": [{"text": "Response text here..."}],
      "role": "model"
    },
    "finishReason": "STOP",
    "index": 0
  }],
  "usageMetadata": {
    "promptTokenCount": 150,
    "candidatesTokenCount": 500,
    "totalTokenCount": 650
  },
  "modelVersion": "gemini-2.0-flash-exp"
}
```

### 3.6. Request Format - EmbedContent

```json
{
  "content": {
    "parts": [{"text": "Text to embed"}]
  }
}
```

### 3.7. Response Format - EmbedContent

```json
{
  "embedding": {
    "values": [0.123, -0.456, 0.789, ...]
  }
}
```

> ⚠️ **NOTA:** O endpoint de embedding NÃO retorna contagem de tokens. `EmbeddingResponse.Usage` será retornado zerado.

### 3.8. Error Response Format

```json
{
  "error": {
    "code": 400,
    "message": "Invalid value at 'contents[0].parts'",
    "status": "INVALID_ARGUMENT"
  }
}
```

### 3.9. Códigos de Erro HTTP

| Código | Status | Significado | Ação |
|--------|--------|-------------|------|
| `400` | INVALID_ARGUMENT | Request malformado | Verificar formato JSON |
| `401` | UNAUTHENTICATED | API key ausente | Verificar header |
| `403` | PERMISSION_DENIED | API key inválida | Retornar `ErrNoAPIKey` |
| `404` | NOT_FOUND | Modelo não existe | Retornar `ErrModelNotFound` |
| `429` | RESOURCE_EXHAUSTED | Rate limit | Retry com backoff |
| `500` | INTERNAL | Erro servidor | Retry com backoff |
| `503` | UNAVAILABLE | Serviço indisponível | Retry com backoff |

### 3.10. Finish Reason Mapping

| Gemini | Padrão codemap | Descrição |
|--------|----------------|-----------|
| `STOP` | `"stop"` | Geração completa normalmente |
| `MAX_TOKENS` | `"length"` | Limite de tokens atingido |
| `SAFETY` | `"content_filter"` | Bloqueado por filtro de segurança |
| `RECITATION` | `"content_filter"` | Bloqueado por detecção de cópia |
| `OTHER` | `"other"` | Motivo não especificado |
| _(vazio/null)_ | `"other"` | Fallback para casos inesperados |

---

## 4. Implementation Plan

### Phase 1: Configuration (`config/config.go`)

**Objetivo:** Preparar o sistema para reconhecer e configurar o provider Gemini.

#### Task 1.1: Adicionar Provider Enum [S]

**Arquivo:** `config/config.go`
**Localização:** Após linha 26 (após `ProviderAnthropic`)

```go
const (
    ProviderOllama    Provider = "ollama"
    ProviderOpenAI    Provider = "openai"
    ProviderAnthropic Provider = "anthropic"
    ProviderGemini    Provider = "gemini"  // NEW
)
```

**Critério de Conclusão:** Constante `ProviderGemini` declarada e compilável.

---

#### Task 1.2: Adicionar Campos no LLMConfig [S]

**Arquivo:** `config/config.go`
**Localização:** Dentro de `LLMConfig struct`, após campos Anthropic

```go
// Gemini-specific settings
GeminiAPIKey  string `yaml:"gemini_api_key"`  // GEMINI_API_KEY env var
GeminiBaseURL string `yaml:"gemini_base_url"` // Default: generativelanguage.googleapis.com
```

**Critério de Conclusão:** Campos adicionados com tags YAML corretas.

---

#### Task 1.3: Atualizar applyEnvOverrides() [S]

**Arquivo:** `config/config.go`
**Localização:** Função `applyEnvOverrides()`, após bloco Anthropic

```go
// Gemini
if v := os.Getenv("GEMINI_API_KEY"); v != "" {
    cfg.LLM.GeminiAPIKey = v
}
if v := os.Getenv("GEMINI_BASE_URL"); v != "" {
    cfg.LLM.GeminiBaseURL = v
}
```

**Critério de Conclusão:** Variáveis de ambiente `GEMINI_API_KEY` e `GEMINI_BASE_URL` reconhecidas.

---

#### Task 1.4: Atualizar Validate() [S]

**Arquivo:** `config/config.go`
**Localização:** Função `Validate()`, dentro do switch de providers

```go
case ProviderGemini:
    if c.LLM.GeminiAPIKey == "" {
        errs = append(errs, "gemini_api_key required for gemini provider (or set GEMINI_API_KEY env var)")
    }
```

**Critério de Conclusão:** Validação falha com mensagem clara quando API key está ausente.

---

### Phase 2: GeminiClient Implementation (`analyze/gemini.go`)

**Objetivo:** Criar o adaptador funcional para a API Gemini.

#### Task 2.1: Criar Arquivo e Struct Base [M]

**Arquivo:** `analyze/gemini.go` (NOVO)

```go
package analyze

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// GeminiClient implements LLMClient for Google's Gemini API.
type GeminiClient struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    config     ClientConfig
}

// NewGeminiClient creates a new Gemini client.
func NewGeminiClient(apiKey, baseURL string, cfg ClientConfig) (*GeminiClient, error) {
    if apiKey == "" {
        return nil, ErrNoAPIKey
    }
    if baseURL == "" {
        baseURL = "https://generativelanguage.googleapis.com/v1beta"
    }
    if cfg.Model == "" {
        cfg.Model = "gemini-2.0-flash-exp"
    }
    if cfg.EmbeddingModel == "" {
        cfg.EmbeddingModel = "text-embedding-004"
    }
    if cfg.Timeout == 0 {
        cfg.Timeout = 60 * time.Second
    }

    return &GeminiClient{
        apiKey:  apiKey,
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: cfg.Timeout,
        },
        config: cfg,
    }, nil
}
```

**Critério de Conclusão:** Struct criada, constructor com validação e defaults.

---

#### Task 2.2: Definir Request/Response Types [M]

**Arquivo:** `analyze/gemini.go`

```go
// geminiPart represents a content part in Gemini API.
type geminiPart struct {
    Text string `json:"text"`
}

// geminiContent represents a message in Gemini API.
type geminiContent struct {
    Role  string       `json:"role,omitempty"`
    Parts []geminiPart `json:"parts"`
}

// geminiGenerationConfig holds generation parameters.
type geminiGenerationConfig struct {
    Temperature     float64  `json:"temperature,omitempty"`
    MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
    StopSequences   []string `json:"stopSequences,omitempty"`
}

// geminiRequest is the request format for Gemini's generateContent API.
type geminiRequest struct {
    SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
    Contents          []geminiContent         `json:"contents"`
    GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

// geminiResponse is the response format from Gemini's generateContent API.
type geminiResponse struct {
    Candidates []struct {
        Content      geminiContent `json:"content"`
        FinishReason string        `json:"finishReason"`
        Index        int           `json:"index"`
    } `json:"candidates"`
    UsageMetadata struct {
        PromptTokenCount     int `json:"promptTokenCount"`
        CandidatesTokenCount int `json:"candidatesTokenCount"`
        TotalTokenCount      int `json:"totalTokenCount"`
    } `json:"usageMetadata"`
    ModelVersion string `json:"modelVersion"`
}

// geminiErrorResponse is the error format from Gemini API.
type geminiErrorResponse struct {
    Error struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
        Status  string `json:"status"`
    } `json:"error"`
}

// geminiEmbedRequest is the request format for Gemini's embedContent API.
type geminiEmbedRequest struct {
    Content geminiContent `json:"content"`
}

// geminiEmbedResponse is the response format from Gemini's embedContent API.
type geminiEmbedResponse struct {
    Embedding struct {
        Values []float64 `json:"values"`
    } `json:"embedding"`
}
```

**Critério de Conclusão:** Todas as 8 structs definidas e mapeando corretamente o JSON da API.

---

#### Task 2.3: Implementar Name() e Ping() [S]

**Arquivo:** `analyze/gemini.go`

```go
// Name returns the provider name.
func (c *GeminiClient) Name() string {
    return "gemini"
}

// Ping checks if the Gemini API is reachable by calling the models endpoint.
// This validates both connectivity and API key validity.
func (c *GeminiClient) Ping(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
    if err != nil {
        return err
    }
    req.Header.Set("x-goog-api-key", c.apiKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("gemini not reachable: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
        return ErrNoAPIKey
    }
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("gemini returned status %d", resp.StatusCode)
    }

    return nil
}
```

**Critério de Conclusão:** `Ping()` valida conectividade E credenciais via `/models`.

---

#### Task 2.4: Implementar Models() [S]

**Arquivo:** `analyze/gemini.go`

```go
// Models returns a list of available Gemini models.
// Returns a static list of known models for simplicity and speed.
func (c *GeminiClient) Models(ctx context.Context) ([]string, error) {
    return []string{
        // Completion models
        "gemini-2.0-flash-exp",
        "gemini-1.5-flash",
        "gemini-1.5-flash-8b",
        "gemini-1.5-pro",
        // Embedding models
        "text-embedding-004",
        "embedding-001",
    }, nil
}
```

**Critério de Conclusão:** Lista estática retornada sem chamada de API (otimização).

---

#### Task 2.5: Implementar Complete() [L]

**Arquivo:** `analyze/gemini.go`

```go
// Complete generates a completion using Gemini's generateContent API.
func (c *GeminiClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    start := time.Now()

    // Use request values or fall back to config defaults
    model := req.Model
    if model == "" {
        model = c.config.Model
    }

    temp := req.Temperature
    if temp == 0 {
        temp = c.config.Temperature
    }

    maxTokens := req.MaxTokens
    if maxTokens == 0 {
        maxTokens = c.config.MaxTokens
    }

    // Convert messages to Gemini format
    // CRITICAL: Extract system messages into separate systemInstruction field
    var systemInstruction *geminiContent
    contents := make([]geminiContent, 0, len(req.Messages))

    for _, m := range req.Messages {
        switch m.Role {
        case "system":
            // Gemini uses a separate systemInstruction field (not in contents array)
            if systemInstruction == nil {
                systemInstruction = &geminiContent{
                    Parts: []geminiPart{{Text: m.Content}},
                }
            } else {
                // Multiple system messages: append to existing
                systemInstruction.Parts = append(systemInstruction.Parts, geminiPart{Text: m.Content})
            }
        case "assistant":
            // CRITICAL: Gemini uses "model" instead of "assistant"
            contents = append(contents, geminiContent{
                Role:  "model",
                Parts: []geminiPart{{Text: m.Content}},
            })
        default:
            // "user" role stays the same
            contents = append(contents, geminiContent{
                Role:  m.Role,
                Parts: []geminiPart{{Text: m.Content}},
            })
        }
    }

    geminiReq := geminiRequest{
        SystemInstruction: systemInstruction,
        Contents:          contents,
        GenerationConfig: &geminiGenerationConfig{
            Temperature:     temp,
            MaxOutputTokens: maxTokens,
            StopSequences:   req.Stop,
        },
    }

    body, err := json.Marshal(geminiReq)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, model)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("x-goog-api-key", c.apiKey)

    // Retry logic with exponential backoff (same pattern as OpenAI/Anthropic)
    var resp *http.Response
    var lastErr error
    for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff: 1s, 4s, 9s, ...
            time.Sleep(time.Duration(attempt*attempt) * time.Second)
        }

        resp, lastErr = c.httpClient.Do(httpReq)
        if lastErr == nil {
            // Check for rate limiting (429)
            if resp.StatusCode == http.StatusTooManyRequests {
                resp.Body.Close()
                lastErr = ErrRateLimited
                continue
            }
            // Check for server errors (5xx) - retry
            if resp.StatusCode >= 500 {
                resp.Body.Close()
                lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
                continue
            }
            if resp.StatusCode == http.StatusOK {
                break
            }
        }

        if resp != nil {
            resp.Body.Close()
        }

        if ctx.Err() != nil {
            return nil, ErrTimeout
        }
    }

    if lastErr != nil {
        return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
    }
    defer resp.Body.Close()

    // Handle non-OK responses
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        var errResp geminiErrorResponse
        if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
            return nil, fmt.Errorf("gemini error: %s", errResp.Error.Message)
        }
        return nil, fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, string(bodyBytes))
    }

    var geminiResp geminiResponse
    if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if len(geminiResp.Candidates) == 0 {
        return nil, fmt.Errorf("no candidates returned")
    }

    // Extract text from all parts
    var content string
    for _, part := range geminiResp.Candidates[0].Content.Parts {
        content += part.Text
    }

    // Map finish reason to standard format
    finishReason := mapGeminiFinishReason(geminiResp.Candidates[0].FinishReason)

    return &CompletionResponse{
        Content:      content,
        Model:        geminiResp.ModelVersion,
        FinishReason: finishReason,
        Duration:     time.Since(start),
        Usage: TokenUsage{
            PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
            CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
            TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
        },
    }, nil
}

// mapGeminiFinishReason converts Gemini finish reasons to standard format.
func mapGeminiFinishReason(reason string) string {
    switch reason {
    case "STOP":
        return "stop"
    case "MAX_TOKENS":
        return "length"
    case "SAFETY", "RECITATION":
        return "content_filter"
    default:
        return "other"
    }
}
```

**Critério de Conclusão:**
- Conversão de mensagens funciona (system → systemInstruction, assistant → model)
- Retry logic implementada
- Finish reasons mapeados corretamente
- Token usage extraído

---

#### Task 2.6: Implementar Embed() [M]

**Arquivo:** `analyze/gemini.go`

```go
// Embed generates an embedding vector using Gemini's embedContent API.
// NOTE: Gemini's embedding endpoint does NOT return token usage information.
func (c *GeminiClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
    start := time.Now()

    model := req.Model
    if model == "" {
        model = c.config.EmbeddingModel
    }

    geminiReq := geminiEmbedRequest{
        Content: geminiContent{
            Parts: []geminiPart{{Text: req.Text}},
        },
    }

    body, err := json.Marshal(geminiReq)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    endpoint := fmt.Sprintf("%s/models/%s:embedContent", c.baseURL, model)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("x-goog-api-key", c.apiKey)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("embed request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("gemini embed error (status %d): %s", resp.StatusCode, string(bodyBytes))
    }

    var geminiResp geminiEmbedResponse
    if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if len(geminiResp.Embedding.Values) == 0 {
        return nil, fmt.Errorf("no embedding returned")
    }

    return &EmbeddingResponse{
        Embedding: geminiResp.Embedding.Values,
        Model:     model,
        Duration:  time.Since(start),
        // Usage is empty - Gemini embed endpoint doesn't return token counts
        Usage: TokenUsage{},
    }, nil
}
```

**Critério de Conclusão:** Embeddings gerados, `Usage` retornado zerado (documentado).

---

### Phase 3: Factory Integration (`analyze/factory.go`)

**Objetivo:** Conectar o novo cliente ao sistema de seleção de providers.

#### Task 3.1: Atualizar NewClient() [S]

**Arquivo:** `analyze/factory.go`
**Localização:** Dentro do switch de `cfg.LLM.Provider`

```go
case config.ProviderGemini:
    return NewGeminiClient(cfg.LLM.GeminiAPIKey, cfg.LLM.GeminiBaseURL, clientCfg)
```

**Critério de Conclusão:** `NewClient()` retorna `GeminiClient` quando provider é "gemini".

---

#### Task 3.2: Verificar NewEmbeddingClient() [S]

**Análise:** O Gemini suporta embeddings nativamente, então a lógica existente funciona:

```go
func NewEmbeddingClient(cfg *config.Config) (LLMClient, error) {
    // Se provider explícito para embeddings, usar ele
    if cfg.LLM.EmbeddingProvider != "" {
        embCfg := *cfg
        embCfg.LLM.Provider = config.Provider(cfg.LLM.EmbeddingProvider)
        return NewClient(&embCfg)
    }

    // Anthropic não tem embeddings - fallback para Ollama
    if cfg.LLM.Provider == config.ProviderAnthropic {
        clientCfg := ClientConfig{...}
        return NewOllamaClient(cfg.LLM.OllamaURL, clientCfg), nil
    }

    // OpenAI, Ollama, e GEMINI suportam embeddings nativamente
    return NewClient(cfg)  // ✅ Gemini passa por aqui
}
```

**Critério de Conclusão:** Nenhuma modificação necessária - verificar que funciona.

---

### Phase 4: Documentation & Testing

#### Task 4.1: Atualizar .env.example [S]

**Arquivo:** `.env.example`

```bash
# =============================================================================
# GEMINI CONFIGURATION
# =============================================================================

# Gemini API key (required for gemini provider)
# GEMINI_API_KEY=your-api-key-here

# Gemini base URL (optional, default: https://generativelanguage.googleapis.com/v1beta)
# GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta
```

---

#### Task 4.2: Atualizar README.md [S]

**Localizações a atualizar:**
1. Lista de providers no overview
2. Tabela de serviços externos
3. Seção de configuração

---

#### Task 4.3: Build e Verificação [S]

```bash
go build -o codemap .    # Deve compilar sem erros
go vet ./...             # Deve passar sem warnings
go fmt ./...             # Formatar código
```

---

## 5. File Changes Summary

| Arquivo | Tipo | Linhas | Descrição |
|---------|------|--------|-----------|
| `config/config.go` | Modificação | ~20 | Adicionar `ProviderGemini`, campos, env overrides, validation |
| `analyze/gemini.go` | **Novo** | ~320 | Implementação completa do `GeminiClient` |
| `analyze/factory.go` | Modificação | ~3 | Adicionar case `ProviderGemini` |
| `.env.example` | Modificação | ~10 | Seção de configuração Gemini |
| `README.md` | Modificação | ~10 | Documentar novo provider |

**Total estimado:** ~360 linhas de código

---

## 6. Risk Mitigation

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| **R1:** Safety filters bloqueando código válido | Média | Alto | Capturar `finishReason: "SAFETY"` e retornar erro descritivo |
| **R2:** Rate limiting agressivo | Média | Médio | Exponential backoff já implementado |
| **R3:** Mudanças na API v1beta | Baixa | Alto | Usar versionamento explícito no `baseURL` |
| **R4:** Embedding dimensions diferentes | Baixa | Médio | `text-embedding-004` retorna 768 dims (compatível com Ollama) |
| **R5:** Token usage ausente em embeddings | Certa | Baixo | Documentado, retornar `TokenUsage{}` zerado |

---

## 7. Testing Strategy

### 7.1. Testes de Compilação

```bash
# Build
go build -o codemap .

# Static analysis
go vet ./...

# Format check
go fmt ./...
```

### 7.2. Testes de Validação (sem API key)

```bash
# Deve falhar com mensagem clara
CODEMAP_LLM_PROVIDER=gemini ./codemap --explain --symbol main . 2>&1
# Esperado: "gemini_api_key required for gemini provider"
```

### 7.3. Testes de Integração (com API key)

```bash
# Configurar
export GEMINI_API_KEY="your-key-here"
export CODEMAP_LLM_PROVIDER="gemini"

# Test Ping (implícito no explain)
./codemap --explain --symbol "main" . 2>&1 | head -5
# Esperado: Não deve mostrar erro de conexão

# Test Complete
./codemap --explain --symbol "NewGraphBuilder" .
# Esperado: Explicação do símbolo

# Test Embed + Search
./codemap --embed .
./codemap --search "function that builds graph" .
# Esperado: Resultados de busca semântica
```

### 7.4. Checklist de Validação

- [ ] `go build` compila sem erros
- [ ] `go vet ./...` sem warnings
- [ ] `go fmt ./...` sem alterações
- [ ] Sem API key → erro claro na validação
- [ ] `--explain` funciona com Gemini
- [ ] `--embed` gera embeddings com Gemini
- [ ] `--search` retorna resultados após embedding
- [ ] Cache funciona corretamente com Gemini

---

## 8. Success Metrics

| Métrica | Critério | Como Medir |
|---------|----------|------------|
| **Funcionalidade** | Usuário pode usar `--explain`, `--summarize`, `--embed`, `--search` com Gemini | Testes manuais |
| **Confiabilidade** | Taxa de erro < 1% em chamadas válidas | Logs de erro |
| **Performance** | Latência similar a OpenAI (~2-5s para completion) | Timing nos logs |
| **Manutenibilidade** | Código segue padrões dos clientes existentes | Code review |

---

## 9. Implementation Checklist

### Phase 1: Configuration
- [ ] 1.1 Adicionar `ProviderGemini` ao enum
- [ ] 1.2 Adicionar `GeminiAPIKey`, `GeminiBaseURL` no `LLMConfig`
- [ ] 1.3 Atualizar `applyEnvOverrides()` para `GEMINI_API_KEY`, `GEMINI_BASE_URL`
- [ ] 1.4 Atualizar `Validate()` para validar `GeminiAPIKey`

### Phase 2: GeminiClient
- [ ] 2.1 Criar struct `GeminiClient` e `NewGeminiClient()`
- [ ] 2.2 Definir tipos de request/response (8 structs)
- [ ] 2.3 Implementar `Name()` e `Ping()`
- [ ] 2.4 Implementar `Models()`
- [ ] 2.5 Implementar `Complete()` com conversão de mensagens e retry
- [ ] 2.6 Implementar `Embed()`

### Phase 3: Factory Integration
- [ ] 3.1 Adicionar case `ProviderGemini` em `NewClient()`
- [ ] 3.2 Verificar `NewEmbeddingClient()` funciona com Gemini

### Phase 4: Documentation & Testing
- [ ] 4.1 Atualizar `.env.example`
- [ ] 4.2 Atualizar `README.md`
- [ ] 4.3 Executar build e verificação
- [ ] 4.4 Executar testes de integração (se API key disponível)

---

## 10. Assumptions

1. O Gemini API está acessível com uma API Key válida do Google AI Studio
2. O formato `v1beta` do Gemini API permanecerá estável durante a implementação
3. A lógica de retry existente é suficiente para lidar com rate limiting
4. O modelo `text-embedding-004` (768 dims) é compatível com o sistema de vetores existente

---

## 11. References

- [Google Gemini API Documentation](https://ai.google.dev/docs)
- [Gemini API Reference - generateContent](https://ai.google.dev/api/rest/v1beta/models/generateContent)
- [Gemini API Reference - embedContent](https://ai.google.dev/api/rest/v1beta/models/embedContent)
- Implementação de referência: `analyze/openai.go`
