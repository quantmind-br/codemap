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
