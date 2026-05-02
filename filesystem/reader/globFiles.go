package reader

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func GlobFiles(root, namePattern string) ([]File, error) {
	parts := strings.Split(namePattern, "/")

	for _, p := range parts {
		if p == "**" {
			continue
		}
		if _, err := filepath.Match(p, ""); err != nil {
			return nil, fmt.Errorf("filepath.Match: %w", err)
		}
	}

	if slices.Contains(parts, "**") {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("filepath.Abs: %w", err)
		}
		walked, err := WalkFiles(root, ListOption{
			SkipExcluded:      true,
			IgnoreWalkError:   true,
			IncludeNonRegular: true,
		})
		if err != nil {
			return nil, fmt.Errorf("WalkFiles: %w", err)
		}
		files := make([]File, 0, len(walked))
		for _, f := range walked {
			rel, err := filepath.Rel(absRoot, f.Path)
			if err != nil {
				continue
			}
			walkedParts := strings.Split(filepath.ToSlash(rel), "/")
			if !isMatch(parts, walkedParts) {
				continue
			}
			files = append(files, f)
		}
		return files, nil
	}

	matches, err := filepath.Glob(filepath.Join(root, namePattern))
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob: %w", err)
	}
	files := make([]File, 0, len(matches))
	for _, full := range matches {
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		files = append(files, newFile(full, info))
	}
	return files, nil
}

func isMatch(patterns, parts []string) bool {
	if len(patterns) == 0 {
		return len(parts) == 0
	}

	pattern := patterns[0]
	if pattern == "**" {
		rest := patterns[1:]
		for len(rest) > 0 && rest[0] == "**" {
			rest = rest[1:]
		}
		if len(rest) == 0 {
			return true
		}
		for i := 0; i <= len(parts); i++ {
			if isMatch(rest, parts[i:]) {
				return true
			}
		}
		return false
	}

	if len(parts) == 0 {
		return false
	}

	match, err := filepath.Match(pattern, parts[0])
	if err != nil || !match {
		return false
	}
	return isMatch(patterns[1:], parts[1:])
}
