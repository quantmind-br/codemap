This project, **codemap**, is a **Command Line Interface (CLI) tool** written in **Go**. It does not expose a traditional HTTP API (like REST or GraphQL). Instead, its "API" surface is defined by its command-line arguments and its machine-readable **JSON output format**, which is designed for programmatic consumption by other tools (like a Python renderer or an LLM context builder).

The core functionality is code analysis, dependency mapping, and visualization of a codebase.

# API Documentation

## APIs Served by This Project

The primary API is the CLI interface, which, when invoked with the `--json` flag, outputs a structured JSON object to standard output (stdout). This JSON output serves as the machine-readable contract.

### Endpoints

The "endpoints" are the different operational modes of the `codemap` CLI tool.

#### 1. Basic Tree/Skyline View

This mode scans the file system and outputs a list of files, optionally with size and diff information.

| Attribute | Detail |
| :--- | :--- |
| **Method** | CLI Execution |
| **Path** | `codemap [path]` |
| **Description** | Scans the directory and outputs file information for visualization (tree or skyline). |
| **Authentication** | None (Relies on local file system permissions). |

**Request**

| Parameter | Location | Type | Description |
| :--- | :--- | :--- | :--- |
| `[path]` | Argument | `string` | The root directory to scan (defaults to `.`). |
| `--skyline` | Flag | `bool` | Enables the city skyline visualization mode. |
| `--animate` | Flag | `bool` | Enables animation (only with `--skyline`). |
| `--diff` | Flag | `bool` | Only includes files changed relative to the reference branch. |
| `--ref` | Flag | `string` | The branch/ref to compare against (default: `main`). |
| `--json` | Flag | `bool` | **Required for API consumption.** Outputs the result as JSON to stdout. |

**Response (Success Format: `scanner.Project` JSON)**

| Field | Type | Description |
| :--- | :--- | :--- |
| `root` | `string` | Absolute path of the project root. |
| `mode` | `string` | The operational mode (`tree` or `skyline`). |
| `animate` | `bool` | Whether animation is enabled. |
| `files` | `array<FileInfo>` | List of files found in the project. |
| `diff_ref` | `string` | The git reference used for diffing (if `--diff` was used). |
| `impact` | `array<ImpactInfo>` | Analysis of potential impact for changed files (if `--diff` was used). |

**`FileInfo` Object Structure:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | `string` | File path relative to the project root. |
| `size` | `int64` | File size in bytes. |
| `ext` | `string` | File extension. |
| `tokens` | `int` | Estimated token count for the file. |
| `is_new` | `bool` | True if the file is newly added in the diff. |
| `added` | `int` | Lines added in the diff. |
| `removed` | `int` | Lines removed in the diff. |

**Example (JSON Output):**

```json
{
  "root": ".",
  "mode": "tree",
  "animate": false,
  "files": [
    {
      "path": "main.go",
      "size": 3500,
      "ext": ".go",
      "tokens": 1000
    },
    // ... more files
  ],
  "diff_ref": "",
  "impact": null
}
```

---

#### 2. Dependency Graph View

This mode performs deep code analysis using tree-sitter grammars to extract functions, types, and imports, mapping the project's internal structure and external dependencies.

| Attribute | Detail |
| :--- | :--- |
| **Method** | CLI Execution |
| **Path** | `codemap --deps [path]` |
| **Description** | Scans the code to build a dependency graph (functions, types, imports). |
| **Authentication** | None (Relies on local file system permissions). |

**Request**

| Parameter | Location | Type | Description |
| :--- | :--- | :--- | :--- |
| `[path]` | Argument | `string` | The root directory to scan (defaults to `.`). |
| `--deps` | Flag | `bool` | **Required.** Enables dependency graph mode. |
| `--detail` | Flag | `int` | Level of detail for extracted symbols: `0` (names only), `1` (names + signatures), `2` (signatures + type fields). |
| `--api` | Flag | `bool` | Compact view showing only public API surface (only for human-readable output). |
| `--diff` | Flag | `bool` | Only analyzes files changed relative to the reference branch. |
| `--ref` | Flag | `string` | The branch/ref to compare against (default: `main`). |
| `--json` | Flag | `bool` | **Required for API consumption.** Outputs the result as JSON to stdout. |

**Response (Success Format: `scanner.DepsProject` JSON)**

| Field | Type | Description |
| :--- | :--- | :--- |
| `root` | `string` | Absolute path of the project root. |
| `mode` | `string` | The operational mode (`deps`). |
| `files` | `array<FileAnalysis>` | Detailed analysis for each scanned file. |
| `external_deps` | `map<string, array<string>>` | Map of external dependencies (e.g., Go modules, Python packages) and the files that import them. |
| `diff_ref` | `string` | The git reference used for diffing (if `--diff` was used). |
| `detail_level` | `int` | The detail level used for the analysis (0, 1, or 2). |

