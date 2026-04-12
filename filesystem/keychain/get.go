package keychain

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Get(service, fallbackPath, key string) string {
	if val := readKeychain(service, fallbackPath, key); val != "" {
		return val
	}
	return os.Getenv(key)
}

func readKeychain(service, fallbackPath, key string) string {
	switch runtime.GOOS {
	case "darwin":
		return getSecretFromMac(service, key)
	default:
		if secret := getSecret(service, key); secret != "" {
			return secret
		}
		return getFallback(fallbackPath, key)
	}
}

func getSecretFromMac(service, key string) string {
	out, err := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", key,
		"-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getSecret(service, key string) string {
	out, err := exec.Command("secret-tool", "lookup",
		"service", service, "account", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getFallback(fallbackPath, key string) string {
	prefix := key + "="
	for _, l := range readFallbackLines(fallbackPath) {
		if v, ok := strings.CutPrefix(l, prefix); ok {
			return v
		}
	}
	return ""
}

func readFallbackLines(fallbackPath string) []string {
	path := filepath.Join(fallbackPath, ".secrets")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
