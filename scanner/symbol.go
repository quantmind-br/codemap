package scanner

import (
	"strings"
)

// SymbolQuery represents the filters for symbol search
type SymbolQuery struct {
	Name string // Substring match (case-insensitive)
	Kind string // "function", "type", "all"
	File string // Filter by specific file (optional)
}

// SymbolMatch represents a found symbol
type SymbolMatch struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`      // "function" or "type"
	Signature string `json:"signature"` // For functions
	TypeKind  string `json:"type_kind"` // For types (struct, class, etc.)
	File      string `json:"file"`
	Line      int    `json:"line"`
	Exported  bool   `json:"exported"`
}

// SearchSymbols searches for symbols in the analyzed files
func SearchSymbols(analyses []FileAnalysis, query SymbolQuery) []SymbolMatch {
	var matches []SymbolMatch
	searchName := strings.ToLower(query.Name)

	for _, analysis := range analyses {
		// Skip if file filter is set and doesn't match
		if query.File != "" && !strings.Contains(analysis.Path, query.File) {
			continue
		}

		// Search functions
		if query.Kind == "" || query.Kind == "all" || query.Kind == "function" {
			for _, fn := range analysis.Functions {
				if searchName == "" || strings.Contains(strings.ToLower(fn.Name), searchName) {
					matches = append(matches, SymbolMatch{
						Name:      fn.Name,
						Kind:      "function",
						Signature: fn.Signature,
						File:      analysis.Path,
						Line:      fn.Line,
						Exported:  fn.IsExported,
					})
				}
			}
		}

		// Search types
		if query.Kind == "" || query.Kind == "all" || query.Kind == "type" {
			for _, t := range analysis.Types {
				if searchName == "" || strings.Contains(strings.ToLower(t.Name), searchName) {
					matches = append(matches, SymbolMatch{
						Name:     t.Name,
						Kind:     "type",
						TypeKind: string(t.Kind),
						File:     analysis.Path,
						Line:     t.Line,
						Exported: t.IsExported,
					})
				}
			}
		}
	}

	return matches
}
