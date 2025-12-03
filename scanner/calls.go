package scanner

import (
	"os"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// CallInfo represents a function call site.
type CallInfo struct {
	CallerFunc string `json:"caller"`             // Name of the function containing the call
	CallerLine int    `json:"caller_line"`        // Line where caller is defined
	CalleeName string `json:"callee"`             // Name of the called function
	CallLine   int    `json:"call_line"`          // Line where the call occurs
	Args       int    `json:"args"`               // Number of arguments
	Receiver   string `json:"receiver,omitempty"` // Object/receiver for method calls
}

// FileCallAnalysis contains all calls found in a file.
type FileCallAnalysis struct {
	Path     string     `json:"path"`
	Language string     `json:"language"`
	Calls    []CallInfo `json:"calls"`
}

// callQueryPatterns maps languages to their call expression query patterns.
// These are simpler inline patterns for common languages.
var callQueryPatterns = map[string]string{
	"go": `
; Function calls
(call_expression
  function: (identifier) @call.name
  arguments: (argument_list) @call.args)

; Method calls
(call_expression
  function: (selector_expression
    operand: (_) @call.receiver
    field: (field_identifier) @call.name)
  arguments: (argument_list) @call.args)
`,
	"python": `
; Function calls
(call
  function: (identifier) @call.name
  arguments: (argument_list) @call.args)

; Method calls
(call
  function: (attribute
    object: (_) @call.receiver
    attribute: (identifier) @call.name)
  arguments: (argument_list) @call.args)
`,
	"javascript": `
; Function calls
(call_expression
  function: (identifier) @call.name
  arguments: (arguments) @call.args)

; Method calls
(call_expression
  function: (member_expression
    object: (_) @call.receiver
    property: (property_identifier) @call.name)
  arguments: (arguments) @call.args)
`,
	"typescript": `
; Function calls
(call_expression
  function: (identifier) @call.name
  arguments: (arguments) @call.args)

; Method calls
(call_expression
  function: (member_expression
    object: (_) @call.receiver
    property: (property_identifier) @call.name)
  arguments: (arguments) @call.args)
`,
	"rust": `
; Function calls
(call_expression
  function: (identifier) @call.name
  arguments: (arguments) @call.args)

; Method calls
(call_expression
  function: (field_expression
    value: (_) @call.receiver
    field: (field_identifier) @call.name)
  arguments: (arguments) @call.args)
`,
	"java": `
; Method invocations
(method_invocation
  name: (identifier) @call.name
  arguments: (argument_list) @call.args)

; Method invocations with object
(method_invocation
  object: (_) @call.receiver
  name: (identifier) @call.name
  arguments: (argument_list) @call.args)
`,
}

// callConfigs caches compiled call queries per language.
var callConfigs = make(map[string]*tree_sitter.Query)

// ExtractCalls analyzes a file and extracts all function/method calls.
func (l *GrammarLoader) ExtractCalls(filePath string) (*FileCallAnalysis, error) {
	lang := DetectLanguage(filePath)
	if lang == "" {
		return nil, nil
	}

	// Check if we have a call query for this language first
	if _, ok := callQueryPatterns[lang]; !ok {
		return nil, nil // No call extraction support for this language
	}

	if err := l.LoadLanguage(lang); err != nil {
		return nil, nil
	}

	config := l.configs[lang]
	if config == nil || config.Language == nil {
		return nil, nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Get or compile the call query
	callQuery, err := l.getCallQuery(lang, config.Language)
	if err != nil || callQuery == nil {
		return nil, nil // No call query for this language
	}

	parser := tree_sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(config.Language)

	tree := parser.Parse(content, nil)
	defer tree.Close()

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	analysis := &FileCallAnalysis{Path: filePath, Language: lang}

	// First, build a map of line ranges to function names
	funcRanges := l.extractFunctionRanges(tree.RootNode(), content, config.Query)

	// Extract calls
	matches := cursor.Matches(callQuery, tree.RootNode(), content)

	var currentCall *CallInfo
	lastMatchID := uint(0xFFFFFFFF) // Use max value to ensure first match triggers initialization

	for match := matches.Next(); match != nil; match = matches.Next() {
		// New match = new call
		if lastMatchID != match.Id() {
			if currentCall != nil && currentCall.CalleeName != "" {
				// Find containing function
				currentCall.CallerFunc = findContainingFunction(currentCall.CallLine, funcRanges)
				analysis.Calls = append(analysis.Calls, *currentCall)
			}
			currentCall = &CallInfo{}
			lastMatchID = match.Id()
		}

		if currentCall == nil {
			currentCall = &CallInfo{}
		}

		for _, capture := range match.Captures {
			captureName := callQuery.CaptureNames()[capture.Index]
			text := capture.Node.Utf8Text(content)
			line := int(capture.Node.StartPosition().Row) + 1

			switch captureName {
			case "call.name":
				currentCall.CalleeName = text
				currentCall.CallLine = line
			case "call.receiver":
				currentCall.Receiver = text
			case "call.args":
				// Count arguments by counting commas + 1 (if not empty)
				currentCall.Args = countArgs(text)
			}
		}
	}

	// Don't forget last call
	if currentCall != nil && currentCall.CalleeName != "" {
		currentCall.CallerFunc = findContainingFunction(currentCall.CallLine, funcRanges)
		analysis.Calls = append(analysis.Calls, *currentCall)
	}

	return analysis, nil
}

// funcRange represents a function's line range for caller detection.
type funcRange struct {
	name      string
	startLine int
	endLine   int
}

// extractFunctionRanges builds a list of function ranges from the AST.
func (l *GrammarLoader) extractFunctionRanges(root *tree_sitter.Node, content []byte, query *tree_sitter.Query) []funcRange {
	var ranges []funcRange

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(query, root, content)

	var currentFunc *funcRange

	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			captureName := query.CaptureNames()[capture.Index]

			if strings.HasPrefix(captureName, "func.name") || captureName == "function" || captureName == "method" {
				// Get the function node (parent of name)
				funcNode := capture.Node.Parent()
				if funcNode != nil {
					currentFunc = &funcRange{
						name:      capture.Node.Utf8Text(content),
						startLine: int(funcNode.StartPosition().Row) + 1,
						endLine:   int(funcNode.EndPosition().Row) + 1,
					}
					ranges = append(ranges, *currentFunc)
				}
			}
		}
	}

	return ranges
}

// getCallQuery returns the compiled call query for a language.
func (l *GrammarLoader) getCallQuery(lang string, tsLang *tree_sitter.Language) (*tree_sitter.Query, error) {
	if q, ok := callConfigs[lang]; ok {
		return q, nil
	}

	pattern, ok := callQueryPatterns[lang]
	if !ok {
		return nil, nil // No call query for this language
	}

	query, err := tree_sitter.NewQuery(tsLang, pattern)
	if err != nil {
		return nil, err
	}

	callConfigs[lang] = query
	return query, nil
}

// findContainingFunction finds which function contains a given line.
func findContainingFunction(line int, ranges []funcRange) string {
	for i := len(ranges) - 1; i >= 0; i-- {
		r := ranges[i]
		if line >= r.startLine && line <= r.endLine {
			return r.name
		}
	}
	return "" // Global scope
}

// countArgs counts the number of arguments in an argument list.
func countArgs(argsText string) int {
	// Remove outer parens/brackets
	argsText = strings.TrimSpace(argsText)
	if len(argsText) < 2 {
		return 0
	}
	inner := argsText[1 : len(argsText)-1]
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return 0
	}
	// Simple comma count (doesn't handle nested commas perfectly)
	return strings.Count(inner, ",") + 1
}
