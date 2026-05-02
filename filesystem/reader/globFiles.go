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

	var paths []string
	if slices.Contains(parts, "**") {
		walked, err := WalkFiles(root, ListOption{
			SkipExcluded:      true,
			IgnoreWalkError:   true,
			IncludeNonRegular: true,
		})
		if err != nil {
			return nil, fmt.Errorf("WalkFiles: %w", err)
		}
		for _, path := range walked {
			walkedParts := strings.Split(path, "/")
			if !isMatch(parts, walkedParts) {
				continue
			}
			paths = append(paths, filepath.Join(root, path))
		}
	} else {
		matches, err := filepath.Glob(filepath.Join(root, namePattern))
		if err != nil {
			return nil, fmt.Errorf("filepath.Glob: %w", err)
		}
		paths = matches
	}

	files := make([]File, 0, len(paths))
	for _, full := range paths {
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
