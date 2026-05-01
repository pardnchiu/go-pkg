package reader

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsEmpty(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("os.Stat: %w", err)
	}

	if !info.IsDir() {
		return info.Size() == 0, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("os.Open: %w", err)
	}
	defer file.Close()

	// * read at most one entry; EOF means empty
	_, err = file.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	if err != nil && !errors.Is(err, fs.ErrClosed) {
		return false, fmt.Errorf("file.Readdirnames: %w", err)
	}
	return false, nil
}
