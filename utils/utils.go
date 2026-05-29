package utils

import (
	"fmt"
	"strings"
)

func GetKeys[V any](obj map[string]V) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys
}

func TruncateString(text string, length int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	result := []rune(text)
	if len(result) > length {
		return string(result[:length]) + "…"
	}
	return string(result)
}

func CompactNumber(n int) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
