package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReadText(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("os.ReadFile: %w", err)
	}
	return string(bytes), nil
}

func WriteText(path, content string) error {
	return WriteFile(path, content, 0644)
}

func AppendText(path, content string) error {
	if IsDenied(path) {
		return fmt.Errorf("access denied: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("file.WriteString: %w", err)
	}
	return nil
}
