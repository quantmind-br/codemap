# PLAN.md - Enhanced Code Analysis Features

## Executive Summary

Este plano descreve a implementação de recursos avançados de análise de código para o `codemap`, focando em:
1. **Assinaturas de funções completas** (parâmetros e retornos)
2. **Extração de tipos** (structs, classes, interfaces, traits, enums)
3. **Modo API Surface** para visualização compacta de APIs públicas

O objetivo é fornecer um mapa mais rico para LLMs sem comprometer a token-efficiency que é o diferencial do codemap.

---

## Princípios de Design

### 1. Backward Compatibility
- Output atual deve permanecer inalterado por padrão
- Novas features ativadas via flags opcionais
- JSON schema deve ser extensível sem quebrar consumers

### 2. Token Efficiency
- Modo padrão continua mostrando apenas nomes
- Detalhes extras são opt-in via `--detail`
- Modo `--api` oferece visualização ultra-compacta

### 3. Consistência Semântica
- Tipos são categorizados por `Kind` semântico normalizado
- Cada linguagem mapeia seus constructos para Kinds universais
- Evita falsos cognatos (Rust `trait` ≠ Java `interface`)

---

## Arquitetura de Mudanças

```
┌─────────────────────────────────────────────────────────────────┐
│                         main.go                                  │
│  + --detail flag (0, 1, 2)                                      │
│  + --api flag                                                    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      scanner/types.go                            │
│  + FuncInfo struct (Name, Signature, Receiver, IsExported)      │
│  + TypeInfo struct (Name, Kind, Fields, IsExported)             │
│  ~ FileAnalysis (Functions: []string → []FuncInfo)              │
│  ~ FileAnalysis (+ Types: []TypeInfo)                           │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     scanner/grammar.go                           │
│  ~ AnalyzeFile() - extended capture handling                    │
│  + buildSignature() - reconstructs signature from parts         │
│  + parseTypeInfo() - extracts type metadata                     │
│  + DetailLevel parameter propagation                            │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    scanner/queries/*.scm                         │
│  ~ All 16 language queries updated with:                        │
│    - @func.name, @func.params, @func.result, @func.receiver     │
│    - @type.name, @type.kind, @type.fields                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      render/depgraph.go                          │
│  ~ Depgraph() - conditional signature display                   │
│  + renderTypes() - type listing in output                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       render/api.go (NEW)                        │
│  + APIView() - compact public API visualization                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Fase 1: Estruturas de Dados Base

### 1.1 Novos Tipos em `scanner/types.go`

```go
// DetailLevel controls how much information is extracted
type DetailLevel int

const (
    DetailNone      DetailLevel = 0 // Only names (current behavior)
    DetailSignature DetailLevel = 1 // Names + signatures
    DetailFull      DetailLevel = 2 // Signatures + type fields
)

// FuncInfo represents a function/method with optional detail
type FuncInfo struct {
    Name       string `json:"name"`
    Signature  string `json:"signature,omitempty"`  // Full signature when detail >= 1
    Receiver   string `json:"receiver,omitempty"`   // For methods (Go, Rust, etc.)
    IsExported bool   `json:"exported,omitempty"`   // Public visibility
}

// TypeKind represents normalized type categories across languages
type TypeKind string

const (
    KindStruct    TypeKind = "struct"    // Go struct, C struct, Rust struct
    KindClass     TypeKind = "class"     // Python/Java/C#/TS class
    KindInterface TypeKind = "interface" // Go interface, Java/C#/TS interface
    KindTrait     TypeKind = "trait"     // Rust trait
    KindEnum      TypeKind = "enum"      // All languages
    KindTypeAlias TypeKind = "alias"     // Go type alias, TS type
    KindProtocol  TypeKind = "protocol"  // Swift protocol
)

// TypeInfo represents a type definition
type TypeInfo struct {
    Name       string   `json:"name"`
    Kind       TypeKind `json:"kind"`
    Fields     []string `json:"fields,omitempty"`     // Field names when detail = 2
    Methods    []string `json:"methods,omitempty"`    // Method names (for classes)
    IsExported bool     `json:"exported,omitempty"`
}

