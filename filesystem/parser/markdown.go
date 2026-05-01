package parser

import (
	"context"
	"fmt"
	"os"
)

func Markdown(ctx context.Context, path string) (string, []Chunk, error) {
	if path == "" {
		return "", nil, fmt.Errorf("markdown: path is required")
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("markdown: os.ReadFile: %w", err)
	}

	text := string(data)
	docs, err := splitParagraphs(ctx, path, text)
	if err != nil {
		return text, docs, err
	}
	if len(docs) == 0 {
		return text, nil, fmt.Errorf("markdown %q: %w", path, ErrEmpty)
	}
	return text, docs, nil
}
