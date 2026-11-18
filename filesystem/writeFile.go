package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-pkg/utils"
)

func WriteFile(path, content string, permission os.FileMode) error {
	_, err := WriteStream(path, strings.NewReader(content), permission)
	return err
}

func WriteStream(path string, r io.Reader, permission os.FileMode) (int64, error) {
	path = utils.AbsPath("", path)

	if IsDenied(path) {
		return 0, fmt.Errorf("access denied: %s", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("os.MkdirAll: %w", err)
	}

	// * ensure atomic write:
	// * pre-save data as temp
	tmpFile, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return 0, fmt.Errorf("os.CreateTemp: %w", err)
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)

	if err := tmpFile.Chmod(permission); err != nil {
		tmpFile.Close()
		return 0, fmt.Errorf("os.Chmod: %w", err)
	}
	size, err := io.Copy(tmpFile, r)
	if err != nil {
		tmpFile.Close()
		return 0, fmt.Errorf("io.Copy: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return 0, fmt.Errorf("os.Close: %w", err)
	}

	// * rename temp to target
	if err := os.Rename(tmp, path); err != nil {
		return 0, fmt.Errorf("os.Rename: %w", err)
	}
	return size, nil
}