// FileAnalysis holds extracted info about a single file for deps mode.
// UPDATED: Functions now use FuncInfo, added Types field
type FileAnalysis struct {
    Path      string     `json:"path"`
    Language  string     `json:"language"`
    Functions []FuncInfo `json:"functions"`           // CHANGED from []string
    Types     []TypeInfo `json:"types,omitempty"`     // NEW
    Imports   []string   `json:"imports"`
}
```

### 1.2 Compatibilidade JSON

Para manter backward compatibility na serialização JSON:

```go
// MarshalJSON customizes JSON output based on detail level
func (f FuncInfo) MarshalJSON() ([]byte, error) {
    // Se Signature está vazio, serializa apenas como string (comportamento antigo)
    if f.Signature == "" && f.Receiver == "" {
        return json.Marshal(f.Name)
    }
    // Caso contrário, serializa objeto completo
    type Alias FuncInfo
    return json.Marshal(Alias(f))
}
```

---

## Fase 2: Queries Tree-sitter Expandidas

### 2.1 Estratégia de Captura

**Problema:** Capturar `(function_declaration) @function` inclui o corpo inteiro.

**Solução:** Usar captures nomeados para componentes específicos:
- `@func.name` - nome da função
- `@func.params` - lista de parâmetros (texto do nó)
- `@func.result` - tipo de retorno
- `@func.receiver` - receiver para métodos

O código Go reconstruirá a signature a partir dos componentes.

### 2.2 Query Go (`scanner/queries/go.scm`)

```scheme
; ============================================
; GO QUERY - Enhanced for signatures and types
; ============================================

; --- FUNCTIONS ---

; Function declarations with full signature components
(function_declaration
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params
  result: (_)? @func.result)

; Method declarations (functions with receivers)
(method_declaration
  receiver: (parameter_list) @func.receiver
  name: (field_identifier) @func.name
  parameters: (parameter_list) @func.params
  result: (_)? @func.result)

; --- TYPES ---

; Struct type definitions
(type_declaration
  (type_spec
    name: (type_identifier) @type.name
    type: (struct_type
      (field_declaration_list)? @type.fields))) @type.struct

; Interface type definitions
(type_declaration
  (type_spec
    name: (type_identifier) @type.name
    type: (interface_type
      (method_spec_list)? @type.methods))) @type.interface

; Type aliases
(type_declaration
  (type_spec
    name: (type_identifier) @type.name
    type: (type_identifier) @type.alias.target)) @type.alias

; --- IMPORTS (unchanged) ---
(import_spec
  path: (interpreted_string_literal) @import)
```

### 2.3 Query Python (`scanner/queries/python.scm`)

```scheme
; ============================================
; PYTHON QUERY - Enhanced for signatures and types
; ============================================

; --- FUNCTIONS ---

; Function definitions with parameters
(function_definition
  name: (identifier) @func.name
  parameters: (parameters) @func.params
  return_type: (type)? @func.result)

; Async function definitions
(function_definition
  "async"
  name: (identifier) @func.name
  parameters: (parameters) @func.params
  return_type: (type)? @func.result) @func.async

; --- CLASSES ---

; Class definitions
(class_definition
  name: (identifier) @type.name
  superclasses: (argument_list)? @type.bases) @type.class

; --- IMPORTS (unchanged) ---
(import_statement
  name: (dotted_name) @import)

(import_from_statement
  module_name: (dotted_name) @import)

(import_from_statement
  module_name: (relative_import) @import)
```

### 2.4 Query TypeScript (`scanner/queries/typescript.scm`)

```scheme
; ============================================
; TYPESCRIPT QUERY - Enhanced for signatures and types
; ============================================

; --- FUNCTIONS ---

; Function declarations
(function_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params
  return_type: (type_annotation)? @func.result)

; Arrow functions assigned to const/let
(lexical_declaration
  (variable_declarator
    name: (identifier) @func.name
    type: (type_annotation)? @func.result
    value: (arrow_function
      parameters: (formal_parameters) @func.params)))

; Method definitions in classes
(method_definition
  name: (property_identifier) @func.name
  parameters: (formal_parameters) @func.params
  return_type: (type_annotation)? @func.result)

; --- TYPES ---

; Interface declarations
(interface_declaration
  name: (type_identifier) @type.name
  body: (interface_body) @type.fields) @type.interface

; Class declarations
(class_declaration
  name: (type_identifier) @type.name
  body: (class_body) @type.fields) @type.class

; Type aliases
(type_alias_declaration
  name: (type_identifier) @type.name
  value: (_) @type.alias.target) @type.alias

; Enum declarations
(enum_declaration
  name: (identifier) @type.name
  body: (enum_body) @type.fields) @type.enum

; --- IMPORTS (unchanged) ---
(import_statement
  source: (string) @import)
```

### 2.5 Query Rust (`scanner/queries/rust.scm`)

```scheme
; ============================================
; RUST QUERY - Enhanced for signatures and types
; ============================================

; --- FUNCTIONS ---

; Function definitions
(function_item
  name: (identifier) @func.name
  parameters: (parameters) @func.params
  return_type: (_)? @func.result)

; Methods in impl blocks (capture self for receiver detection)
(impl_item
  type: (_) @func.receiver.type
  body: (declaration_list
    (function_item
      name: (identifier) @func.name
      parameters: (parameters) @func.params
      return_type: (_)? @func.result)))

; --- TYPES ---

