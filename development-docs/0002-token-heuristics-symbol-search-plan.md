# Refactoring/Design Plan: Token Heuristics, `get_symbol` Tool e Melhorias MCP

> **Versão:** 2.0 (Otimizado)
> **Data:** 2025-12-03
> **Status:** Pronto para implementação

---

## 1. Executive Summary & Goals

O objetivo principal é **melhorar a utilidade do `codemap` para agentes LLM**, focando em:

1. **Context Management (Tokens):** Estimativa de tokens por arquivo com alertas visuais (P0)
2. **Semantic Search (`get_symbol`):** Busca precisa de funções/tipos por nome (P0)
3. **MCP Enhancement:** Exposição do modo `--api` e refatoração de path validation (P1)

### Mudanças-Chave vs. Plano Original

| Aspecto | Plano Original | Plano Otimizado |
|---------|----------------|-----------------|
| Token Heuristics | Map `TokenRatios` por linguagem | Ratio fixo universal (3.5 chars/token) |
| Captura de Linha | Mencionada como "questão aberta" | **Fase 0 obrigatória** - Prerequisito |
| `SymbolInfo` | Novo struct com campos duplicados | Reutiliza `FuncInfo`/`TypeInfo` + campo `Line` |
| Formato de Saída | "ASCII formatado" (vago) | Especificação exata definida |

---

## 2. Current Situation Analysis

### Arquitetura Existente

```
scanner/types.go     → FileInfo, FuncInfo, TypeInfo, FileAnalysis
scanner/walker.go    → ScanFiles(), ScanForDeps()
scanner/grammar.go   → AnalyzeFile(), funcCapture, typeCapture
render/tree.go       → Tree()
render/api.go        → APIView()
mcp/main.go          → 7 handlers existentes
```

### Gap Crítico Identificado

**`funcCapture` e `typeCapture` NÃO capturam linha de definição:**

```go
// scanner/grammar.go:256-261 - ATUAL
type funcCapture struct {
    name     string
    params   string
    result   string
    receiver string
    // FALTA: line int
}
```

Isso **bloqueia** a implementação de `get_symbol` com localização.

---

## 3. Proposed Solution

### 3.1. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI / MCP Layer                          │
├─────────────────────────────────────────────────────────────────┤
│  main.go         mcp/main.go                                    │
│     │               │                                            │
│     │               ├── handleGetSymbol (NEW)                   │
│     │               ├── handleGetDependencies (mode param)      │
│     │               └── validatePath (refactored)               │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                        Scanner Layer                             │
├─────────────────────────────────────────────────────────────────┤
│  types.go                                                        │
│     ├── FileInfo { ..., Tokens int }  (modified)                │
│     ├── FuncInfo { ..., Line int }    (modified)                │
│     └── TypeInfo { ..., Line int }    (modified)                │
│                                                                  │
│  walker.go                                                       │
│     └── ScanFiles() → calcula Tokens                            │
│                                                                  │
│  grammar.go                                                      │
│     ├── funcCapture { ..., line int } (modified)                │
│     └── typeCapture { ..., line int } (modified)                │
│                                                                  │
│  symbol.go (NEW)                                                 │
│     └── SearchSymbols(analyses, query) → []SymbolMatch          │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                        Render Layer                              │
├─────────────────────────────────────────────────────────────────┤
│  tree.go                                                         │
│     └── Tree() → exibe tokens + warnings                        │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2. Data Model Changes

#### `scanner/types.go`

```go
// MODIFICAÇÃO: FileInfo
type FileInfo struct {
    Path    string `json:"path"`
    Size    int64  `json:"size"`
    Ext     string `json:"ext"`
    Tokens  int    `json:"tokens,omitempty"`  // NEW: Estimativa de tokens
    IsNew   bool   `json:"is_new,omitempty"`
    Added   int    `json:"added,omitempty"`
    Removed int    `json:"removed,omitempty"`
}

// MODIFICAÇÃO: FuncInfo
type FuncInfo struct {
    Name       string `json:"name"`
    Signature  string `json:"signature,omitempty"`
    Receiver   string `json:"receiver,omitempty"`
    IsExported bool   `json:"exported,omitempty"`
    Line       int    `json:"line,omitempty"`  // NEW: Linha de definição
}

// MODIFICAÇÃO: TypeInfo
type TypeInfo struct {
    Name       string   `json:"name"`
    Kind       TypeKind `json:"kind"`
    Fields     []string `json:"fields,omitempty"`
    Methods    []string `json:"methods,omitempty"`
    IsExported bool     `json:"exported,omitempty"`
    Line       int      `json:"line,omitempty"`  // NEW: Linha de definição
}

// NEW: Constante para estimativa de tokens
const CharsPerToken = 3.5  // Conservador para maioria dos tokenizers BPE

// NEW: Threshold para warning
const LargeFileTokens = 8000

// NEW: Função helper
func EstimateTokens(size int64) int {
    return int(float64(size) / CharsPerToken)
}
```

#### `scanner/grammar.go`

