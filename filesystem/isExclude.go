package filesystem

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type exclude struct {
	file   string
	negate bool
}

var invalidNegateRegex = regexp.MustCompile(`^!{2,}`)

func IsExcluded(workDir, absPath string) bool {
	rel, err := filepath.Rel(workDir, absPath)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}

	excluded := false
	for _, ex := range loadExcludes(workDir) {
		match, err := filepath.Match(ex.file, filepath.Base(rel))
		if err != nil {
			continue
		}
		if !match {
			match = strings.HasPrefix(rel, ex.file+"/") ||
				strings.Contains(rel, "/"+ex.file+"/")
		}
		if match {
			excluded = !ex.negate
		}
	}
	return excluded
}

func loadExcludes(dir string) []exclude {
	seen := make(map[exclude]struct{})
	var out []exclude
	add := func(ex exclude) {
		if _, ok := seen[ex]; ok {
			return
		}
		seen[ex] = struct{}{}
		out = append(out, ex)
	}

	for _, raw := range excludeList {
		if ex, ok := parseExcludeLine(raw); ok {
			add(ex)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() ||
			!strings.HasSuffix(name, "ignore") ||
			!strings.HasPrefix(name, ".") {
			continue
		}
		for _, ex := range parseIgnoreFile(filepath.Join(dir, name)) {
			add(ex)
		}
	}
	return out
}

func parseExcludeLine(raw string) (exclude, bool) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return exclude{}, false
	}
	if invalidNegateRegex.MatchString(line) {
		return exclude{}, false
	}

	negate := false
	if strings.HasPrefix(line, "!") {
		negate = true
		line = strings.TrimPrefix(line, "!")
		if line == "" {
			return exclude{}, false
		}
	}
	line = strings.TrimPrefix(line, "/")
	line = strings.TrimSuffix(line, "/")
	if line == "" {
		return exclude{}, false
	}
	return exclude{file: line, negate: negate}, true
}

func parseIgnoreFile(path string) []exclude {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var out []exclude
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if ex, ok := parseExcludeLine(scanner.Text()); ok {
			out = append(out, ex)
		}
	}
	return out
}
