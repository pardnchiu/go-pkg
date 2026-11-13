package filesystem

import (
	_ "embed"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
)

type SensitiveDic struct {
	Dirs       []string `json:"dirs"`
	Files      []string `json:"files"`
	Prefixes   []string `json:"prefixes"`
	Extensions []string `json:"extensions"`
}

//go:embed jsons/sensitive_path.json
var sensitiveDicRaw []byte

var sensitiveDic = func() SensitiveDic {
	var dic SensitiveDic
	if err := json.Unmarshal(sensitiveDicRaw, &dic); err != nil {
		panic("[Internal] jsons/sensitive_path.json: " + err.Error())
	}
	return dic
}()

func IsSensitive(path string, extra ...SensitiveDic) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return false
	}

	for _, list := range append([]SensitiveDic{sensitiveDic}, extra...) {
		if list.match(path) {
			return true
		}
	}
	return false
}

func (l SensitiveDic) match(path string) bool {
	for segment := range strings.SplitSeq(path, string(filepath.Separator)) {
		if slices.Contains(l.Dirs, segment) {
			return true
		}
	}

	base := filepath.Base(path)
	if slices.Contains(l.Files, base) {
		return true
	}
	if slices.ContainsFunc(l.Prefixes, func(p string) bool { return strings.HasPrefix(base, p) }) {
		return true
	}
	return slices.Contains(l.Extensions, strings.ToLower(filepath.Ext(base)))
}
