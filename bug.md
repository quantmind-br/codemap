# Bug Analysis Report: Ferramentas MCP em Processamento Eterno

## 1\. Executive Summary

A causa mais provável para as ferramentas do servidor MCP (`explain_symbol` e `summarize_module`) ficarem em **processamento eterno** (requisição não concluída/timeout) é um **erro no loop de retry ou na manipulação do contexto (`context.Context`)** dentro da lógica de comunicação com os provedores de LLM. Considerando que as funcionalidades na CLI funcionam, o problema está na execução via servidor MCP, onde o contexto da requisição pode estar sendo cancelado ou expirado, mas o loop de retry não o respeita corretamente, ou o tempo de espera total excede o timeout do cliente HTTP.

-----

## 2\. Bug Description and Context (from `User Task`)

  * **Observed Behavior:** As ferramentas do servidor MCP `explain_symbol` e `summarize_module`, quando acionadas, ficam processando eternamente e a requisição nunca é concluída.
  * **Expected Behavior:** A requisição deveria ser concluída dentro do tempo limite configurado (`config.LLM.Timeout`) ou retornar um erro claro (ex: timeout, erro de API).
  * **Steps to Reproduce (STR):** Acionar as ferramentas `explain_symbol` ou `summarize_module` via cliente MCP.
  * **Environment (if provided):** Go project (`codemap`). O problema é exclusivo da versão *server/MCP*, enquanto a versão *CLI* funciona corretamente.
  * **Error Messages (if any):** Não especificado, mas o sintoma é um timeout silencioso (processamento eterno).

-----

## 3\. Code Execution Path Analysis

O problema de "processamento eterno" (hanging/timeout) em um servidor web/API que faz chamadas externas (LLM) geralmente ocorre em um dos seguintes pontos:

1.  **Bloqueio Síncrono:** Uma chamada de rede bloqueando indefinidamente sem respeitar o timeout do contexto.
2.  **Retry Infinito/Longo:** O loop de retry de uma chamada de rede está configurado para tentar novamente mesmo após o timeout do contexto pai ter expirado.

As duas ferramentas problemáticas (`explain_symbol` e `summarize_module`) utilizam o mesmo fluxo principal: `handle*()` $\rightarrow$ `analyze.NewClient()` $\rightarrow$ `client.Complete()`.

### 3.1. Entry Point(s) and Initial State

  * **Entry Point:** `mcp/main.go`: `handleExplainSymbol` ou `handleSummarizeModule`.
  * **Initial State:** O `context.Context` (`ctx`) da requisição MCP é criado (pode ter um timeout implícito ou ser o contexto padrão do servidor).

### 3.2. Key Functions/Modules/Components in the Execution Path

| Componente | Função/Caminho | Responsabilidade Presumida |
| :--- | :--- | :--- |
| **MCP Handler** | `mcp/main.go` | Carrega `config`, cria `LLMClient`, inicia `reqCtx` com timeout. |
| **Client Factory** | `analyze/factory.go` | Instancia o cliente LLM (`OllamaClient`, `OpenAIClient`, etc.) a partir da config. |
| **Client Adapter** | `analyze/*client*.go` | Executa a chamada `Complete()` HTTP para a API externa, incluindo retry logic. |
| **Context** | `context.Context` | Limita o tempo total de execução. O timeout configurado é de `60s` por padrão (`config/config.go:DefaultConfig().LLM.Timeout`). |

### 3.3. Execution Flow Tracing (Exemplo: `OpenAIClient.Complete`)

O fluxo de execução é crucial aqui, especialmente o loop de retry e a forma como o `context.Context` é usado. O `handleExplainSymbol` cria um contexto com timeout:

```go
// mcp/main.go:1020 (handleExplainSymbol)
reqCtx, reqCancel := context.WithTimeout(ctx, time.Duration(cfg.LLM.Timeout)*time.Second)
defer reqCancel()

resp, err := client.Complete(reqCtx, &analyze.CompletionRequest{...})
```

