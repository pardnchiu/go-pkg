package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pardnchiu/go-pkg/utils"
)

func validateDir(path string) (home, workDir string, err error) {
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("os.UserHomeDir: %w", err)
	}

	workDir, err = filepath.EvalSymlinks(utils.AbsPath("", path))
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

func resolveDir(path string) string {
	return utils.AbsPath("", path)
}

func resolveBinds(list []string) []string {
	out := make([]string, 0, len(list))
	for _, one := range list {
		one = strings.TrimSpace(one)
		if one == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(one); err == nil {
			out = append(out, resolved)
			continue
		}
		abs := utils.AbsPath("", one)
		if abs == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		out = append(out, abs)
	}
	return out
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
		if one := resolveDenied(homeDir, d); one != "" {
			dirs = append(dirs, one)
		}
	}
	for _, f := range cachedDenied.Files {
		if one := resolveDenied(homeDir, f); one != "" {
			files = append(files, one)
		}
	}
	return
}

func resolveDenied(homeDir, entry string) string {
	if strings.TrimSpace(entry) == "" {
		return ""
	}
	return utils.AbsPath(homeDir, entry)
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
