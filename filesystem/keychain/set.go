package keychain

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pardnchiu/go-utils/filesystem"
)

func Set(service, fallbackPath, key, value string) error {
	if value == "" {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		return setSecretOnMac(service, key, value)
	default:
		if ok := setSecret(service, key, value); ok == nil {
			return nil
		}
		return setFallback(fallbackPath, key, value)
	}
}

func setSecretOnMac(service, key, value string) error {
	exec.Command("security", "delete-generic-password",
		"-s", service,
		"-a", key).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", key,
		"-w", value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("security add-generic-password: %s", out)
	}
	return nil
}

func setSecret(service, key, value string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", service+"/"+key,
		"service", service, "account", key)
	cmd.Stdin = strings.NewReader(value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secret-tool store: %s", out)
	}
	return nil
}

func setFallback(fallbackPath, key, value string) error {
	path := filepath.Join(fallbackPath, ".secrets")
	lines := readFallbackLines(fallbackPath)
	prefix := key + "="
	found := false
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) {
			lines[i] = prefix + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, prefix+value)
	}
	data := strings.Join(lines, "\n") + "\n"
	return filesystem.WriteFile(path, data, 0600)
}
