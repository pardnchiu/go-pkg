package filesystem

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func Copy(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("os.Stat: %w", err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("src %q is not a regular file", src)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("Close: %w", err)
	}

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("os.Rename: %w", err)
	}
	return nil
}

func Move(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return fmt.Errorf("os.Rename: %w", err)
	}

	// * cross-device: copy then remove src
	if err := Copy(src, dst); err != nil {
		return fmt.Errorf("Copy: %w", err)
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("os.Remove: %w", err)
	}
	return nil
}

func Remove(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("os.Remove: %w", err)
}
