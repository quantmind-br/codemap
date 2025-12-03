package scanner

import (
	"encoding/json"
	"strings"
	"unicode"
)

// DetailLevel controls how much information is extracted
type DetailLevel int

const (
	DetailNone      DetailLevel = 0 // Only names (current behavior)
	DetailSignature DetailLevel = 1 // Names + signatures
	DetailFull      DetailLevel = 2 // Signatures + type fields
)

// Token estimation constants
const (
	CharsPerToken   = 3.5  // Conservative estimate for BPE tokenizers
	LargeFileTokens = 8000 // Threshold for warning indicator
)

// EstimateTokens estimates the number of tokens for a given file size
func EstimateTokens(size int64) int {
	return int(float64(size) / CharsPerToken)
}

// FileInfo represents a single file in the codebase.
type FileInfo struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Ext     string `json:"ext"`
	Tokens  int    `json:"tokens,omitempty"` // Estimated token count
	IsNew   bool   `json:"is_new,omitempty"`
	Added   int    `json:"added,omitempty"`
	Removed int    `json:"removed,omitempty"`
}

// Project represents the root of the codebase for tree/skyline mode.
type Project struct {
	Root    string       `json:"root"`
	Mode    string       `json:"mode"`
	Animate bool         `json:"animate"`
	Files   []FileInfo   `json:"files"`
	DiffRef string       `json:"diff_ref,omitempty"`
	Impact  []ImpactInfo `json:"impact,omitempty"`
}

// FuncInfo represents a function/method with optional detail
type FuncInfo struct {
	Name       string `json:"name"`
	Signature  string `json:"signature,omitempty"` // Full signature when detail >= 1
	Receiver   string `json:"receiver,omitempty"`  // For methods (Go, Rust, etc.)
	IsExported bool   `json:"exported,omitempty"`  // Public visibility
	Line       int    `json:"line,omitempty"`      // Line number of definition (1-indexed)
}

// MarshalJSON customizes JSON output for backward compatibility
// When no extended info is present, serialize as plain string
func (f FuncInfo) MarshalJSON() ([]byte, error) {
	if f.Signature == "" && f.Receiver == "" && !f.IsExported && f.Line == 0 {
		return json.Marshal(f.Name)
	}
	type Alias FuncInfo
	return json.Marshal(Alias(f))
}

// UnmarshalJSON handles both string and object formats
func (f *FuncInfo) UnmarshalJSON(data []byte) error {
	// Try string first
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		f.Name = name
		return nil
	}
	// Fall back to object
	type Alias FuncInfo
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*f = FuncInfo(alias)
	return nil
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
	Fields     []string `json:"fields,omitempty"`  // Field names when detail = 2
	Methods    []string `json:"methods,omitempty"` // Method names (for classes)
	IsExported bool     `json:"exported,omitempty"`
	Line       int      `json:"line,omitempty"` // Line number of definition (1-indexed)
}

// FileAnalysis holds extracted info about a single file for deps mode.
type FileAnalysis struct {
	Path      string     `json:"path"`
	Language  string     `json:"language"`
	Functions []FuncInfo `json:"functions"`
	Types     []TypeInfo `json:"types,omitempty"`
	Imports   []string   `json:"imports"`
}

// DepsProject is the JSON output for --deps mode.
type DepsProject struct {
	Root         string              `json:"root"`
	Mode         string              `json:"mode"`
	Files        []FileAnalysis      `json:"files"`
	ExternalDeps map[string][]string `json:"external_deps"`
	DiffRef      string              `json:"diff_ref,omitempty"`
	DetailLevel  int                 `json:"detail_level,omitempty"`
}

// IsExportedName checks if a symbol name is exported based on language conventions
func IsExportedName(name, lang string) bool {
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