A chamada `client.Complete(reqCtx, ...)` (por exemplo, em `analyze/openai.go`):

```go
// analyze/openai.go:167 (Complete - Retry Logic)
// Loop: for attempt := 0; attempt <= c.config.MaxRetries; attempt++ { // MaxRetries defaults to 3 (4 attempts total)
	// 1. New Request: httpReq, err := http.NewRequestWithContext(ctx, "POST", ...)
	// 2. Do Request: resp, lastErr = c.httpClient.Do(httpReq)
	// 3. Check for Timeout/Cancellation: if ctx.Err() != nil { return nil, ErrTimeout }
	// ...
// }
```

**Análise Crítica: `GeminiClient.Complete` (Suspeito)**

O `GeminiClient` introduzido em `analyze/gemini.go` tem um erro na lógica de retry que é uma causa comum de *hangups* em Go.

```go
// analyze/gemini.go:217 (Complete - inside retry loop)
		// Check context before each attempt
		if ctx.Err() != nil { // <-- CORRETO, checa antes da tentativa
			return nil, ErrTimeout
		}

		// Create new request for each attempt (body reader is consumed after each request)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body)) // <-- OK
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-goog-api-key", c.apiKey)

		resp, lastErr = c.httpClient.Do(httpReq) // <-- BLOQUEIA: Usa o contexto (ctx)

		if lastErr != nil {
			// Network error - retry
			if ctx.Err() != nil { // <-- CORRETO, checa depois da falha
				return nil, ErrTimeout
			}
			continue
		}
// ...
```

O `OpenAIClient` e o `AnthropicClient` usam a variável `c.httpClient.Timeout` em vez do contexto para `http.Client.Do()`. A menos que o `c.httpClient` seja inicializado com o `reqCtx` (o que não ocorre), o timeout primário da requisição é determinado pelo `http.Client.Timeout` (`ClientConfig.Timeout`, default 60s), e o `reqCtx` da MCP é usado apenas para cancelamento.

O `GeminiClient` (e `AnthropicClient` em `analyze/anthropic.go`) *re-cria* o `http.NewRequestWithContext(ctx, ...)` em cada retry, o que é correto para usar o `reqCtx` da MCP.

O ponto crítico de falha é a **lógica de *backoff* do retry**:

```go
// analyze/gemini.go:213 (Complete)
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second) // <-- Ex: 1s, 4s, 9s...
		}
        // ... faz a requisição
	}
```

Com `MaxRetries=3`, o tempo de *backoff* é de $1s + 4s + 9s = 14s$. Isso é aceitável, mas se o `http.Client` for configurado com um timeout longo, o tempo total pode exceder o timeout da requisição MCP.

**O real suspeito é a manipulação de timeouts do `http.Client` (analyze/client.go) em conjunto com a lógica de retry:**

O `http.Client` é criado em `analyze/gemini.go:34`:

```go
// analyze/gemini.go:34
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
```

O `cfg.Timeout` padrão é **60 segundos** (`config/config.go:DefaultConfig`). O loop de retry tentará **4 vezes** (0 a 3) com até $14s$ de *backoff* total. O tempo total de *wait* (4 \* $60s$ (timeout) + $14s$ (backoff)) é muito longo (aprox. $4$ minutos) e o cliente MCP (que tipicamente usa um timeout muito menor) falhará, mas o loop de retry no servidor continuará rodando até que o `reqCtx` finalmente expire.

O **problema original de "processando eternamente"** provavelmente se deve ao cliente LLM não honrando o `ctx.Err() != nil` **após uma falha de rede inicial**, ou o cliente não sendo configurado com o `reqCtx` *e* o `httpClient.Timeout` sendo muito longo.

**Em `analyze/anthropic.go`:**

```go
// analyze/anthropic.go:174 (Complete)
		resp, lastErr = c.httpClient.Do(httpReq)
		// ...
		if lastErr != nil {
			// ...
		}
		if resp != nil {
			resp.Body.Close()
		}

		if ctx.Err() != nil { // <-- Checagem de cancelamento/timeout do contexto
			return nil, ErrTimeout
		}
```

