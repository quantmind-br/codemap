package analyze

import (
	"fmt"
	"strings"

	"codemap/graph"
)

// Prompt templates for code explanation and summarization.
// These are designed to be token-efficient while producing useful summaries.

// ExplainSymbolPrompt generates a prompt for explaining a code symbol.
func ExplainSymbolPrompt(source *SymbolSource) []Message {
	systemPrompt := `You are a code documentation expert. Explain code clearly and concisely.
Focus on:
1. What the code does (purpose)
2. How it works (key logic)
3. Important parameters and return values
4. Any notable patterns or edge cases

Be brief but comprehensive. Use technical terms appropriately.`

	var userContent strings.Builder

	userContent.WriteString(fmt.Sprintf("Explain this %s %s", source.Language, kindName(source.Node)))

	if source.Node.Name != "" {
		userContent.WriteString(fmt.Sprintf(" named `%s`", source.Node.Name))
	}

	if source.Node.Package != "" {
		userContent.WriteString(fmt.Sprintf(" in package `%s`", source.Node.Package))
	}

	userContent.WriteString(":\n\n")
	userContent.WriteString("```" + source.Language + "\n")
	userContent.WriteString(source.Source)
	userContent.WriteString("\n```")

	// Add signature if available
	if source.Node.Signature != "" && !strings.Contains(source.Source, source.Node.Signature) {
		userContent.WriteString(fmt.Sprintf("\n\nSignature: `%s`", source.Node.Signature))
	}

	// Add docstring if available
	if source.Node.DocString != "" {
		userContent.WriteString(fmt.Sprintf("\n\nExisting documentation:\n%s", source.Node.DocString))
	}

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent.String()},
	}
}

// ExplainSymbolWithContextPrompt generates a prompt including surrounding context.
func ExplainSymbolWithContextPrompt(source *SymbolSource) []Message {
	if source.Context == nil {
		return ExplainSymbolPrompt(source)
	}

	systemPrompt := `You are a code documentation expert. Explain code clearly and concisely.
Focus on:
1. What the code does (purpose)
2. How it works (key logic)
3. How it relates to surrounding code
4. Important parameters and return values

Be brief but comprehensive. Use technical terms appropriately.`

	var userContent strings.Builder

	userContent.WriteString(fmt.Sprintf("Explain this %s %s", source.Language, kindName(source.Node)))

	if source.Node.Name != "" {
		userContent.WriteString(fmt.Sprintf(" named `%s`", source.Node.Name))
	}

	userContent.WriteString(":\n\n")

	// Add context before
	if source.Context.Before != "" {
		userContent.WriteString("Context (code before):\n```" + source.Language + "\n")
		userContent.WriteString(source.Context.Before)
		userContent.WriteString("\n```\n\n")
	}

	// Main source
	userContent.WriteString("Target code:\n```" + source.Language + "\n")
	userContent.WriteString(source.Source)
	userContent.WriteString("\n```")

	// Add context after
	if source.Context.After != "" {
		userContent.WriteString("\n\nContext (code after):\n```" + source.Language + "\n")
		userContent.WriteString(source.Context.After)
		userContent.WriteString("\n```")
	}

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent.String()},
	}
}

// SummarizeModulePrompt generates a prompt for summarizing a module/directory.
func SummarizeModulePrompt(modulePath string, sources []*SymbolSource) []Message {
	systemPrompt := `You are a code documentation expert. Summarize code modules clearly and concisely.
Provide:
1. Module purpose (what does this module do?)
2. Key components (main files, classes, functions)
3. Dependencies and relationships
4. Public API (exported functions/types)

Be structured and scannable. Use bullet points where appropriate.`

	var userContent strings.Builder

	userContent.WriteString(fmt.Sprintf("Summarize this module at `%s`:\n\n", modulePath))

	// List files with their key exports
	userContent.WriteString("**Files:**\n")
	for _, source := range sources {
		userContent.WriteString(fmt.Sprintf("\n### %s\n", source.Node.Name))
		userContent.WriteString("```" + source.Language + "\n")

		// Include first 100 lines or less
		lines := strings.Split(source.Source, "\n")
		if len(lines) > 100 {
			userContent.WriteString(strings.Join(lines[:100], "\n"))
			userContent.WriteString(fmt.Sprintf("\n// ... (%d more lines)\n", len(lines)-100))
		} else {
			userContent.WriteString(source.Source)
		}

		userContent.WriteString("\n```\n")
	}

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent.String()},
	}
}

// SummarizeFilesPrompt generates a prompt for summarizing multiple files concisely.
func SummarizeFilesPrompt(files []string, overviews []string) []Message {
	systemPrompt := `You are a code documentation expert. Create a brief module overview.
Output should be:
1. One-sentence module purpose
2. Key components (2-5 bullet points)
3. Main dependencies

Be extremely concise. Target 100-200 words total.`

	var userContent strings.Builder

	userContent.WriteString("Summarize this module based on file overviews:\n\n")

	for i, file := range files {
		userContent.WriteString(fmt.Sprintf("**%s:**\n%s\n\n", file, overviews[i]))
	}

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent.String()},
	}
}

// QuickExplainPrompt generates a minimal prompt for quick explanations.
func QuickExplainPrompt(source *SymbolSource) []Message {
	systemPrompt := "Explain code in one paragraph. Be concise."

	userContent := fmt.Sprintf("```%s\n%s\n```\n\nWhat does this do?",
		source.Language, source.Source)

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}
}

// CallGraphExplainPrompt explains a function in context of its callers and callees.
func CallGraphExplainPrompt(source *SymbolSource, callers, callees []string) []Message {
	systemPrompt := `You are a code documentation expert. Explain this function's role in the codebase.
Focus on:
1. What it does
2. Why callers might use it
3. What it depends on (callees)

Be concise but informative.`

	var userContent strings.Builder

	userContent.WriteString(fmt.Sprintf("Explain `%s`:\n\n", source.Node.Name))
	userContent.WriteString("```" + source.Language + "\n")
	userContent.WriteString(source.Source)
	userContent.WriteString("\n```\n")

	if len(callers) > 0 {
		userContent.WriteString(fmt.Sprintf("\nCalled by: %s\n", strings.Join(callers, ", ")))
	}

	if len(callees) > 0 {
		userContent.WriteString(fmt.Sprintf("\nCalls: %s\n", strings.Join(callees, ", ")))
	}

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent.String()},
	}
}

// kindName returns a human-readable name for a node kind.
func kindName(node *graph.Node) string {
	if node == nil {
		return "code"
	}

	switch node.Kind {
	case graph.KindFunction:
		return "function"
	case graph.KindMethod:
		return "method"
	case graph.KindType:
		return "type"
	case graph.KindFile:
		return "file"
	case graph.KindPackage:
		return "package"
	case graph.KindVariable:
		return "variable"
	default:
		return "code"
	}
}
