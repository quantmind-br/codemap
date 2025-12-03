package scanner

import (
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

// IgnoredDirs are directories to skip during scanning
var IgnoredDirs = map[string]bool{
	".git":           true,
	"node_modules":   true,
	"vendor":         true,
	"Pods":           true,
	"build":          true,
	"DerivedData":    true,
	".idea":          true,
	".vscode":        true,
	"__pycache__":    true,
	".DS_Store":      true,
	"venv":           true,
	".venv":          true,
	".env":           true,
	".pytest_cache":  true,
	".mypy_cache":    true,
	".ruff_cache":    true,
	".coverage":      true,
	"htmlcov":        true,
	".tox":           true,
	"dist":           true,
	".next":          true,
	".nuxt":          true,
	"target":         true,
	".gradle":        true,
	".cargo":         true,
	".grammar-build": true,
	"grammars":       true,
}

// WalkOptions configures the file walking behavior.
type WalkOptions struct {
	// Gitignore patterns to apply (can be nil)
	Gitignore *ignore.GitIgnore

	// LanguageFilter if true, only visits files with supported languages
	LanguageFilter bool
}

// WalkFunc is the callback function type for WalkFiles.
// It receives the absolute path, relative path, and file info for each file.
// Return filepath.SkipDir to skip a directory, or any other error to stop walking.
type WalkFunc func(absPath, relPath string, info os.FileInfo) error

// WalkFiles walks the directory tree and calls fn for each file.
// This decouples file system traversal from data generation, allowing
// callers to process files as needed without duplicating walk logic.
func WalkFiles(root string, opts WalkOptions, fn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip if matched by common ignore patterns
		if info.IsDir() {
			if IgnoredDirs[info.Name()] {
				return filepath.SkipDir
			}
		} else {
			if IgnoredDirs[info.Name()] {
				return nil
			}
		}

		// Skip if matched by .gitignore
		if opts.Gitignore != nil && opts.Gitignore.MatchesPath(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (we only process files)
		if info.IsDir() {
			return nil
		}

		// Apply language filter if enabled
		if opts.LanguageFilter && DetectLanguage(path) == "" {
			return nil
		}

		// Call the user-provided function
		return fn(path, relPath, info)
	})
}

// LoadGitignore loads .gitignore from root if it exists
func LoadGitignore(root string) *ignore.GitIgnore {
	gitignorePath := filepath.Join(root, ".gitignore")

	if _, err := os.Stat(gitignorePath); err == nil {
		if gitignore, err := ignore.CompileIgnoreFile(gitignorePath); err == nil {
			return gitignore
		}
	}

	return nil
}

// ScanFiles walks the directory tree and returns all files.
// This is a convenience wrapper around WalkFiles for collecting FileInfo.
func ScanFiles(root string, gitignore *ignore.GitIgnore) ([]FileInfo, error) {
	var files []FileInfo

	opts := WalkOptions{
		Gitignore: gitignore,
	}

	err := WalkFiles(root, opts, func(absPath, relPath string, info os.FileInfo) error {
		files = append(files, FileInfo{
			Path:   relPath,
			Size:   info.Size(),
			Ext:    filepath.Ext(absPath),
			Tokens: EstimateTokens(info.Size()),
		})
		return nil
	})

	return files, err
}

// ScanForDeps walks the directory tree and analyzes files for dependencies.
// This is a convenience wrapper around WalkFiles for collecting FileAnalysis.
// detailLevel controls the depth of extraction (0=names, 1=signatures, 2=full)
func ScanForDeps(root string, gitignore *ignore.GitIgnore, loader *GrammarLoader, detailLevel DetailLevel) ([]FileAnalysis, error) {
	var analyses []FileAnalysis

	opts := WalkOptions{
		Gitignore:      gitignore,
		LanguageFilter: true, // Only analyze supported languages
	}

	err := WalkFiles(root, opts, func(absPath, relPath string, info os.FileInfo) error {
		// Analyze file with the specified detail level
		analysis, err := loader.AnalyzeFile(absPath, detailLevel)
		if err != nil || analysis == nil {
			return nil // Skip files that can't be analyzed
		}

		// Use relative path in output
		analysis.Path = relPath
		analyses = append(analyses, *analysis)

		return nil
	})

	return analyses, err
}