A única maneira de este loop travar é se `c.httpClient.Do(httpReq)` **retornar `lastErr == nil`** (conexão bem-sucedida com erro de status, ex: 400 Bad Request) mas o `resp.StatusCode` **não for `http.StatusOK` ou `http.StatusTooManyRequests`**, e o loop **quebrar o `if lastErr == nil`**, mas **não houver `break`**, fazendo o loop rodar as 4 tentativas, consumindo muito tempo.

**Identificando o Bug Lógico no Retry:**

Em `analyze/anthropic.go:176`:

```go
// analyze/anthropic.go:176
		if lastErr == nil {
			// Check for rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				lastErr = ErrRateLimited
				continue
			}
			if resp.StatusCode == http.StatusOK {
				break // <-- SUCESSO! Sai do loop.
			}
		}

		if resp != nil {
			resp.Body.Close()
		}
		// Se lastErr != nil (erro de rede) ou if (sucesso, mas não 200/429), o loop continua.

		if ctx.Err() != nil {
			return nil, ErrTimeout // <-- Esta é a única checagem de timeout no final da iteração.
		}
```

Se a API Anthropic retornar um `500 Internal Server Error` ou um erro `400 Bad Request` que **não seja `429`**, o código **não cai no `break`** (pois `resp.StatusCode != http.StatusOK`) e o `lastErr` **é `nil`** (pois o `http.Client` recebeu uma resposta). O loop continua para a próxima iteração **sem um `break`** ou **retorno de erro imediato**.

O loop tentará novamente 4 vezes, com `time.Sleep` somado, resultando em um atraso significativo antes de finalmente sair do loop no final de 4 tentativas e retornar um erro genérico:

```go
// analyze/anthropic.go:193
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}
	// ... (lógica de erro para status != 200 fora do loop)
```

Se o erro for `400 Bad Request`, o loop continuará 4 vezes sem necessidade, resultando no atraso, mas **não no travamento eterno** (o loop termina).

**O verdadeiro travamento eterno (se ocorrer) só pode acontecer se o `ctx.Err() != nil` nunca for verdadeiro ou se o `http.Client.Do()` travar indefinidamente, o que é mitigado pelo `c.httpClient.Timeout` de 60s.**

**Revisitando `GeminiClient`:**

Em `analyze/gemini.go:230-244`, a lógica de `GeminiClient` é mais robusta e **inclui `continue` e `return`** para diferentes status HTTP, minimizando o risco de retries desnecessários:

```go
// analyze/gemini.go:230 (GeminiClient.Complete - switch statement)
		switch {
		case resp.StatusCode == http.StatusOK:
			lastErr = nil
		case resp.StatusCode == http.StatusNotFound:
			resp.Body.Close()
			return nil, ErrModelNotFound // <-- RETORNO IMEDIATO: NÃO GERA MAIS RETRIES
		case resp.StatusCode == http.StatusTooManyRequests:
			resp.Body.Close()
			lastErr = ErrRateLimited
			continue // <-- RETRY
		case resp.StatusCode >= 500:
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue // <-- RETRY
		default:
			lastErr = nil // <-- 4xx client errors that are not 404/429. Will exit loop with break below.
		}
		break // <-- Sai do loop se for OK ou 4xx não-retryable.
	}
```

Esta lógica é mais correta que a de Anthropic e deve sair do loop rapidamente.

### 3.4. Data State and Flow Analysis

O fluxo de dados (`CompletionRequest` $\rightarrow$ `LLMClient` $\rightarrow$ `CompletionResponse`) é padrão. A falha não parece ser de manipulação de dados.

**Conclusão da Execução:** O sintoma sugere que o tempo de execução excede o tempo limite do cliente MCP que faz a chamada para o `codemap-mcp`, ou que o `codemap-mcp` está gastando muito tempo em **retries desnecessários** ou **sleeps de backoff** quando encontra um erro não-retryable (como um `400 Bad Request` na API Anthropic).