```go
// MODIFICAÇÃO: funcCapture
type funcCapture struct {
    name     string
    params   string
    result   string
    receiver string
    line     int  // NEW
}

// MODIFICAÇÃO: typeCapture
type typeCapture struct {
    name   string
    kind   TypeKind
    fields string
    line   int  // NEW
}
```

#### `scanner/symbol.go` (NEW)

```go
package scanner

// SymbolQuery representa os filtros de busca
type SymbolQuery struct {
    Name string   // Substring match (case-insensitive)
    Kind string   // "function", "type", "all"
    File string   // Filtrar por arquivo específico (opcional)
}

// SymbolMatch representa um símbolo encontrado
type SymbolMatch struct {
    Name      string `json:"name"`
    Kind      string `json:"kind"`      // "function" ou "type"
    Signature string `json:"signature"` // Para funções
    TypeKind  string `json:"type_kind"` // Para tipos (struct, class, etc.)
    File      string `json:"file"`
    Line      int    `json:"line"`
    Exported  bool   `json:"exported"`
}

// SearchSymbols busca símbolos nas análises existentes
func SearchSymbols(analyses []FileAnalysis, query SymbolQuery) []SymbolMatch
```

#### `mcp/main.go`

```go
// MODIFICAÇÃO: DepsInput
type DepsInput struct {
    Path   string `json:"path" jsonschema:"Path to the project directory"`
    Detail int    `json:"detail,omitempty" jsonschema:"0=names, 1=signatures, 2=full"`
    Mode   string `json:"mode,omitempty" jsonschema:"deps (default), api"`  // NEW
}

// NEW: SymbolInput
type SymbolInput struct {
    Path string `json:"path" jsonschema:"Path to the project directory"`
    Name string `json:"name" jsonschema:"Symbol name to search (substring match)"`
    Kind string `json:"kind,omitempty" jsonschema:"function, type, or all (default)"`
    File string `json:"file,omitempty" jsonschema:"Filter to specific file"`
}
```

---

## 4. Detailed Action Plan

### Phase 0: Line Capture Fix (BLOCKER)

**Objetivo:** Corrigir o gap crítico que impede `get_symbol` de reportar localização.

| Task | Arquivo | Mudança | Critério de Conclusão |
|------|---------|---------|----------------------|
| 0.1 | `scanner/grammar.go` | Adicionar campo `line int` em `funcCapture` e `typeCapture` | Structs atualizados |
| 0.2 | `scanner/grammar.go` | Em `AnalyzeFile`, capturar linha via `node.StartPosition().Row + 1` | Linha capturada durante parsing |
| 0.3 | `scanner/types.go` | Adicionar campo `Line int` em `FuncInfo` e `TypeInfo` | Structs atualizados |
| 0.4 | `scanner/grammar.go` | Propagar linha em `Build()` de `funcCapture` e `typeCapture` | `FuncInfo.Line` e `TypeInfo.Line` preenchidos |

**Implementação `0.2`:**
```go
// Em AnalyzeFile, no loop de captures:
for i, capture := range match.Captures {
    captureName := query.CaptureNames()[capture.Index]
    text := capture.Node.Utf8Text(content)
    startLine := int(capture.Node.StartPosition().Row) + 1  // 1-indexed

    if strings.HasPrefix(captureName, "func.") {
        handleFuncCapture(funcBuilders, match.ID(), captureName, text, startLine)
    }
    // similar para types
}
```

---

### Phase 1: Token Estimation (P0)

**Objetivo:** Fornecer visibilidade de context window para LLMs.

| Task | Arquivo | Mudança | Critério de Conclusão |
|------|---------|---------|----------------------|
| 1.1 | `scanner/types.go` | Adicionar `Tokens int` em `FileInfo`, `CharsPerToken`, `LargeFileTokens`, `EstimateTokens()` | Constantes e helper definidos |
| 1.2 | `scanner/walker.go` | Em `ScanFiles`, calcular e preencher `Tokens` para cada arquivo | `FileInfo.Tokens` preenchido |
| 1.3 | `render/tree.go` | Exibir tokens no formato `~Xk` e `⚠` para arquivos > 8k tokens | Output visual atualizado |
| 1.4 | `render/tree.go` | Adicionar total de tokens no header | Header exibe `Tokens: ~Xk` |

**Formato de Saída (Task 1.3):**
```
├── scanner/
│   ├── grammar.go        (~4.2k tokens)
│   ├── walker.go         (~1.8k tokens)
│   └── huge_file.go      (~12k tokens) ⚠
```

**Formato Header (Task 1.4):**
```
╭──────────────────── codemap ────────────────────╮
│ Files: 42 | Size: 156KB | Tokens: ~45k          │
│ Top Extensions: .go (28), .md (8), .yaml (4)    │
╰─────────────────────────────────────────────────╯
```

---

### Phase 2: Symbol Search - `get_symbol` (P0)

**Objetivo:** Permitir busca semântica precisa de símbolos.