; Struct definitions
(struct_item
  name: (type_identifier) @type.name
  body: (field_declaration_list)? @type.fields) @type.struct

; Enum definitions
(enum_item
  name: (type_identifier) @type.name
  body: (enum_variant_list) @type.fields) @type.enum

; Trait definitions
(trait_item
  name: (type_identifier) @type.name
  body: (declaration_list) @type.methods) @type.trait

; Type aliases
(type_item
  name: (type_identifier) @type.name
  type: (_) @type.alias.target) @type.alias

; --- IMPORTS (unchanged) ---
(use_declaration
  argument: (scoped_identifier) @import)

(use_declaration
  argument: (identifier) @import)

(mod_item
  name: (identifier) @module)
```

### 2.6 Query Java (`scanner/queries/java.scm`)

```scheme
; ============================================
; JAVA QUERY - Enhanced for signatures and types
; ============================================

; --- FUNCTIONS ---

; Method declarations
(method_declaration
  type: (_) @func.result
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Constructor declarations
(constructor_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params)

; --- TYPES ---

; Class declarations
(class_declaration
  name: (identifier) @type.name
  interfaces: (super_interfaces)? @type.implements
  body: (class_body) @type.fields) @type.class

; Interface declarations
(interface_declaration
  name: (identifier) @type.name
  body: (interface_body) @type.methods) @type.interface

; Enum declarations
(enum_declaration
  name: (identifier) @type.name
  body: (enum_body) @type.fields) @type.enum

; --- IMPORTS (unchanged) ---
(import_declaration
  (scoped_identifier) @import)
```

### 2.7 Query C# (`scanner/queries/c_sharp.scm`)

```scheme
; ============================================
; C# QUERY - Enhanced for signatures and types
; ============================================

; --- FUNCTIONS ---

; Method declarations
(method_declaration
  returns: (_) @func.result
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params)

; Constructor declarations
(constructor_declaration
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params)

; --- TYPES ---

; Class declarations
(class_declaration
  name: (identifier) @type.name
  bases: (base_list)? @type.bases
  body: (declaration_list) @type.fields) @type.class

; Interface declarations
(interface_declaration
  name: (identifier) @type.name
  body: (declaration_list) @type.methods) @type.interface

; Struct declarations
(struct_declaration
  name: (identifier) @type.name
  body: (declaration_list) @type.fields) @type.struct

; Enum declarations
(enum_declaration
  name: (identifier) @type.name
  body: (enum_member_declaration_list) @type.fields) @type.enum

; --- IMPORTS (unchanged) ---
(using_directive
  (qualified_name) @import)

(using_directive
  (identifier) @import)
```

---

## Fase 3: Lógica de Análise Expandida

### 3.1 Modificação em `scanner/grammar.go`

```go
// AnalyzeFile extracts functions, types, and imports
// detailLevel controls depth of extraction:
//   0 = names only (current behavior)
//   1 = names + signatures
//   2 = names + signatures + type fields
func (l *GrammarLoader) AnalyzeFile(filePath string, detailLevel DetailLevel) (*FileAnalysis, error) {
    lang := DetectLanguage(filePath)
    if lang == "" {
        return nil, nil
    }

    if err := l.LoadLanguage(lang); err != nil {
        return nil, nil
    }

    config := l.configs[lang]
    content, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }

    parser := tree_sitter.NewParser()
    defer parser.Close()
    parser.SetLanguage(config.Language)

    tree := parser.Parse(content, nil)
    defer tree.Close()

    cursor := tree_sitter.NewQueryCursor()
    defer cursor.Close()

    analysis := &FileAnalysis{Path: filePath, Language: lang}

    // Temporary storage for building composite captures
    funcBuilder := make(map[uint32]*funcCapture) // match_id -> components
    typeBuilder := make(map[uint32]*typeCapture)

    matches := cursor.Matches(config.Query, tree.RootNode(), content)
    for match := matches.Next(); match != nil; match = matches.Next() {
        for _, capture := range match.Captures {
            name := config.Query.CaptureNames()[capture.Index]
            text := strings.Trim(capture.Node.Utf8Text(content), `"'`)

            // Route to appropriate handler based on capture name prefix
            switch {
            case strings.HasPrefix(name, "func."):
                handleFuncCapture(funcBuilder, match.ID, name, text, capture.Node)
            case strings.HasPrefix(name, "type."):
                handleTypeCapture(typeBuilder, match.ID, name, text, capture.Node, detailLevel)
            case name == "import" || name == "module":
                analysis.Imports = append(analysis.Imports, text)
            // Legacy support: plain @function capture
            case name == "function" || name == "method":
                analysis.Functions = append(analysis.Functions, FuncInfo{Name: text})
            }
        }
    }

    // Build final function list from captured components
    for _, fc := range funcBuilder {
        funcInfo := fc.Build(detailLevel, lang)
        analysis.Functions = append(analysis.Functions, funcInfo)
    }

    // Build final type list
    for _, tc := range typeBuilder {
        typeInfo := tc.Build(detailLevel)
        analysis.Types = append(analysis.Types, typeInfo)
    }

    analysis.Functions = dedupeFuncs(analysis.Functions)
    analysis.Types = dedupeTypes(analysis.Types)
    analysis.Imports = dedupe(analysis.Imports)

    return analysis, nil
}