-----

## 4\. Potential Root Causes and Hypotheses

### 4.1. Hypothesis 1: Retry Loop Não Honra Contexto em Erros de Status (OpenAI/Anthropic)

  - **Rationale/Evidence:** Em `analyze/anthropic.go`, um `resp.StatusCode` diferente de `200` ou `429` (ex: `400` Bad Request) resulta em `lastErr == nil`, mas não quebra o loop de retry. O loop continua por 4 iterações, introduzindo um atraso de $14$ segundos (no backoff), somado a 4 timeouts de $60$ segundos (se o timeout do cliente for usado). Este comportamento, embora não seja um *loop infinito*, causa um **atraso significativo** que excede o timeout do cliente chamador do MCP, fazendo a requisição parecer "eterna" (até o cliente MCP desistir).
  - **Code (if relevant):**
    ```go
    // analyze/anthropic.go:176
    if lastErr == nil {
        // ... check for 429 and 200 (break)
    }
    // No explicit 'else if resp.StatusCode != 200' break/return for other codes.
    // The logic continues, leading to unnecessary retries and time.Sleep.
    ```
  - **How it leads to the bug:** Um erro de API (ex: Bad Request por prompt inválido) dispara 4 tentativas de retry com backoff, fazendo o servidor MCP gastar no mínimo $14s$ (backoff) + $4 \times (\text{latência de rede})$ antes de falhar fora do loop e retornar o erro final. Se o cliente MCP tiver um timeout menor, ele desiste e a requisição no servidor continua em background até o final do loop/timeout.

### 4.2. Hypothesis 2: LLM Client Configurado com Timeout Excessivo

  - **Rationale/Evidence:** O `ClientConfig.Timeout` padrão é de `60 * time.Second` (em `config/config.go`). O `MaxRetries` padrão é `3`. O tempo total de espera do cliente LLM no servidor MCP é de $4 \times (\text{Timeout do HTTP Client}) + \text{Backoff}$. Se o timeout do cliente MCP for menor que, digamos, $30$ segundos, o servidor MCP fará um trabalho considerável (e lento) antes de desistir, mas a resposta nunca chegará ao cliente.
  - **Code (if relevant):**
    ```go
    // analyze/client.go:88
    Timeout: 60 * time.Second, // Default timeout
    // analyze/client.go:89
    MaxRetries: 3, // Default max retries (4 attempts total)
    ```
  - **How it leads to the bug:** O tempo de espera configurado na lógica de retry é muito longo para a expectativa de resposta do cliente MCP.

### 4.3. Most Likely Cause(s)

**Hypothesis 1** é a causa mais provável para o **atraso excessivo** que faz a requisição parecer travada, especialmente quando o LLM retorna um erro imediato (400) que não é retryable (429) ou sucesso (200). O loop de retry em `analyze/anthropic.go` (e possivelmente `openai.go`, que tem uma estrutura similar) deve ser corrigido para evitar retries em erros de cliente (4xx) não relacionados a rate limiting.

-----

## 5\. Supporting Evidence from Code (if `File Structure` is provided)

A lógica em `analyze/anthropic.go` é o exemplo mais claro da falha de *break* no loop:

```go
// analyze/anthropic.go:173
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// ... (sleep de backoff)
		resp, lastErr = c.httpClient.Do(httpReq)
		if lastErr == nil {
			// Check for rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				lastErr = ErrRateLimited
				continue
			}
			if resp.StatusCode == http.StatusOK {
				break // <-- Só quebra aqui (sucesso)
			}
			// *** PROBLEMA: Outros status 4xx/5xx (onde lastErr é nil) continuam o loop e o sleep
		}
		// ...
	}
	
	// A única lógica que trata o erro 400 está fora do loop:
	// analyze/anthropic.go:198-202
	if resp.StatusCode != http.StatusOK {
		// ... lê body e retorna erro
	}
```

## 6\. Recommended Steps for Debugging and Verification

