package reader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-pkg/filesystem"
)

func WalkFiles(root string, opts ...ListOption) ([]string, error) {
	opt := getListOption(opts)
	var absRoot string
	if opt.SkipExcluded {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("filepath.Abs: %w", err)
		}
		absRoot = abs
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if opt.IgnoreWalkError {
				return nil
			}
			return err
		}
		if path == root {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
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
		if !opt.IncludeNonRegular && !entry.Type().IsRegular() {
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

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("filepath.Rel: %w", err)
		}

		files = append(files, filepath.ToSlash(rel))

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.WalkDir: %w", err)
	}
	return files, nil
}