// funcCapture collects components of a function signature
type funcCapture struct {
    name     string
    params   string
    result   string
    receiver string
    node     *tree_sitter.Node
}

// Build constructs FuncInfo from captured components
func (fc *funcCapture) Build(detail DetailLevel, lang string) FuncInfo {
    info := FuncInfo{
        Name:       fc.name,
        IsExported: isExported(fc.name, lang),
    }

    if detail >= DetailSignature && fc.params != "" {
        info.Signature = buildSignature(fc, lang)
    }

    if fc.receiver != "" {
        info.Receiver = fc.receiver
    }

    return info
}

// buildSignature reconstructs function signature from components
func buildSignature(fc *funcCapture, lang string) string {
    var sig strings.Builder

    switch lang {
    case "go":
        sig.WriteString("func ")
        if fc.receiver != "" {
            sig.WriteString(fc.receiver)
            sig.WriteString(" ")
        }
        sig.WriteString(fc.name)
        sig.WriteString(fc.params)
        if fc.result != "" {
            sig.WriteString(" ")
            sig.WriteString(fc.result)
        }

    case "python":
        sig.WriteString("def ")
        sig.WriteString(fc.name)
        sig.WriteString(fc.params)
        if fc.result != "" {
            sig.WriteString(" -> ")
            sig.WriteString(fc.result)
        }

    case "typescript", "javascript":
        sig.WriteString("function ")
        sig.WriteString(fc.name)
        sig.WriteString(fc.params)
        if fc.result != "" {
            sig.WriteString(": ")
            sig.WriteString(fc.result)
        }

    case "rust":
        sig.WriteString("fn ")
        sig.WriteString(fc.name)
        sig.WriteString(fc.params)
        if fc.result != "" {
            sig.WriteString(" -> ")
            sig.WriteString(fc.result)
        }

    case "java", "c_sharp":
        if fc.result != "" {
            sig.WriteString(fc.result)
            sig.WriteString(" ")
        }
        sig.WriteString(fc.name)
        sig.WriteString(fc.params)

    default:
        sig.WriteString(fc.name)
        sig.WriteString(fc.params)
    }

    return sig.String()
}

// isExported checks if a symbol is publicly visible
func isExported(name, lang string) bool {
    if name == "" {
        return false
    }

    switch lang {
    case "go":
        // Go: exported if starts with uppercase
        r := []rune(name)
        return unicode.IsUpper(r[0])

    case "python":
        // Python: exported if doesn't start with _
        return !strings.HasPrefix(name, "_")

    case "rust":
        // Rust: would need `pub` keyword analysis
        // For now, assume all are potentially public
        return true

    default:
        // Most languages: assume public unless private keyword
        return true
    }
}

// typeCapture collects components of a type definition
type typeCapture struct {
    name   string
    kind   TypeKind
    fields string // Raw text of fields block
    node   *tree_sitter.Node
}

// Build constructs TypeInfo from captured components
func (tc *typeCapture) Build(detail DetailLevel) TypeInfo {
    info := TypeInfo{
        Name: tc.name,
        Kind: tc.kind,
    }

    if detail >= DetailFull && tc.fields != "" {
        info.Fields = parseFieldNames(tc.fields)
    }

    return info
}

// parseFieldNames extracts field/member names from raw block text
func parseFieldNames(fieldsText string) []string {
    // Simple extraction: find identifiers at start of lines
    var fields []string
    lines := strings.Split(fieldsText, "\n")

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || line == "{" || line == "}" {
            continue
        }

        // Extract first identifier (field name)
        parts := strings.Fields(line)
        if len(parts) > 0 {
            name := strings.TrimSuffix(parts[0], ":")
            name = strings.TrimSuffix(name, ",")
            if name != "" && !strings.HasPrefix(name, "//") {
                fields = append(fields, name)
            }
        }
    }

    return fields
}
```

### 3.2 Handler para Captures

```go
// handleFuncCapture routes function-related captures to builder
func handleFuncCapture(builders map[uint32]*funcCapture, matchID uint32, name, text string, node *tree_sitter.Node) {
    if builders[matchID] == nil {
        builders[matchID] = &funcCapture{}
    }
    fc := builders[matchID]

    switch name {
    case "func.name":
        fc.name = text
    case "func.params":
        fc.params = text
    case "func.result":
        fc.result = text
    case "func.receiver":
        fc.receiver = text
    }
    fc.node = node
}

