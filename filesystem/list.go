package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ListOption struct {
	SkipExcluded bool
}

func getListOption(opts []ListOption) ListOption {
	if len(opts) == 0 {
		return ListOption{}
	}
	return opts[len(opts)-1]
}

func ListFiles(dir string, opts ...ListOption) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	opt := getListOption(opts)
	var absDir string
	if opt.SkipExcluded {
		absDir, err = filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("filepath.Abs: %w", err)
		}
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		if opt.SkipExcluded && IsExcluded(absDir, filepath.Join(absDir, e.Name())) {
			continue
		}
		files = append(files, e.Name())
	}
	return files, nil
}

func ListDirs(dir string, opts ...ListOption) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	opt := getListOption(opts)
	var absDir string
	if opt.SkipExcluded {
		absDir, err = filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("filepath.Abs: %w", err)
		}
	}

	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if opt.SkipExcluded && IsExcluded(absDir, filepath.Join(absDir, e.Name())) {
			continue
		}
		dirs = append(dirs, e.Name())
	}
	return dirs, nil
}

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
				if IsExcluded(absRoot, abs) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if opt.SkipExcluded {
			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("filepath.Abs: %w", err)
			}
			if IsExcluded(absRoot, abs) {
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
