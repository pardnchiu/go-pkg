package reader

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

func GlobFiles(root, namePattern string) ([]string, error) {
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
		walked, err := WalkFiles(root, ListOption{
			SkipExcluded:      true,
			IgnoreWalkError:   true,
			IncludeNonRegular: true,
		})
		if err != nil {
			return nil, fmt.Errorf("WalkFiles: %w", err)
		}

		var matches []string
		for _, path := range walked {
			walkedParts := strings.Split(path, "/")
			if !isMatch(parts, walkedParts) {
				continue
			}
			matches = append(matches, filepath.Join(root, path))
		}
		return matches, nil
	}

	matches, err := filepath.Glob(filepath.Join(root, namePattern))
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob: %w", err)
	}
	return matches, nil
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