// handleTypeCapture routes type-related captures to builder
func handleTypeCapture(builders map[uint32]*typeCapture, matchID uint32, name, text string, node *tree_sitter.Node, detail DetailLevel) {
    if builders[matchID] == nil {
        builders[matchID] = &typeCapture{}
    }
    tc := builders[matchID]

    switch name {
    case "type.name":
        tc.name = text
    case "type.fields", "type.methods":
        if detail >= DetailFull {
            tc.fields = text
        }
    case "type.struct":
        tc.kind = KindStruct
    case "type.class":
        tc.kind = KindClass
    case "type.interface":
        tc.kind = KindInterface
    case "type.trait":
        tc.kind = KindTrait
    case "type.enum":
        tc.kind = KindEnum
    case "type.alias":
        tc.kind = KindTypeAlias
    case "type.protocol":
        tc.kind = KindProtocol
    }
    tc.node = node
}
```

---

## Fase 4: CLI e Flags

### 4.1 Modificação em `main.go`

```go
func main() {
    // Existing flags...
    skylineMode := flag.Bool("skyline", false, "Enable skyline visualization mode")
    animateMode := flag.Bool("animate", false, "Enable animation (use with --skyline)")
    depsMode := flag.Bool("deps", false, "Enable dependency graph mode (function/import analysis)")
    diffMode := flag.Bool("diff", false, "Only show files changed vs main (or use --ref to specify branch)")
    diffRef := flag.String("ref", "main", "Branch/ref to compare against (use with --diff)")
    jsonMode := flag.Bool("json", false, "Output JSON (for Python renderer compatibility)")
    debugMode := flag.Bool("debug", false, "Show debug info (gitignore loading, paths, etc.)")
    helpMode := flag.Bool("help", false, "Show help")

    // NEW FLAGS
    detailLevel := flag.Int("detail", 0, "Detail level: 0=names, 1=signatures, 2=full (use with --deps)")
    apiMode := flag.Bool("api", false, "Show public API surface only (compact view)")

    flag.Parse()

    // ... rest of main
}

func runDepsMode(absRoot, root string, gitignore *ignore.GitIgnore, jsonMode bool, diffRef string, changedFiles map[string]bool, detailLevel int, apiMode bool) {
    loader := scanner.NewGrammarLoader()

    // ... grammar check ...

    analyses, err := scanner.ScanForDeps(root, gitignore, loader, scanner.DetailLevel(detailLevel))
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error scanning for deps: %v\n", err)
        os.Exit(1)
    }

    // ... filter changed files ...

    depsProject := scanner.DepsProject{
        Root:         absRoot,
        Mode:         "deps",
        Files:        analyses,
        ExternalDeps: scanner.ReadExternalDeps(absRoot),
        DiffRef:      diffRef,
        DetailLevel:  detailLevel, // NEW field
    }

    if jsonMode {
        json.NewEncoder(os.Stdout).Encode(depsProject)
    } else if apiMode {
        render.APIView(depsProject) // NEW renderer
    } else {
        render.Depgraph(depsProject)
    }
}
```

### 4.2 Help Text Atualizado

```go
if *helpMode {
    fmt.Println("codemap - Generate a brain map of your codebase for LLM context")
    fmt.Println()
    fmt.Println("Usage: codemap [options] [path]")
    fmt.Println()
    fmt.Println("Options:")
    fmt.Println("  --help              Show this help message")
    fmt.Println("  --skyline           City skyline visualization")
    fmt.Println("  --animate           Animated skyline (use with --skyline)")
    fmt.Println("  --deps              Dependency flow map (functions & imports)")
    fmt.Println("  --detail <level>    Detail level for --deps: 0=names, 1=signatures, 2=full")
    fmt.Println("  --api               Show public API surface (compact, use with --deps)")
    fmt.Println("  --diff              Only show files changed vs main")
    fmt.Println("  --ref <branch>      Branch to compare against (default: main)")
    fmt.Println()
    fmt.Println("Examples:")
    fmt.Println("  codemap .                       # Basic tree view")
    fmt.Println("  codemap --deps .                # Dependency flow (names only)")
    fmt.Println("  codemap --deps --detail 1 .    # Dependencies with signatures")
    fmt.Println("  codemap --deps --api .         # Public API surface view")
    fmt.Println("  codemap --deps --detail 2 .    # Full detail with type fields")
    os.Exit(0)
}
```

---

## Fase 5: Renderização

### 5.1 Modificação em `render/depgraph.go`

```go
// Depgraph renders the dependency flow visualization
func Depgraph(project scanner.DepsProject) {
    files := project.Files
    detailLevel := project.DetailLevel

    // ... existing code ...

    // Render each system
    for _, system := range systemNames {
        sysFiles := systems[system]
        systemName := getSystemName(system)

        // ... existing header code ...

        for _, f := range sysFiles {
            basename := filepath.Base(f.Path)
            nameNoExt := extPattern.ReplaceAllString(basename, "")

            // NEW: Show types if present
            if len(f.Types) > 0 && detailLevel >= 1 {
                renderTypes(f.Types, detailLevel)
            }

            // ... existing dependency rendering ...

            // NEW: Show function signatures if detail level warrants
            if detailLevel >= 1 {
                renderFunctionsDetailed(f.Functions)
            }
        }
    }

    // ... existing summary code ...
}

