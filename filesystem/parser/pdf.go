package parser

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func PDF(ctx context.Context, path string) (string, []Chunk, error) {
	if path == "" {
		return "", nil, fmt.Errorf("pdf: path is required")
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", path, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("pdf: pdftotext: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	text := stdout.String()
	pages := strings.Split(text, "\f")
	if n := len(pages); n > 0 && pages[n-1] == "" {
		pages = pages[:n-1]
	}

	docs, err := makeChunks(ctx, path, pages)
	if err != nil {
		return text, docs, err
	}
	if len(docs) == 0 {
		return text, nil, fmt.Errorf("pdf %q: %w", path, ErrEmpty)
	}
	return text, docs, nil
}
