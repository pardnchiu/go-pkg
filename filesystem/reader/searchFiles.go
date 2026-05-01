package reader

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pardnchiu/go-pkg/filesystem"
)

var binaryExts = map[string]bool{
	".exe":   true,
	".bin":   true,
	".so":    true,
	".dylib": true,
	".dll":   true,
	".o":     true,
	".a":     true,
}

func SearchFiles(root, namePattern string, filePatterns []string, maxSize int64, opts ...ListOption) ([]string, error) {
	regex, err := regexp.Compile(namePattern)
	if err != nil {
		return nil, fmt.Errorf("regexp.Compile: %w", err)
	}

	if maxSize <= 0 {
		maxSize = 1 << 20
	}

	opt := getListOption(opts)
	var absRoot string
	if opt.SkipExcluded {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("filepath.Abs: %w", err)
		}
		absRoot = abs
	}

	var matches []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if opt.IgnoreWalkError {
				slog.Warn("filepath.WalkDir", slog.String("error", err.Error()))
				return nil
			}
			return err
		}

		if path == root {
			return nil
		}

		basePath := filepath.Base(path)
		if strings.HasPrefix(basePath, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if opt.SkipExcluded {
				abs, err := filepath.Abs(path)
				if err != nil {
					return fmt.Errorf("filepath.Abs: %w", err)
				}
				if filesystem.IsExcluded(absRoot, abs) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if binaryExts[filepath.Ext(path)] {
			return nil
		}

		if opt.SkipExcluded {
			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("filepath.Abs: %w", err)
			}
			if filesystem.IsExcluded(absRoot, abs) {
				return nil
			}
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		if len(filePatterns) > 0 {
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			if !isMatch(filePatterns, parts) {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !opt.IncludeNonRegular && !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > maxSize {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if bytes.IndexByte(data[:min(len(data), 512)], 0) >= 0 {
			return nil
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(make([]byte, 0, 64*1024), len(data))
		for scanner.Scan() {
			if regex.MatchString(scanner.Text()) {
				matches = append(matches, path)
				break
			}
		}
		return nil
	})
	return matches, err
}
