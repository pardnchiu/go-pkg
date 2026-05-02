package reader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pardnchiu/go-pkg/filesystem"
)

func ListFiles(dir string, opts ...ListOption) ([]File, error) {
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

	files := make([]File, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !opt.IncludeNonRegular && !e.Type().IsRegular() {
			continue
		}
		if opt.SkipExcluded && filesystem.IsExcluded(absDir, filepath.Join(absDir, e.Name())) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, newFile(filepath.Join(dir, e.Name()), info))
	}
	return files, nil
}
