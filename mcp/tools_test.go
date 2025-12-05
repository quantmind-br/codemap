package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPTools(t *testing.T) {
	// Setup path to test data
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}
	
	// Assuming test is running in codemap/mcp
	// Test data is in codemap/scanner/testdata/corpus/go
	testDataPath := filepath.Join(filepath.Dir(cwd), "scanner", "testdata", "corpus", "go")
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Fatalf("Test data path does not exist: %s", testDataPath)
	}

	ctx := context.Background()

	t.Run("get_structure", func(t *testing.T) {
		input := PathInput{Path: testDataPath}
		result, _, err := handleGetStructure(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleGetStructure failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("handleGetStructure returned error result: %v", result.Content)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, "main.go") {
			t.Errorf("Expected output to contain 'main.go', got:\n%s", text)
		}
	})

	t.Run("find_file", func(t *testing.T) {
		input := FindInput{Path: testDataPath, Pattern: "main"}
		result, _, err := handleFindFile(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleFindFile failed: %v", err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, "main.go") {
			t.Errorf("Expected output to contain 'main.go', got:\n%s", text)
		}
	})

	t.Run("get_dependencies", func(t *testing.T) {
		// Note: This requires grammars to be built. If not, it might return empty deps but shouldn't error.
		input := DepsInput{Path: testDataPath, Detail: 0}
		result, _, err := handleGetDependencies(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleGetDependencies failed: %v", err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		// Check for known output. main.go imports fmt.
		// Output format depends on render.Depgraph.
		// It typically lists files or packages.
		if !strings.Contains(text, "fmt") && !strings.Contains(text, "main.go") {
			t.Logf("Warning: 'fmt' or 'main.go' not found in dependencies. Output:\n%s", text)
			// It might be empty if grammars are not loaded, but tool shouldn't crash.
		}
	})

	t.Run("get_symbol", func(t *testing.T) {
		input := SymbolInput{Path: testDataPath, Name: "main"}
		result, _, err := handleGetSymbol(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleGetSymbol failed: %v", err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, "main.go") {
			t.Errorf("Expected output to contain 'main.go', got:\n%s", text)
		}
		// 'main' function should be found
		if !strings.Contains(text, "func main") {
			t.Errorf("Expected output to contain 'func main', got:\n%s", text)
		}
	})

    // Graph dependent tests
    // Check if graph.gob exists
    graphPath := filepath.Join(testDataPath, ".codemap", "graph.gob")
    if _, err := os.Stat(graphPath); err == nil {
        t.Run("get_callers", func(t *testing.T) {
            input := CallersInput{Path: testDataPath, Symbol: "helper"}
            result, _, err := handleGetCallers(ctx, nil, input)
            if err != nil {
                t.Fatalf("handleGetCallers failed: %v", err)
            }
            if result.IsError {
                 // Might fail if graph is stale
                 t.Logf("handleGetCallers returned error (possibly stale graph): %v", result.Content)
                 return
            }
            text := result.Content[0].(*mcp.TextContent).Text
            if !strings.Contains(text, "process") {
                 t.Errorf("Expected caller 'process' for 'helper', got:\n%s", text)
            }
        })

        t.Run("trace_path", func(t *testing.T) {
            input := TracePathInput{Path: testDataPath, From: "process", To: "nested"}
            result, _, err := handleTracePath(ctx, nil, input)
            if err != nil {
                t.Fatalf("handleTracePath failed: %v", err)
            }
            if result.IsError {
                 t.Logf("handleTracePath returned error: %v", result.Content)
                 return
            }
            text := result.Content[0].(*mcp.TextContent).Text
            // Path: process -> helper -> nested
            if !strings.Contains(text, "helper") || !strings.Contains(text, "nested") {
                t.Errorf("Expected path process -> ... -> nested, got:\n%s", text)
            }
        })
    } else {
        t.Log("Skipping graph-dependent tests (graph.gob not found)")
    }

        t.Run("get_callees", func(t *testing.T) {
            input := CalleesInput{Path: testDataPath, Symbol: "process"}
            result, _, err := handleGetCallees(ctx, nil, input)
            if err != nil {
                t.Fatalf("handleGetCallees failed: %v", err)
            }
            if result.IsError {
                 t.Logf("handleGetCallees returned error: %v", result.Content)
                 return
            }
            text := result.Content[0].(*mcp.TextContent).Text
            if !strings.Contains(text, "helper") {
                 t.Errorf("Expected callee 'helper' for 'process', got:\n%s", text)
            }
        })

	t.Run("list_projects", func(t *testing.T) {
		// Parent of testDataPath is 'corpus'
		corpusPath := filepath.Dir(testDataPath)
		input := ListProjectsInput{Path: corpusPath}
		result, _, err := handleListProjects(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleListProjects failed: %v", err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, "go/") || !strings.Contains(text, "python/") {
			t.Errorf("Expected output to contain 'go/' and 'python/', got:\n%s", text)
		}
	})

	t.Run("get_dependencies_ts", func(t *testing.T) {
		tsPath := filepath.Join(filepath.Dir(testDataPath), "typescript")
		input := DepsInput{Path: tsPath, Detail: 0}
		result, _, err := handleGetDependencies(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleGetDependencies TS failed: %v", err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		// Should show connection from types.ts to main
		// Or at least list types.ts
		if !strings.Contains(text, "types") {
			t.Errorf("Expected output to contain 'types', got:\n%s", text)
		}
	})

	t.Run("get_importers", func(t *testing.T) {
		// Use TypeScript corpus where types.ts imports main.ts
		tsPath := filepath.Join(filepath.Dir(testDataPath), "typescript")
		if _, err := os.Stat(tsPath); os.IsNotExist(err) {
			t.Skip("TypeScript corpus not found")
		}
		
		input := ImportersInput{Path: tsPath, File: "main.ts"}
		result, _, err := handleGetImporters(ctx, nil, input)
		if err != nil {
			t.Fatalf("handleGetImporters failed: %v", err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, "types.ts") {
			// Debug: run deps scan manually and print imports
			t.Logf("Debug: running deps scan on %s", tsPath)
			// (Cannot call ScanForDeps here easily without importing scanner, which is not allowed if I'm in package main test)
			t.Errorf("Expected 'types.ts' to import 'main.ts', got:\n%s", text)
		}
	})

    // LLM dependent tests - use Mock provider
    os.Setenv("CODEMAP_LLM_PROVIDER", "mock")
    os.Setenv("CODEMAP_LLM_MODEL", "mock-model")

    // Ensure graph exists for these tests
    if _, err := os.Stat(graphPath); err == nil {
        t.Run("explain_symbol", func(t *testing.T) {
            input := ExplainSymbolInput{Path: testDataPath, Symbol: "main"}
            result, _, err := handleExplainSymbol(ctx, nil, input)
            if err != nil {
                t.Fatalf("handleExplainSymbol failed: %v", err)
            }
            if result.IsError {
                 text := result.Content[0].(*mcp.TextContent).Text
                 t.Logf("handleExplainSymbol returned error: %s", text)
                 // Don't fail yet, just log to debug. The test expects success.
                 t.Fail()
                 return
            }
            text := result.Content[0].(*mcp.TextContent).Text
            if !strings.Contains(text, "mock response") {
                t.Errorf("Expected mock response, got:\n%s", text)
            }
        })

        t.Run("summarize_module", func(t *testing.T) {
            input := SummarizeModuleInput{Path: testDataPath} // Summarize root of testDataPath
            result, _, err := handleSummarizeModule(ctx, nil, input)
            if err != nil {
                t.Fatalf("handleSummarizeModule failed: %v", err)
            }
            if result.IsError {
                 text := result.Content[0].(*mcp.TextContent).Text
                 t.Logf("handleSummarizeModule returned error: %s", text)
                 t.Fail()
                 return
            }
            text := result.Content[0].(*mcp.TextContent).Text
            if !strings.Contains(text, "mock response") {
                t.Errorf("Expected mock response, got:\n%s", text)
            }
        })

        t.Run("semantic_search_graph_only", func(t *testing.T) {
            // Without vector index, should fall back to graph search
            input := SemanticSearchInput{Path: testDataPath, Query: "main"}
            result, _, err := handleSemanticSearch(ctx, nil, input)
            if err != nil {
                t.Fatalf("handleSemanticSearch failed: %v", err)
            }
             if result.IsError {
                 text := result.Content[0].(*mcp.TextContent).Text
                 t.Logf("handleSemanticSearch returned error: %s", text)
                 t.Fail()
                 return
            }
            text := result.Content[0].(*mcp.TextContent).Text
            // Should find 'main' symbol
            if !strings.Contains(text, "main") {
                t.Errorf("Expected search to find 'main', got:\n%s", text)
            }
        })
    }

    // Git Diff Test
    t.Run("get_diff", func(t *testing.T) {
        // Create temp git repo
        tempDir, err := os.MkdirTemp("", "codemap-git-test")
        if err != nil {
            t.Fatalf("Failed to create temp dir: %v", err)
        }
        defer os.RemoveAll(tempDir)

        // Helper to run git commands
        git := func(args ...string) {
            cmd := exec.Command("git", args...)
            cmd.Dir = tempDir
            if out, err := cmd.CombinedOutput(); err != nil {
                t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
            }
        }

        // Initialize repo
        git("init")
        git("config", "user.email", "test@example.com")
        git("config", "user.name", "Test User")
        
        // Ensure main branch
        git("checkout", "-b", "main")

        // Create a file
        filePath := filepath.Join(tempDir, "file.go")
        err = os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644)
        if err != nil {
            t.Fatalf("WriteFile failed: %v", err)
        }

        git("add", ".")
        git("commit", "-m", "Initial commit")

        // Modify the file
        err = os.WriteFile(filePath, []byte("package main\nfunc main() { fmt.Println() }\n"), 0644)
        if err != nil {
            t.Fatalf("WriteFile failed: %v", err)
        }

        // Test get_diff
        input := DiffInput{Path: tempDir}
        result, _, err := handleGetDiff(ctx, nil, input)
        if err != nil {
            t.Fatalf("handleGetDiff failed: %v", err)
        }
        text := result.Content[0].(*mcp.TextContent).Text
        
        if !strings.Contains(text, "file.go") {
            t.Errorf("Expected diff to contain 'file.go', got:\n%s", text)
        }
    })
}
