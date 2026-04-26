package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func CheckDir(path string, create bool) error {
	if path == "" {
		return errors.New("path is empty")
	}

	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path %q exists but is not a directory", path)
		}
		return nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("os.Stat: %w", err)
	}

	if !create {
		return fmt.Errorf("directory %q does not exist", path)
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}
	return nil
}
