package parser

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var pptxRegex = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

func PPTX(ctx context.Context, path string) (string, []Chunk, error) {
	if path == "" {
		return "", nil, fmt.Errorf("pptx: path is required")
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", nil, fmt.Errorf("pptx: zip.OpenReader: %w", err)
	}
	defer zr.Close()

	type slideEntry struct {
		num int
		zf  *zip.File
	}
	var slides []slideEntry
	for _, f := range zr.File {
		m := pptxRegex.FindStringSubmatch(f.Name)
		if m == nil {
			continue
		}
		n, perr := strconv.Atoi(m[1])
		if perr != nil {
			continue
		}
		slides = append(slides, slideEntry{num: n, zf: f})
	}
	if len(slides) == 0 {
		return "", nil, fmt.Errorf("pptx %q: %w", path, ErrEmpty)
	}
	slices.SortFunc(slides, func(a, b slideEntry) int { return a.num - b.num })

	slideTexts := make([]string, 0, len(slides))
	var full strings.Builder
	for i, s := range slides {
		if err := ctx.Err(); err != nil {
			return full.String(), nil, err
		}
		rc, oerr := s.zf.Open()
		if oerr != nil {
			return full.String(), nil, fmt.Errorf("pptx: slide %d open: %w", s.num, oerr)
		}
		text, terr := pptxSlideText(ctx, rc)
		rc.Close()
		if terr != nil {
			return full.String(), nil, fmt.Errorf("pptx: slide %d parse: %w", s.num, terr)
		}
		if i > 0 {
			full.WriteString("\n\n")
		}
		full.WriteString(text)
		slideTexts = append(slideTexts, text)
	}

	fullText := full.String()
	docs, err := makeChunks(ctx, path, slideTexts)
	if err != nil {
		return fullText, docs, err
	}
	if len(docs) == 0 {
		return fullText, nil, fmt.Errorf("pptx %q: %w", path, ErrEmpty)
	}
	return fullText, docs, nil
}

func pptxSlideText(ctx context.Context, r io.Reader) (string, error) {
	dec := xml.NewDecoder(r)
	var b strings.Builder
	for {
		if err := ctx.Err(); err != nil {
			return b.String(), err
		}
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return b.String(), err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				var content string
				if err := dec.DecodeElement(&content, &t); err != nil {
					return b.String(), err
				}
				b.WriteString(content)
			case "br":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nil
}
