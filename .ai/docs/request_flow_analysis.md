# Request Flow Analysis
The `codemap` application is a Command Line Interface (CLI) tool written in Go. Its "request flow" is the process of parsing command-line arguments, executing the requested analysis mode, and rendering the output. It does not handle external network requests, so the analysis is adapted to the CLI execution lifecycle.

## Entry Points Overview
The system has a single entry point:
*   **`main.go:main()`**: This function is the application's starting point. It is responsible for parsing command-line flags, determining the execution mode (tree, skyline, or dependency graph), and orchestrating the file scanning and rendering process.

## Request Routing Map
The "routing" is a decision tree based on the presence and combination of command-line flags.

| Request (Flags) | Preprocessing/Mode Selection | Core Logic Handler | Final Output/Renderer |
| :--- | :--- | :--- | :--- |
| `--help` | Flag parsing | Print help message | `os.Exit(0)` |
| `--deps` | `runDepsMode` call | `scanner.ScanForDeps` | `render.Depgraph` or `render.APIView` |
| `--skyline` | Set `mode = "skyline"` | `scanner.ScanFiles` | `render.Skyline` |
| (Default) | Set `mode = "tree"` | `scanner.ScanFiles` | `render.Tree` |
| Any Mode + `--diff` | `scanner.GitDiffInfo` | `scanner.FilterToChangedWithInfo`, `scanner.AnalyzeImpact` | (Mode-specific renderer) |
| Any Mode + `--json` | (Mode-specific logic) | (Mode-specific logic) | `json.NewEncoder(os.Stdout).Encode` |

## Middleware Pipeline
In the context of this CLI, the "middleware pipeline" consists of sequential preprocessing steps applied to the input path and flags before the core analysis begins.

1.  **Argument Parsing:** `flag.Parse()` reads all command-line arguments and sets the global flag variables.
2.  **Root Path Resolution:** The positional argument (or `.` by default) is resolved to an absolute path using `filepath.Abs()`.
3.  **Gitignore Loading:** `scanner.LoadGitignore(root)` reads the project's `.gitignore` file to establish file exclusion rules for the subsequent scan.
4.  **Diff Analysis (`--diff`):** If the `--diff` flag is present, `scanner.GitDiffInfo(absRoot, *diffRef)` is executed. This step uses Git to determine the set of files that have changed relative to the specified reference branch (defaulting to `main`).
5.  **Impact Analysis (Post-Scan, if `--diff`):** For non-dependency modes, after `scanner.ScanFiles` completes, `scanner.FilterToChangedWithInfo` and `scanner.AnalyzeImpact` are called to annotate the scanned files with diff and impact information.

## Controller/Handler Analysis
The application logic is split into two main "controllers" based on the execution mode:

### 1. `runDepsMode` (Dependency Graph Handler)
This function handles the `--deps` flag and is responsible for deep code analysis.
*   **Grammar Check:** It first verifies the availability of tree-sitter grammars using `scanner.NewGrammarLoader().HasGrammars()`. If grammars are missing, it prints an error and exits.
*   **Core Analysis:** It calls `scanner.ScanForDeps(root, gitignore, loader, detailLevel)` to perform syntax-aware analysis, extracting functions, types, and dependencies based on the requested `detailLevel`.
*   **External Dependencies:** It calls `scanner.ReadExternalDeps(absRoot)` (defined in `scanner/deps.go`) to parse manifest files (`go.mod`, `package.json`, `requirements.txt`, etc.) and collect external library dependencies.
*   **Response:** The results are packaged into a `scanner.DepsProject` struct and passed to either `render.APIView` (if `--api` is set), `render.Depgraph`, or JSON encoding.

### 2. Main Logic Block (Tree/Skyline Handler)
This block handles the default tree view and the skyline visualization.
*   **Core Analysis:** It calls `scanner.ScanFiles(root, gitignore)` to recursively walk the directory tree, collecting basic file metadata (size, path, etc.).
*   **Response:** The results are packaged into a `scanner.Project` struct and passed to either `render.Skyline` (if `--skyline` is set), `render.Tree`, or JSON encoding.

## Authentication & Authorization Flow
Not applicable. As a local CLI tool, `codemap` operates on the local file system and does not implement any authentication or authorization mechanisms.

## Error Handling Pathways
Error handling is immediate and results in application termination with a non-zero exit code (`os.Exit(1)`), printing the error to `os.Stderr`.

*   **Initialization Errors:** Errors during `filepath.Abs` or `scanner.GitDiffInfo` (e.g., invalid git reference) cause immediate exit.
*   **Scanning Errors:** Errors during `scanner.ScanFiles` (file system traversal failure) or `scanner.ScanForDeps` (tree-sitter parsing failure) cause immediate exit.
*   **Configuration Errors:** If `--deps` is used without required tree-sitter grammars, a detailed warning is printed, and the application exits.

## Request Lifecycle Diagram

```mermaid
graph TD
    A[Start: main.go:main()] --> B{Parse Flags: flag.Parse()};
    B --> C{Is --help?};
    C -- Yes --> D[Print Help & Exit];
    C -- No --> E[Resolve Root Path];
    E --> F[Load .gitignore];
    F --> G{Is --diff?};
    G -- Yes --> H[scanner.GitDiffInfo];
    G -- No --> I{Is --deps?};

    H --> I;

    I -- Yes --> J[Call runDepsMode];
    J --> K{Check Grammars};
    K -- Missing --> L[Error: Grammars Missing & Exit];
    K -- OK --> M[scanner.ScanForDeps];
    M --> N[scanner.ReadExternalDeps];
    N --> O{Is --json?};
    O -- Yes --> P[JSON Encode DepsProject];
    O -- No --> Q{Is --api?};
    Q -- Yes --> R[render.APIView];
    Q -- No --> S[render.Depgraph];
    S --> Z[End];
    R --> Z;
    P --> Z;
    L --> Z;

    I -- No --> T[scanner.ScanFiles];
    T --> U{Is --diff?};
    U -- Yes --> V[Filter & AnalyzeImpact];
    U -- No --> W[Build Project Struct];
    V --> W;
    W --> X{Is --json?};
    X -- Yes --> P;
    X -- No --> Y{Is --skyline?};
    Y -- Yes --> R1[render.Skyline];
    Y -- No --> S1[render.Tree];
    R1 --> Z;
    S1 --> Z;
```