| Task | Arquivo | Mudança | Critério de Conclusão |
|------|---------|---------|----------------------|
| 2.1 | `scanner/symbol.go` | Criar arquivo com `SymbolQuery`, `SymbolMatch`, `SearchSymbols()` | Arquivo criado e compila |
| 2.2 | `scanner/symbol.go` | Implementar `SearchSymbols()` que filtra `[]FileAnalysis` | Retorna matches filtrados |
| 2.3 | `mcp/main.go` | Adicionar `SymbolInput` struct | Struct definido |
| 2.4 | `mcp/main.go` | Implementar `handleGetSymbol()` | Handler funcional |
| 2.5 | `mcp/main.go` | Registrar tool `get_symbol` no servidor MCP | Tool aparece em `status` |

**Formato de Saída `get_symbol`:**
```
=== Symbol Search: "Config" ===

Found 3 matches:

  scanner/types.go:18
  ├─ type Config struct
  └─ Fields: Root, Mode, Files

  scanner/grammar.go:45
  ├─ func NewConfig(path string) *Config
  └─ Signature: NewConfig(path string) *Config

  mcp/main.go:120
  ├─ func (c *Config) Validate() error
  └─ Receiver: *Config

───────────────────────────────────
Matches: 3 (1 type, 2 functions)
```

---

### Phase 3: MCP Enhancements (P1)

**Objetivo:** Expor modo API e centralizar validação.

| Task | Arquivo | Mudança | Critério de Conclusão |
|------|---------|---------|----------------------|
| 3.1 | `mcp/main.go` | Adicionar `Mode string` em `DepsInput` | Struct atualizado |
| 3.2 | `mcp/main.go` | Em `handleGetDependencies`, usar `render.APIView` quando `mode="api"` | Modo API funcional |
| 3.3 | `mcp/main.go` | Criar `validatePath(path string) (string, error)` | Helper definido |
| 3.4 | `mcp/main.go` | Refatorar handlers para usar `validatePath` | Zero duplicação de código |

**Implementação `validatePath`:**
```go
func validatePath(path string) (string, error) {
    if path == "" {
        return "", fmt.Errorf("path is required")
    }
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    if _, err := os.Stat(absPath); os.IsNotExist(err) {
        return "", fmt.Errorf("path does not exist: %s", absPath)
    }
    return absPath, nil
}
```

---

## 5. Dependency Graph

```
Phase 0 (Line Capture)
    │
    ├──────────────────────┐
    ▼                      ▼
Phase 1 (Tokens)     Phase 2 (get_symbol)
    │                      │
    └──────────┬───────────┘
               ▼
         Phase 3 (MCP)
```

**Nota:** Phases 1 e 2 podem ser desenvolvidas em paralelo após Phase 0.

---

## 6. Risk Mitigation

| Risco | Probabilidade | Impacto | Mitigação |
|-------|---------------|---------|-----------|
| R1: Captura de linha quebra parsing existente | Baixa | Alto | Testes manuais em múltiplas linguagens antes de merge |
| R2: Token ratio impreciso | Alta | Baixo | Usar `~` (aproximação) e focar no warning > 8k |
| R3: `SearchSymbols` lento em projetos grandes | Média | Médio | Filtrar por arquivo primeiro se `query.File` especificado |
| R4: Breaking change em JSON output | Média | Alto | Campos novos são `omitempty`, não quebra consumers existentes |

---

## 7. Validation Criteria

### Phase 0
- [ ] `./codemap --deps .` exibe funções/tipos com linha no JSON (`--json`)
- [ ] Linha é 1-indexed e corresponde à definição real

### Phase 1
- [ ] `./codemap .` exibe `(~Xk tokens)` após cada arquivo
- [ ] Arquivos > 8k tokens mostram `⚠`
- [ ] Header mostra total de tokens do projeto

### Phase 2
- [ ] `codemap-mcp` lista `get_symbol` em `status`
- [ ] `get_symbol(path=".", name="Scan")` retorna todas funções/tipos com "Scan"
- [ ] Output inclui `file:line` para cada match

### Phase 3
- [ ] `get_dependencies(path=".", mode="api")` retorna output idêntico a `render.APIView`
- [ ] Nenhum handler em `mcp/main.go` tem `filepath.Abs` duplicado

---

## 8. Questões Resolvidas

| Questão Original | Resolução |
|------------------|-----------|
| `get_symbol` como função separada ou modo em `ScanForDeps`? | **Função separada** em `scanner/symbol.go` que consome output de `ScanForDeps` |
| Onde extrair linha de definição? | Em `handleFuncCapture`/`handleTypeCapture` via `node.StartPosition().Row + 1` |
| Test Mapping (P2): nome de arquivo ou conteúdo? | **Apenas nome de arquivo** (removido do escopo atual - P2 postergado) |
| Token ratio por linguagem ou fixo? | **Fixo (3.5)** - Simplificação justificada por baixa variação em tokenizers BPE |

---

## 9. Out of Scope (P2+)

Removido do escopo atual para manter foco:

- Test Mapping (`TestPattern`, `TestMapping`)
- JSON output format para MCP (`--format=json`)
- CLI `--format=json` para tree mode
- Cross-file reference tracking (`ImportedBy`)

Estes itens podem ser implementados em iteração futura.
