package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func validateDir(path string) (home, workDir string, err error) {
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("os.UserHomeDir: %w", err)
	}

	workDir, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", fmt.Errorf("filepath.EvalSymlinks: %w", err)
	}

	if !strings.HasPrefix(workDir, home+"/") && workDir != home {
		return "", "", fmt.Errorf("just allow paths under home: %s", workDir)
	}

	return home, workDir, nil
}

type deniedMap struct {
	Dirs  []string `json:"dirs"`
	Files []string `json:"files"`
}

type NetworkMode int

const (
	NetworkAllow NetworkMode = iota
	NetworkDeny
)

type WriteScope int

const (
	WriteWork WriteScope = iota
	WriteHome
)

type BindSpec struct {
	WriteScope WriteScope
	ReadOnly   []string
	ReadWrite  []string
}

type Option struct {
	AllowAll     bool
	CPUPercent   int
	MemoryMB     int
	Network      NetworkMode
	DropCaps     bool
	MinimalBinds *BindSpec
}

func resolveDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("filepath.Abs: %w", err)
	}
	return abs, nil
}

var minimalRoBinds = []string{
	"/usr",
	"/etc",
	"/bin",
	"/sbin",
	"/lib",
	"/lib32",
	"/lib64",
	"/opt",
}

var (
	cachedDenied deniedMap
	newOnce      sync.Once
)

func New(data []byte) {
	newOnce.Do(func() {
		_ = json.Unmarshal(data, &cachedDenied)
	})
}

func deniedPaths(homeDir string) (dirs []string, files []string) {
	for _, d := range cachedDenied.Dirs {
		dirs = append(dirs, filepath.Join(homeDir, d))
	}
	for _, f := range cachedDenied.Files {
		files = append(files, filepath.Join(homeDir, f))
	}
	return
}

func ParseMemory(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory string")
	}

	lower := strings.ToLower(s)
	mult := 1
	num := lower
	switch {
	case strings.HasSuffix(lower, "gib"):
		mult = 1024
		num = lower[:len(lower)-3]
	case strings.HasSuffix(lower, "mib"):
		num = lower[:len(lower)-3]
	case strings.HasSuffix(lower, "gb"), strings.HasSuffix(lower, "gi"):
		mult = 1024
		num = lower[:len(lower)-2]
	case strings.HasSuffix(lower, "mb"), strings.HasSuffix(lower, "mi"):
		num = lower[:len(lower)-2]
	case strings.HasSuffix(lower, "g"):
		mult = 1024
		num = lower[:len(lower)-1]
	case strings.HasSuffix(lower, "m"):
		num = lower[:len(lower)-1]
	}

	n, err := strconv.Atoi(strings.TrimSpace(num))
	if err != nil {
		return 0, fmt.Errorf("unsupported memory format %q", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("negative memory: %q", s)
	}
	return n * mult, nil
}