O foco deve ser garantir que o loop de retry seja estrito quanto ao que é retryable.

  * **Logging:**

      * Adicionar logging detalhado no loop de retry de **todos os clientes** (`OpenAIClient`, `AnthropicClient`, `GeminiClient`, `OllamaClient`).
          * Logar o `attempt` atual: `logger.debug("Attempt %d for %s", attempt, model)`
          * Logar o `resp.StatusCode` e o `lastErr` após `c.httpClient.Do(httpReq)`.
          * Logar o valor de `ctx.Err()` antes de `return nil, ErrTimeout` para confirmar o cancelamento.
      * Em `mcp/main.go` nos handlers, logar o valor de `cfg.LLM.Timeout` e `cfg.LLM.MaxRetries` antes de iniciar a requisição.

  * **Breakpoints:**

      * Setar um breakpoint em `analyze/anthropic.go:176` (dentro do `if lastErr == nil`) para inspecionar o `resp.StatusCode` quando for diferente de 200 e 429.
      * Setar um breakpoint em `time.Sleep(...)` em todos os clientes para verificar se o backoff é executado em erros não-retryable.

  * **Test Scenarios/Requests:**

      * **Cenário 1 (400 Bad Request):** Use uma API key válida (Anthropic) e um prompt conhecido por falhar (ex: prompt muito longo ou formato inválido - se possível simular) e observe se o loop faz 4 retries desnecessários.
      * **Cenário 2 (Timeout):** Configure `cfg.LLM.Timeout = 1s` e `cfg.LLM.MaxRetries = 1` e verifique se o erro retorna rapidamente.

  * **Refinamento do Código (Correção Sugerida):**

      * Modificar a lógica de retry para quebrar o loop (usando `return` ou `break`) imediatamente em *qualquer* `resp.StatusCode` que não seja `429` (Rate Limit) ou `5xx` (Server Error). Códigos `4xx` (exceto 429) são erros de cliente e não devem ser repetidos.

<!-- end list -->

```go
// Correção Sugerida para analyze/anthropic.go:176
// ...
		if lastErr == nil {
			// Check for retryable HTTP codes
			switch {
			case resp.StatusCode == http.StatusOK:
				break // Success
			case resp.StatusCode == http.StatusTooManyRequests:
				resp.Body.Close()
				lastErr = ErrRateLimited
				continue // Retry for Rate Limit
			case resp.StatusCode >= 500:
				resp.Body.Close()
				lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
				continue // Retry for Server Error
			default:
				// All other codes (e.g., 400 Bad Request) are not retryable client errors
				break // Exit the retry loop to handle the error outside
			}
		}
// ...
```

## 7\. Bug Impact Assessment

O bug impede o uso das funcionalidades de LLM do servidor MCP, forçando o cliente a aguardar (e possivelmente fazer timeout) por um período de tempo desnecessariamente longo ($4$ minutos se o timeout do cliente HTTP for $60$s) em caso de erro. Isso degrada a experiência do usuário e desperdiça recursos do servidor.

## 8\. Assumptions Made During Analysis

  * A versão CLI está usando a mesma lógica de `analyze/` package, mas possivelmente com uma configuração de retry/timeout diferente.
  * O cliente MCP tem um timeout menor que o tempo total de execução do loop de retry do servidor.
  * O sintoma de "processando eternamente" é, na verdade, um timeout prolongado que o cliente MCP não gerencia bem.
  * Os `GeminiClient` e `OpenAIClient` têm lógica de retry semelhante ao `AnthropicClient` para o loop principal de decisão de `break`/`continue`/`retry`.

## 9\. Open Questions / Areas for Further Investigation

  * Qual é o provedor LLM configurado (`config.LLM.Provider`) quando o bug ocorre? (Gemini, Anthropic, OpenAI ou Ollama?)
  * Qual é o timeout configurado no cliente que faz a chamada ao `codemap-mcp`?
  * O erro de status (ex: 400) está ocorrendo devido ao tamanho/formato do prompt (relacionado ao `explain_symbol`/`summarize_module` em particular) ou é um erro de credenciais/configuração?