package scanner

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:embed queries/*.scm
var queryFiles embed.FS

// LanguageConfig holds dynamically loaded parser and query
type LanguageConfig struct {
	Language *tree_sitter.Language
	Query    *tree_sitter.Query
}

// GrammarLoader handles dynamic loading of tree-sitter grammars
type GrammarLoader struct {
	configs    map[string]*LanguageConfig
	grammarDir string
}

// LangInfo holds display names for a language
type LangInfo struct {
	Short string // Compact label: "JS", "Py"
	Full  string // Full name: "JavaScript", "Python"
}

// LangDisplay maps internal language names to display names
var LangDisplay = map[string]LangInfo{
	"go":         {"Go", "Go"},
	"python":     {"Py", "Python"},
	"javascript": {"JS", "JavaScript"},
	"typescript": {"TS", "TypeScript"},
	"rust":       {"Rs", "Rust"},
	"ruby":       {"Rb", "Ruby"},
	"c":          {"C", "C"},
	"cpp":        {"C++", "C++"},
	"java":       {"Java", "Java"},
	"swift":      {"Swift", "Swift"},
	"bash":       {"Sh", "Bash"},
	"kotlin":     {"Kt", "Kotlin"},
	"c_sharp":    {"C#", "C#"},
	"php":        {"PHP", "PHP"},
	"dart":       {"Dart", "Dart"},
	"r":          {"R", "R"},
}

// Extension to language mapping
var extToLang = map[string]string{
	".go":    "go",
	".py":    "python",
	".js":    "javascript",
	".jsx":   "javascript",
	".mjs":   "javascript",
	".ts":    "typescript",
	".tsx":   "typescript",
	".rs":    "rust",
	".rb":    "ruby",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".hpp":   "cpp",
	".cc":    "cpp",
	".java":  "java",
	".swift": "swift",
	".sh":    "bash",
	".bash":  "bash",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".cs":    "c_sharp",
	".php":   "php",
	".dart":  "dart",
	".r":     "r",
	".R":     "r",
}

// NewGrammarLoader creates a loader that searches for grammars
func NewGrammarLoader() *GrammarLoader {
	loader := &GrammarLoader{
		configs: make(map[string]*LanguageConfig),
	}

	// Find grammar directory - check env var first (for Homebrew install)
	possibleDirs := []string{}
	if envDir := os.Getenv("CODEMAP_GRAMMAR_DIR"); envDir != "" {
		possibleDirs = append(possibleDirs, envDir)
	}
	possibleDirs = append(possibleDirs,
		filepath.Join(getExecutableDir(), "grammars"),
		filepath.Join(getExecutableDir(), "..", "lib", "grammars"),
		"/opt/homebrew/opt/codemap/libexec/grammars", // Homebrew Apple Silicon
		"/usr/local/opt/codemap/libexec/grammars",    // Homebrew Intel Mac
		"/usr/local/lib/codemap/grammars",
		filepath.Join(os.Getenv("HOME"), ".codemap", "grammars"),
		"./grammars",         // For development
		"./scanner/grammars", // For development from root
	)

	for _, dir := range possibleDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			loader.grammarDir = dir
			break
		}
	}

	return loader
}

// HasGrammars returns true if grammar directory was found
func (l *GrammarLoader) HasGrammars() bool {
	return l.grammarDir != ""
}

// GrammarDir returns the grammar directory path (for diagnostics)
func (l *GrammarLoader) GrammarDir() string {
	return l.grammarDir
}

// LoadLanguage dynamically loads a grammar from .so/.dylib
func (l *GrammarLoader) LoadLanguage(lang string) error {
	if _, exists := l.configs[lang]; exists {
		return nil // Already loaded
	}

	if l.grammarDir == "" {
		return fmt.Errorf("no grammar directory found")
	}

	// OS-specific library extension
	var libExt string
	switch runtime.GOOS {
	case "darwin":
		libExt = ".dylib"
	case "windows":
		libExt = ".dll"
	default:
		libExt = ".so"
	}

	// Load shared library
	libPath := filepath.Join(l.grammarDir, fmt.Sprintf("libtree-sitter-%s%s", lang, libExt))
	lib, err := loadLibrary(libPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", libPath, err)
	}

	// Get language function
	langFunc, err := getLanguageFunc(lib, lang)
	if err != nil {
		return fmt.Errorf("get func for %s: %w", lang, err)
	}
	language := tree_sitter.NewLanguage(langFunc())

	// Load query
	queryBytes, err := queryFiles.ReadFile(fmt.Sprintf("queries/%s.scm", lang))
	if err != nil {
		return fmt.Errorf("no query for %s", lang)
	}

	query, qerr := tree_sitter.NewQuery(language, string(queryBytes))
	if qerr != nil {
		return fmt.Errorf("bad query for %s: %v", lang, qerr)
	}

	l.configs[lang] = &LanguageConfig{Language: language, Query: query}
	return nil
}

// DetectLanguage returns the language name for a file path
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	return extToLang[ext]
}

// AnalyzeFile extracts functions and imports
// detailLevel controls depth of extraction (0=names, 1=signatures, 2=full)
func (l *GrammarLoader) AnalyzeFile(filePath string, detailLevel DetailLevel) (*FileAnalysis, error) {
	lang := DetectLanguage(filePath)
	if lang == "" {
		return nil, nil
	}

	if err := l.LoadLanguage(lang); err != nil {
		return nil, nil // Skip if grammar unavailable
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
	funcBuilder := make(map[uint]*funcCapture)
	typeBuilder := make(map[uint]*typeCapture)

	// Use Matches() API - iterate over query matches
	matches := cursor.Matches(config.Query, tree.RootNode(), content)
	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			captureName := config.Query.CaptureNames()[capture.Index]
			text := strings.Trim(capture.Node.Utf8Text(content), `"'`)
			// Extract line number (1-indexed)
			line := int(capture.Node.StartPosition().Row) + 1

			// Route to appropriate handler based on capture name prefix
			switch {
			case strings.HasPrefix(captureName, "func."):
				handleFuncCapture(funcBuilder, match.Id(), captureName, text, line)
			case strings.HasPrefix(captureName, "type."):
				handleTypeCapture(typeBuilder, match.Id(), captureName, text, line, detailLevel)
			case captureName == "import" || captureName == "module":
				analysis.Imports = append(analysis.Imports, text)
			// Legacy support: plain @function/@method capture (current queries)
			case captureName == "function" || captureName == "method":
				analysis.Functions = append(analysis.Functions, FuncInfo{Name: text, Line: line})
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
		typeInfo := tc.Build(detailLevel, lang)
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
	line     int
}

// Build constructs FuncInfo from captured components
func (fc *funcCapture) Build(detail DetailLevel, lang string) FuncInfo {
	info := FuncInfo{
		Name:       fc.name,
		IsExported: IsExportedName(fc.name, lang),
		Line:       fc.line,
		ParamCount: countParams(fc.params),
	}

	if detail >= DetailSignature && fc.params != "" {
		info.Signature = buildSignature(fc, lang)
	}

	if fc.receiver != "" {
		info.Receiver = fc.receiver
	}

	return info
}

// countParams counts the number of parameters from a parameter list string.
// Returns -1 for variadic functions or empty params, 0 for no params.
func countParams(params string) int {
	params = strings.TrimSpace(params)
	if params == "" || params == "()" {
		return 0
	}

	// Remove outer parentheses if present
	if strings.HasPrefix(params, "(") && strings.HasSuffix(params, ")") {
		params = params[1 : len(params)-1]
	}
	params = strings.TrimSpace(params)
	if params == "" {
		return 0
	}

	// Check for variadic indicators
	if strings.Contains(params, "...") || strings.Contains(params, "*args") || strings.Contains(params, "**kwargs") {
		return -1 // Variadic
	}

	// Count parameters by tracking parentheses/brackets depth
	// to avoid counting commas inside nested types like func(int, int)
	count := 1
	depth := 0
	for _, ch := range params {
		switch ch {
		case '(', '[', '{', '<':
			depth++
		case ')', ']', '}', '>':
			depth--
		case ',':
			if depth == 0 {
				count++
			}
		}
	}

	return count
}

// handleFuncCapture routes function-related captures to builder
func handleFuncCapture(builders map[uint]*funcCapture, matchID uint, name, text string, line int) {
	if builders[matchID] == nil {
		builders[matchID] = &funcCapture{}
	}
	fc := builders[matchID]

	switch name {
	case "func.name":
		fc.name = text
		fc.line = line // Capture line on name (primary identifier)
	case "func.params":
		fc.params = text
	case "func.result":
		fc.result = text
	case "func.receiver":
		fc.receiver = text
	}
}

// typeCapture collects components of a type definition
type typeCapture struct {
	name   string
	kind   TypeKind
	fields string
	line   int
}

// Build constructs TypeInfo from captured components
func (tc *typeCapture) Build(detail DetailLevel, lang string) TypeInfo {
	info := TypeInfo{
		Name:       tc.name,
		Kind:       tc.kind,
		IsExported: IsExportedName(tc.name, lang),
		Line:       tc.line,
	}

	if detail >= DetailFull && tc.fields != "" {
		info.Fields = parseFieldNames(tc.fields)
	}

	return info
}

// handleTypeCapture routes type-related captures to builder
func handleTypeCapture(builders map[uint]*typeCapture, matchID uint, name, text string, line int, detail DetailLevel) {
	if builders[matchID] == nil {
		builders[matchID] = &typeCapture{}
	}
	tc := builders[matchID]

	switch name {
	case "type.name":
		tc.name = text
		tc.line = line // Capture line on name (primary identifier)
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

// parseFieldNames extracts field/member names from raw block text
func parseFieldNames(fieldsText string) []string {
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
			if name != "" && !strings.HasPrefix(name, "//") && !strings.HasPrefix(name, "#") {
				fields = append(fields, name)
			}
		}
	}

	return fields
}

// dedupeFuncs removes duplicate functions by name
func dedupeFuncs(funcs []FuncInfo) []FuncInfo {
	seen := make(map[string]bool)
	var out []FuncInfo
	for _, f := range funcs {
		if !seen[f.Name] {
			seen[f.Name] = true
			out = append(out, f)
		}
	}
	return out
}

// dedupeTypes removes duplicate types by name
func dedupeTypes(types []TypeInfo) []TypeInfo {
	seen := make(map[string]bool)
	var out []TypeInfo
	for _, t := range types {
		if !seen[t.Name] {
			seen[t.Name] = true
			out = append(out, t)
		}
	}
	return out
}

func getExecutableDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func dedupe(s []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
