package filesystem

import (
	_ "embed"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

//go:embed jsons/readable_path.json
var readablePathRaw []byte

var ReadablePaths = func() []string {
	var list []string
	if err := json.Unmarshal(readablePathRaw, &list); err != nil {
		panic("[Internal] jsons/readable_path.json: " + err.Error())
	}
	return list
}()

var resolvedReadablePaths = sync.OnceValue(func() []string {
	list := make([]string, 0, len(ReadablePaths)*2)
	for _, path := range ReadablePaths {
		list = append(list, path)
		if resolved, err := filepath.EvalSymlinks(path); err == nil && resolved != path && resolved != string(filepath.Separator) {
			list = append(list, resolved)
		}
	}
	return list
})

func IsReadable(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	realPath, err := RealPath(path)
	if err != nil {
		return false
	}

	realPath = filepath.Clean(realPath)
	if IsDenied(realPath) {
		return false
	}

	return slices.ContainsFunc(resolvedReadablePaths(), func(rootPath string) bool {
		return realPath == rootPath || strings.HasPrefix(realPath, rootPath+string(filepath.Separator))
	})
}