// renderTypes displays type definitions
func renderTypes(types []scanner.TypeInfo, detail int) {
    for _, t := range types {
        icon := typeIcon(t.Kind)
        if detail >= 2 && len(t.Fields) > 0 {
            fmt.Printf("    %s %s { %s }\n", icon, t.Name, strings.Join(t.Fields, ", "))
        } else {
            fmt.Printf("    %s %s\n", icon, t.Name)
        }
    }
}

// typeIcon returns an ASCII icon for the type kind
func typeIcon(kind scanner.TypeKind) string {
    switch kind {
    case scanner.KindStruct:
        return "[S]"
    case scanner.KindClass:
        return "[C]"
    case scanner.KindInterface:
        return "[I]"
    case scanner.KindTrait:
        return "[T]"
    case scanner.KindEnum:
        return "[E]"
    case scanner.KindTypeAlias:
        return "[A]"
    case scanner.KindProtocol:
        return "[P]"
    default:
        return "[?]"
    }
}

// renderFunctionsDetailed shows functions with signatures
func renderFunctionsDetailed(funcs []scanner.FuncInfo) {
    for _, f := range funcs {
        if f.Signature != "" {
            fmt.Printf("      ƒ %s\n", f.Signature)
        } else {
            fmt.Printf("      ƒ %s()\n", f.Name)
        }
    }
}
```

### 5.2 Novo arquivo `render/api.go`

```go
package render

import (
    "fmt"
    "path/filepath"
    "sort"
    "strings"

    "codemap/scanner"
)

// APIView renders a compact public API surface view
func APIView(project scanner.DepsProject) {
    files := project.Files
    projectName := filepath.Base(project.Root)

    if len(files) == 0 {
        fmt.Println("  No source files found.")
        return
    }

    fmt.Println()
    fmt.Printf("=== API Surface: %s ===\n", projectName)
    fmt.Println()

    // Group by directory
    packages := make(map[string][]scanner.FileAnalysis)
    for _, f := range files {
        dir := filepath.Dir(f.Path)
        if dir == "." {
            dir = projectName
        }
        packages[dir] = append(packages[dir], f)
    }

    // Sort package names
    var pkgNames []string
    for name := range packages {
        pkgNames = append(pkgNames, name)
    }
    sort.Strings(pkgNames)

    for _, pkg := range pkgNames {
        pkgFiles := packages[pkg]

        // Collect all exported types and functions
        var exportedTypes []scanner.TypeInfo
        var exportedFuncs []scanner.FuncInfo

        for _, f := range pkgFiles {
            for _, t := range f.Types {
                if t.IsExported {
                    exportedTypes = append(exportedTypes, t)
                }
            }
            for _, fn := range f.Functions {
                if fn.IsExported {
                    exportedFuncs = append(exportedFuncs, fn)
                }
            }
        }

        // Skip packages with no exports
        if len(exportedTypes) == 0 && len(exportedFuncs) == 0 {
            continue
        }

        fmt.Printf("%s/\n", pkg)

        // Group methods by receiver type
        methodsByType := make(map[string][]scanner.FuncInfo)
        var standaloneFuncs []scanner.FuncInfo

        for _, fn := range exportedFuncs {
            if fn.Receiver != "" {
                // Extract type name from receiver
                typeName := extractTypeName(fn.Receiver)
                methodsByType[typeName] = append(methodsByType[typeName], fn)
            } else {
                standaloneFuncs = append(standaloneFuncs, fn)
            }
        }

        // Print types with their methods
        for _, t := range exportedTypes {
            icon := typeIcon(t.Kind)

            if len(t.Fields) > 0 {
                fmt.Printf("  %s %s {%s}\n", icon, t.Name, strings.Join(t.Fields, ", "))
            } else {
                fmt.Printf("  %s %s\n", icon, t.Name)
            }

            // Print methods for this type
            if methods, ok := methodsByType[t.Name]; ok {
                for _, m := range methods {
                    if m.Signature != "" {
                        fmt.Printf("    + %s\n", m.Signature)
                    } else {
                        fmt.Printf("    + (%s) %s()\n", m.Receiver, m.Name)
                    }
                }
            }
        }

        // Print standalone functions
        if len(standaloneFuncs) > 0 {
            for _, fn := range standaloneFuncs {
                if fn.Signature != "" {
                    fmt.Printf("  + %s\n", fn.Signature)
                } else {
                    fmt.Printf("  + %s()\n", fn.Name)
                }
            }
        }

        fmt.Println()
    }

    // Summary
    totalTypes := 0
    totalFuncs := 0
    for _, f := range files {
        for _, t := range f.Types {
            if t.IsExported {
                totalTypes++
            }
        }
        for _, fn := range f.Functions {
            if fn.IsExported {
                totalFuncs++
            }
        }
    }

    fmt.Printf("─────────────────────────────────────────────────────────────\n")
    fmt.Printf("Exported: %d types · %d functions\n", totalTypes, totalFuncs)
    fmt.Println()
}