**`FileAnalysis` Object Structure:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | `string` | File path relative to the project root. |
| `language` | `string` | Detected programming language (e.g., `go`, `python`). |
| `functions` | `array<FuncInfo>` | List of functions and methods defined in the file. |
| `types` | `array<TypeInfo>` | List of type definitions (structs, classes, interfaces). |
| `imports` | `array<string>` | List of imported modules/packages. |

**`FuncInfo` Object Structure:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Function/method name. |
| `signature` | `string` | Full function signature (if `detail_level >= 1`). |
| `receiver` | `string` | Receiver type for methods (e.g., `(*Project)`). |
| `exported` | `bool` | True if the symbol is publicly visible (based on language rules). |
| `line` | `int` | Line number of the definition. |

**`TypeInfo` Object Structure:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Type name. |
| `kind` | `TypeKind` | Category of the type (`struct`, `class`, `interface`, etc.). |
| `fields` | `array<string>` | Field names (if `detail_level = 2`). |
| `methods` | `array<string>` | Method names associated with the type. |
| `exported` | `bool` | True if the type is publicly visible. |
| `line` | `int` | Line number of the definition. |

**Example (JSON Output):**

```json
{
  "root": ".",
  "mode": "deps",
  "files": [
    {
      "path": "main.go",
      "language": "go",
      "functions": [
        {
          "name": "main",
          "signature": "func main()",
          "exported": false,
          "line": 19
        },
        {
          "name": "runDepsMode",
          "signature": "func runDepsMode(absRoot, root string, gitignore *ignore.GitIgnore, jsonMode bool, diffRef string, changedFiles map[string]bool, detailLevel int, apiMode bool)",
          "exported": false,
          "line": 137
        }
      ],
      "types": [],
      "imports": [
        "encoding/json",
        "flag",
        "fmt",
        "os",
        "path/filepath",
        "codemap/render",
        "codemap/scanner",
        "github.com/sabhiram/go-gitignore"
      ]
    }
  ],
  "external_deps": {
    "github.com/sabhiram/go-gitignore": [
      "main.go"
    ]
  },
  "detail_level": 1
}
```

### Authentication & Security

Since `codemap` is a local CLI tool, it does not implement traditional network-based authentication.

*   **Authentication:** None. Execution is authorized by the user's shell environment and file system permissions.
*   **Security:** The tool operates entirely on the local file system. Security relies on the operating system's access control mechanisms. It reads code files and executes the `git` command-line tool.

### Rate Limiting & Constraints

*   **Rate Limiting:** Not applicable.
*   **Constraints:** The `--deps` mode requires pre-compiled **tree-sitter grammars** to be available on the system. If grammars are missing, the tool will exit with an error and instructions on how to install them.

## External API Dependencies

The project has one primary external dependency that involves executing an external program.

### Services Consumed

#### 1. Git Command Line Interface (CLI)

The project relies on the local installation of the `git` CLI tool to perform version control operations when the `--diff` flag is used.

| Attribute | Detail |
| :--- | :--- |
| **Service Name & Purpose** | **Git CLI:** Used to determine which files have changed between the current working directory and a specified reference (e.g., `main` branch). |
| **Base URL/Configuration** | Local execution of the `git` binary (e.g., `/usr/bin/git`). |
| **Endpoints Used** | The Go code executes `git diff --name-status <ref> -- .` to get the list of changed files and their status (Added, Modified, Deleted). |
| **Authentication Method** | None. Relies on the local Git configuration and repository access permissions. |
| **Error Handling** | If the `git` command fails (e.g., invalid reference, not a git repository), the `scanner.GitDiffInfo` function returns an error, and `main.go` prints the error to `stderr` and exits with status code 1. |
| **Retry/Circuit Breaker Configuration** | None. The command is executed once. |

### Integration Patterns

*   **Library Integration:** The project uses the `github.com/sabhiram/go-gitignore` library for file exclusion logic, which is a standard Go library integration pattern.
*   **External Process Execution:** The integration with Git is done via executing an external process and capturing its standard output, a common pattern for CLI tools interacting with system utilities.

## Available Documentation

The project includes internal documentation and development plans, but no formal, external API specification files (like OpenAPI/Swagger, GraphQL schemas, or Protocol Buffers).

| Path | Description | Quality Evaluation |
| :--- | :--- | :--- |
| `main.go` | Defines the CLI interface and command-line flags, which serve as the primary API definition. | **High.** Clear definition of inputs and modes. |
| `scanner/types.go` | Defines the `Project` and `DepsProject` structs, which are the JSON output schemas (the API contract). | **High.** Explicitly defines the machine-readable output structure. |
| `/.ai/docs/api_analysis.md` | Internal documentation for AI agents, likely containing a preliminary analysis of the API surface. | **Medium.** Useful for context, but not the canonical source. |
| `README.md` | Provides usage examples for the CLI, which implicitly documents the API's invocation. | **High.** Excellent for practical usage. |
| `scanner/git.go` | Implements the logic for interacting with the external Git CLI. | **High.** Confirms and details the external dependency. |