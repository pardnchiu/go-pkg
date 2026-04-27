package filesystem

import (
	"path/filepath"
	"strings"
)

func IsDenied(path string) bool {
	if len(deniedMap.Dirs) == 0 && len(deniedMap.Files) == 0 &&
		len(deniedMap.Prefixes) == 0 && len(deniedMap.Extensions) == 0 {
		return false
	}

	resolved, err := RealPath(path)
	if err != nil {
		return true
	}
	clean := filepath.Clean(resolved)
	base := filepath.Base(clean)

	for _, dir := range deniedMap.Dirs {
		if strings.Contains(clean, "/"+dir+"/") || strings.Contains(clean, "/"+dir) {
			return true
		}
	}
	for _, f := range deniedMap.Files {
		if strings.Contains(clean, f) {
			return true
		}
	}
	for _, prefix := range deniedMap.Prefixes {
		// .example variants are exempt (e.g. .env.example allowed, .env.prod denied)
		if strings.HasPrefix(base, prefix) && !strings.Contains(base, ".example") {
			return true
		}
	}
	for _, ext := range deniedMap.Extensions {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}
	return false
}
