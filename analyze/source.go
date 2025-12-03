package analyze

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codemap/graph"
)

// SymbolSource represents the extracted source code for a symbol.
type SymbolSource struct {
	// Node is the graph node this source belongs to
	Node *graph.Node

	// Source is the raw source code
	Source string

	// Language is the programming language
	Language string

	// ContentHash is a SHA256 hash of the source for cache invalidation
	ContentHash string

	// Context provides surrounding code for context
	Context *SourceContext
}

// SourceContext provides surrounding code context for a symbol.
type SourceContext struct {
	// Before contains lines before the symbol (e.g., imports, comments)
	Before string

	// After contains lines after the symbol
	After string

	// FileHeader contains the file-level documentation
	FileHeader string
}

// ReadSymbolSource extracts the source code for a symbol from its file.
// The projectRoot is used to resolve relative paths in the node.
func ReadSymbolSource(projectRoot string, node *graph.Node) (*SymbolSource, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	if node.Path == "" {
		return nil, fmt.Errorf("node has no path")
	}

	// Resolve full path
	fullPath := filepath.Join(projectRoot, node.Path)

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", fullPath, err)
	}

	lines := strings.Split(string(content), "\n")

	// Extract source based on line numbers
	var source string
	if node.Line > 0 && node.EndLine > 0 {
		// Extract specific lines (1-indexed to 0-indexed)
		startLine := node.Line - 1
		endLine := node.EndLine

		if startLine < 0 {
			startLine = 0
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}

		source = strings.Join(lines[startLine:endLine], "\n")
	} else if node.Line > 0 {
		// Only start line available, use heuristic to find end
		startLine := node.Line - 1
		if startLine < 0 {
			startLine = 0
		}

		// For files, use entire content
		if node.Kind == graph.KindFile {
			source = string(content)
		} else {
			// Use a reasonable chunk (50 lines) if we don't have end line
			endLine := startLine + 50
			if endLine > len(lines) {
				endLine = len(lines)
			}
			source = strings.Join(lines[startLine:endLine], "\n")
		}
	} else {
		// No line info, use entire file for file nodes
		if node.Kind == graph.KindFile {
			source = string(content)
		} else {
			return nil, fmt.Errorf("node has no line information")
		}
	}

	// Compute content hash
	hash := sha256.Sum256([]byte(source))
	contentHash := hex.EncodeToString(hash[:])

	// Detect language from file extension
	language := detectLanguage(node.Path)

	return &SymbolSource{
		Node:        node,
		Source:      source,
		Language:    language,
		ContentHash: contentHash,
	}, nil
}

// ReadSymbolSourceWithContext extracts source code with surrounding context.
func ReadSymbolSourceWithContext(projectRoot string, node *graph.Node, contextLines int) (*SymbolSource, error) {
	source, err := ReadSymbolSource(projectRoot, node)
	if err != nil {
		return nil, err
	}

	if contextLines <= 0 {
		return source, nil
	}

	// Read the file again for context
	fullPath := filepath.Join(projectRoot, node.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return source, nil // Return source without context on error
	}

	lines := strings.Split(string(content), "\n")

	// Extract context before
	if node.Line > 1 {
		startContext := node.Line - 1 - contextLines
		if startContext < 0 {
			startContext = 0
		}
		endContext := node.Line - 1
		source.Context = &SourceContext{
			Before: strings.Join(lines[startContext:endContext], "\n"),
		}
	}

	// Extract context after
	if node.EndLine > 0 && node.EndLine < len(lines) {
		startContext := node.EndLine
		endContext := node.EndLine + contextLines
		if endContext > len(lines) {
			endContext = len(lines)
		}
		if source.Context == nil {
			source.Context = &SourceContext{}
		}
		source.Context.After = strings.Join(lines[startContext:endContext], "\n")
	}

	// Extract file header (first comment block)
	source.Context.FileHeader = extractFileHeader(lines)

	return source, nil
}

// ReadModuleSource reads all source files in a directory.
func ReadModuleSource(projectRoot, modulePath string) ([]*SymbolSource, error) {
	fullPath := filepath.Join(projectRoot, modulePath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", fullPath, err)
	}

	if !info.IsDir() {
		// Single file
		node := &graph.Node{
			Kind: graph.KindFile,
			Name: filepath.Base(modulePath),
			Path: modulePath,
		}
		source, err := ReadSymbolSource(projectRoot, node)
		if err != nil {
			return nil, err
		}
		return []*SymbolSource{source}, nil
	}

	// Directory - read all source files
	var sources []*SymbolSource
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", fullPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isSourceFile(name) {
			continue
		}

		relPath := filepath.Join(modulePath, name)
		node := &graph.Node{
			Kind: graph.KindFile,
			Name: name,
			Path: relPath,
		}

		source, err := ReadSymbolSource(projectRoot, node)
		if err != nil {
			continue // Skip files we can't read
		}

		sources = append(sources, source)
	}

	return sources, nil
}

// detectLanguage returns the programming language based on file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescript"
	case ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".cs":
		return "csharp"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".sh", ".bash":
		return "bash"
	default:
		return "text"
	}
}

// isSourceFile checks if a filename is a recognized source file.
func isSourceFile(name string) bool {
	if strings.HasPrefix(name, ".") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rs", ".java",
		".c", ".h", ".cpp", ".cc", ".cxx", ".hpp", ".rb", ".php",
		".swift", ".kt", ".kts", ".scala", ".cs":
		return true
	default:
		return false
	}
}

// extractFileHeader extracts the file-level documentation comment.
func extractFileHeader(lines []string) string {
	var header []string
	inComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Package declarations end header
		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "from ") ||
			strings.HasPrefix(trimmed, "use ") ||
			strings.HasPrefix(trimmed, "require ") {
			break
		}

		// Track comment blocks
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			header = append(header, line)
			continue
		}

		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
			header = append(header, line)
			continue
		}

		if inComment {
			header = append(header, line)
			if strings.Contains(trimmed, "*/") {
				inComment = false
			}
			continue
		}

		// Empty lines are ok in header
		if trimmed == "" && len(header) > 0 {
			header = append(header, line)
			continue
		}

		// Non-empty, non-comment line ends header
		if trimmed != "" {
			break
		}
	}

	return strings.TrimSpace(strings.Join(header, "\n"))
}

// ContentHash computes a SHA256 hash of the given content.
func ContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ReadFileLines reads specific lines from a file.
func ReadFileLines(path string, startLine, endLine int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}