// extractTypeName extracts type name from receiver like "(l *GrammarLoader)"
func extractTypeName(receiver string) string {
    // Remove parentheses
    r := strings.Trim(receiver, "()")

    // Split by space, take last part (the type)
    parts := strings.Fields(r)
    if len(parts) == 0 {
        return ""
    }

    typePart := parts[len(parts)-1]
    // Remove pointer asterisk
    return strings.TrimPrefix(typePart, "*")
}
```

---

## Fase 6: MCP Server Updates

### 6.1 Modificação em `mcp/main.go`

Atualizar as tools MCP para suportar o novo detail level:

```go
// get_dependencies tool - add detail parameter
case "get_dependencies":
    path := getStringArg(args, "path", ".")
    detail := getIntArg(args, "detail", 0) // NEW

    absPath, _ := filepath.Abs(path)
    gitignore := scanner.LoadGitignore(absPath)
    loader := scanner.NewGrammarLoader()

    if !loader.HasGrammars() {
        return mcp.NewToolResultError("No tree-sitter grammars found")
    }

    analyses, err := scanner.ScanForDeps(absPath, gitignore, loader, scanner.DetailLevel(detail))
    if err != nil {
        return mcp.NewToolResultError(err.Error())
    }

    project := scanner.DepsProject{
        Root:         absPath,
        Mode:         "deps",
        Files:        analyses,
        ExternalDeps: scanner.ReadExternalDeps(absPath),
        DetailLevel:  detail,
    }

    result, _ := json.MarshalIndent(project, "", "  ")
    return mcp.NewToolResultText(string(result))
```

---

## Fase 7: Testes

### 7.1 Estrutura de Testes

```
scanner/
  grammar_test.go      # Test signature extraction
  types_test.go        # Test type parsing

render/
  api_test.go          # Test API view rendering

testdata/
  go/
    sample.go          # Test Go parsing
  python/
    sample.py          # Test Python parsing
  typescript/
    sample.ts          # Test TS parsing
  rust/
    sample.rs          # Test Rust parsing
