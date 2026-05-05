package gmcore_finder

// Package gmcore_finder provides file searching and content search utilities.
//
// Examples:
//
//	// Find files by name
//	files, _ := FindByName("/path", "*.go", nil)
//
//	// Find files by extension
//	files, _ := FindByExtension("/path", ".txt", nil)
//
//	// Search file contents
//	matches, _ := Search("/path", "pattern", nil)
//	for _, m := range matches {
//	    fmt.Printf("%s:%d: %s\n", m.Path, m.Line, m.Content)
//	}

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FindOptions struct {
	Recursive    bool
	MaxDepth     int
	IncludeFiles bool
	IncludeDirs  bool
	MatchHidden  bool
	Regex        bool
}

type SearchOptions struct {
	Recursive    bool
	MaxDepth     int
	CaseSensitive bool
	WholeWord    bool
	Regex        bool
	Include      []string
	Exclude      []string
}

type FileMatch struct {
	Path    string
	Line    int
	Content string
}

type DirMatch struct {
	Path  string
	Depth int
}

func DefaultFindOptions() *FindOptions {
	return &FindOptions{
		Recursive:    true,
		IncludeFiles: true,
		IncludeDirs:  true,
		MatchHidden:  false,
	}
}

func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		Recursive:    true,
		CaseSensitive: false,
		WholeWord:    false,
		Regex:        false,
	}
}

func FindFiles(root string, opts *FindOptions) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}

	if opts == nil {
		opts = DefaultFindOptions()
	}

	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		depth := len(strings.Split(relPath, string(filepath.Separator)))

		if !opts.MatchHidden && isHidden(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if opts.MaxDepth > 0 && depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			if opts.IncludeDirs {
				matches = append(matches, path)
			}
			return nil
		}

		if opts.IncludeFiles {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("find failed: %w", err)
	}

	return matches, nil
}

func FindDirs(root string, opts *FindOptions) ([]DirMatch, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}

	if opts == nil {
		opts = DefaultFindOptions()
		opts.IncludeDirs = true
		opts.IncludeFiles = false
	}

	var matches []DirMatch
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		depth := len(strings.Split(relPath, string(filepath.Separator)))

		if !opts.MatchHidden && isHidden(relPath) {
			return nil
		}

		if opts.MaxDepth > 0 && depth > opts.MaxDepth {
			return nil
		}

		if info.IsDir() {
			matches = append(matches, DirMatch{
				Path:  path,
				Depth: depth,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("find dirs failed: %w", err)
	}

	return matches, nil
}

func SearchInFile(path string, pattern string, opts *SearchOptions) ([]FileMatch, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var matches []FileMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		content := scanner.Text()

		matched, err := matchPattern(content, pattern, opts)
		if err != nil {
			continue
		}

		if matched {
			matches = append(matches, FileMatch{
				Path:    path,
				Line:    lineNum,
				Content: content,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return matches, nil
}

func Search(root string, pattern string, opts *SearchOptions) ([]FileMatch, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	if opts == nil {
		opts = DefaultSearchOptions()
	}

	var matches []FileMatch

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		depth := len(strings.Split(relPath, string(filepath.Separator)))

		if !opts.Recursive && depth > 1 {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if opts.MaxDepth > 0 && depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldExclude(relPath, opts.Exclude) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if len(opts.Include) > 0 && !shouldInclude(relPath, opts.Include) {
			return nil
		}

		fileMatches, err := SearchInFile(path, pattern, opts)
		if err != nil {
			return nil
		}

		matches = append(matches, fileMatches...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return matches, nil
}

func CountMatches(root string, pattern string, opts *SearchOptions) (int, error) {
	if root == "" {
		return 0, fmt.Errorf("root path cannot be empty")
	}
	if pattern == "" {
		return 0, fmt.Errorf("pattern cannot be empty")
	}

	if opts == nil {
		opts = DefaultSearchOptions()
	}

	var count int

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		matches, err := SearchInFile(path, pattern, opts)
		if err != nil {
			return nil
		}

		count += len(matches)
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("count matches failed: %w", err)
	}

	return count, nil
}

func FindByName(root string, name string, opts *FindOptions) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	if opts == nil {
		opts = DefaultFindOptions()
	}

	namePattern := strings.ToLower(name)

	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		depth := len(strings.Split(relPath, string(filepath.Separator)))

		if !opts.MatchHidden && isHidden(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if opts.MaxDepth > 0 && depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fileName := strings.ToLower(info.Name())

		isMatch := false
		if opts.Regex {
			matched, _ := regexp.MatchString(name, info.Name())
			isMatch = matched
		} else {
			isMatch = strings.Contains(fileName, namePattern)
		}

		if !isMatch {
			return nil
		}

		if info.IsDir() {
			if opts.IncludeDirs {
				matches = append(matches, path)
			}
			if !opts.Recursive {
				return filepath.SkipDir
			}
		} else if opts.IncludeFiles {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("find by name failed: %w", err)
	}

	return matches, nil
}

func FindByExtension(root string, ext string, opts *FindOptions) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}
	if ext == "" {
		return nil, fmt.Errorf("extension cannot be empty")
	}

	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	extLower := strings.ToLower(ext)

	if opts == nil {
		opts = DefaultFindOptions()
	}

	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		depth := len(strings.Split(relPath, string(filepath.Separator)))

		if !opts.MatchHidden && isHidden(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if opts.MaxDepth > 0 && depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		fileExt := strings.ToLower(filepath.Ext(info.Name()))
		if fileExt == extLower && opts.IncludeFiles {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("find by extension failed: %w", err)
	}

	return matches, nil
}

func matchPattern(content string, pattern string, opts *SearchOptions) (bool, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}

	if opts.Regex {
		if opts.CaseSensitive {
			return regexp.MatchString(pattern, content)
		}
		matched, _ := regexp.MatchString("(?i)"+pattern, content)
		return matched, nil
	}

	if opts.WholeWord {
		wordPattern := `\b` + regexp.QuoteMeta(pattern) + `\b`
		if opts.CaseSensitive {
			matched, _ := regexp.MatchString(wordPattern, content)
			return matched, nil
		}
		matched, _ := regexp.MatchString("(?i)"+wordPattern, content)
		return matched, nil
	}

	if opts.CaseSensitive {
		return strings.Contains(content, pattern), nil
	}

	return strings.Contains(strings.ToLower(content), strings.ToLower(pattern)), nil
}

func isHidden(path string) bool {
	if path == "." {
		return false
	}
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if len(part) > 0 && part[0] == '.' {
			return true
		}
	}
	return false
}

func shouldInclude(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
	}

	return false
}

func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
	}
	return false
}
