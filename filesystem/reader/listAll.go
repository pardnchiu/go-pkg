package reader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pardnchiu/go-utils/filesystem"
)

func ListAll(dir string, opts ...ListOption) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	opt := getListOption(opts)
	if !opt.SkipExcluded {
		return entries, nil
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("filepath.Abs: %w", err)
	}

	out := make([]os.DirEntry, 0, len(entries))
	for _, e := range entries {
		if filesystem.IsExcluded(absDir, filepath.Join(absDir, e.Name())) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