```

### 7.2 Test Cases para Go

```go
// scanner/grammar_test.go
func TestAnalyzeGoSignatures(t *testing.T) {
    loader := NewGrammarLoader()

    // Create temp file with test content
    content := `package test

type Config struct {
    Name string
    Port int
}

type Handler interface {
    Handle(ctx context.Context, req Request) (Response, error)
}

func NewConfig(name string, port int) *Config {
    return &Config{Name: name, Port: port}
}

func (c *Config) Validate() error {
    if c.Port < 0 {
        return errors.New("invalid port")
    }
    return nil
}
`
    tmpFile := createTempFile(t, "test.go", content)
    defer os.Remove(tmpFile)

    // Test detail level 0 (names only)
    analysis, err := loader.AnalyzeFile(tmpFile, DetailNone)
    require.NoError(t, err)
    require.Len(t, analysis.Functions, 2)
    assert.Equal(t, "NewConfig", analysis.Functions[0].Name)
    assert.Empty(t, analysis.Functions[0].Signature)

    // Test detail level 1 (signatures)
    analysis, err = loader.AnalyzeFile(tmpFile, DetailSignature)
    require.NoError(t, err)
    require.Len(t, analysis.Functions, 2)
    assert.Equal(t, "func NewConfig(name string, port int) *Config", analysis.Functions[0].Signature)
    assert.Equal(t, "func (c *Config) Validate() error", analysis.Functions[1].Signature)

    // Test types
    require.Len(t, analysis.Types, 2)
    assert.Equal(t, "Config", analysis.Types[0].Name)
    assert.Equal(t, KindStruct, analysis.Types[0].Kind)
    assert.Equal(t, "Handler", analysis.Types[1].Name)
    assert.Equal(t, KindInterface, analysis.Types[1].Kind)

    // Test detail level 2 (fields)
    analysis, err = loader.AnalyzeFile(tmpFile, DetailFull)
    require.NoError(t, err)
    assert.Contains(t, analysis.Types[0].Fields, "Name")
    assert.Contains(t, analysis.Types[0].Fields, "Port")
}
```

### 7.3 Test Cases para Python

```go
func TestAnalyzePythonSignatures(t *testing.T) {
    content := `
class UserService:
    def __init__(self, db: Database):
        self.db = db

    async def get_user(self, user_id: int) -> User:
        return await self.db.get(user_id)

    def _private_method(self):
        pass

def create_app(config: Config) -> Application:
    return Application(config)
`
    tmpFile := createTempFile(t, "test.py", content)
    defer os.Remove(tmpFile)

    loader := NewGrammarLoader()
    analysis, err := loader.AnalyzeFile(tmpFile, DetailSignature)
    require.NoError(t, err)

    // Check class
    require.Len(t, analysis.Types, 1)
    assert.Equal(t, "UserService", analysis.Types[0].Name)
    assert.Equal(t, KindClass, analysis.Types[0].Kind)

    // Check functions
    require.Len(t, analysis.Functions, 4)

    // Check exported detection
    var exported []string
    for _, f := range analysis.Functions {
        if f.IsExported {
            exported = append(exported, f.Name)
        }
    }
    assert.Contains(t, exported, "__init__")
    assert.Contains(t, exported, "get_user")
    assert.Contains(t, exported, "create_app")
    assert.NotContains(t, exported, "_private_method")
}
```

---

## Cronograma de Implementação

### Sprint 1: Fundação (Types + Basic Signatures)
- [ ] Implementar `FuncInfo` e `TypeInfo` em `types.go`
- [ ] Atualizar `FileAnalysis` struct
- [ ] Implementar JSON marshaling customizado
- [ ] Adicionar flag `--detail` em `main.go`
- [ ] Atualizar `ScanForDeps` para aceitar `DetailLevel`

### Sprint 2: Go + Python Queries
- [ ] Reescrever `go.scm` com captures granulares
- [ ] Reescrever `python.scm` com captures granulares
- [ ] Implementar `handleFuncCapture` e `handleTypeCapture`
- [ ] Implementar `buildSignature` para Go e Python
- [ ] Escrever testes unitários

### Sprint 3: TypeScript/JavaScript + Rust
- [ ] Reescrever `typescript.scm`
- [ ] Reescrever `javascript.scm`
- [ ] Reescrever `rust.scm`
- [ ] Implementar `buildSignature` para TS/JS/Rust
- [ ] Adicionar testes

### Sprint 4: Java + C# + Demais
- [ ] Reescrever `java.scm`
- [ ] Reescrever `c_sharp.scm`
- [ ] Atualizar queries restantes (C, C++, Swift, etc.)
- [ ] Implementar `buildSignature` para linguagens restantes

### Sprint 5: Renderização + API View
- [ ] Atualizar `depgraph.go` para signatures
- [ ] Implementar `render/api.go`
- [ ] Adicionar flag `--api`
- [ ] Testes de integração

### Sprint 6: MCP + Polish
- [ ] Atualizar MCP tools com `detail` parameter
- [ ] Atualizar documentação
- [ ] Performance testing
- [ ] Bug fixes finais

---

## Riscos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Tree-sitter query syntax errors | Alta | Médio | Testar cada query isoladamente com playground |
| Quebra de backward compatibility | Média | Alto | Custom JSON marshaling mantém output antigo |
| Performance degradation | Baixa | Médio | DetailLevel 0 usa código otimizado atual |
| Queries inconsistentes entre linguagens | Alta | Médio | Definir spec clara de captures obrigatórios |

---

## Definições de Pronto

### Feature "Completa" quando:
1. Todas as 16 linguagens suportam o feature
2. Testes unitários cobrem casos de borda
3. JSON output é backward compatible
4. Documentação atualizada no README
5. MCP server atualizado

### Aceitação:
- `codemap --deps .` funciona identicamente ao atual
- `codemap --deps --detail 1 .` mostra signatures
- `codemap --deps --api .` mostra API surface compacta
- JSON output é parseável por consumers existentes

---

## Notas de Implementação

### Ordem de Prioridade das Linguagens
1. **Go** - Linguagem do projeto, mais fácil de testar
2. **Python** - Alta demanda, syntax clara
3. **TypeScript** - Muito usado, tipos explícitos
4. **Rust** - Syntax complexa mas bem definida
5. **Java/C#** - Similar entre si
6. **C/C++** - Mais complexo, menor prioridade

### Padrões de Código
- Preferir funções puras para transformações
- Usar table-driven tests
- Manter cada query file sob 100 linhas se possível
- Documentar edge cases nas queries com comentários

---

*Documento criado em: 2024-12-03*
*Última atualização: 2024-12-03*
*Autor: Claude Code Analysis*
