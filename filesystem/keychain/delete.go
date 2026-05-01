package keychain

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pardnchiu/go-pkg/filesystem"
)

func Delete(key string) error {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("security", "delete-generic-password",
			"-s", service,
			"-a", key).Run()
	default:
		exec.Command("secret-tool", "clear",
			"service", service, "account", key).Run()
		deleteFallback(key)
	}
	return nil
}

func deleteFallback(key string) {
	path := filepath.Join(fallbackPath, ".secrets")
	lines := readFallbackLines()
	prefix := key + "="
	filtered := lines[:0]
	for _, l := range lines {
		if !strings.HasPrefix(l, prefix) {
			filtered = append(filtered, l)
		}
	}
	data := strings.Join(filtered, "\n")
	if len(filtered) > 0 {
		data += "\n"
	}
	filesystem.WriteFile(path, data, 0600)
}
