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
	"strings"
)

var docxPartPattern = regexp.MustCompile(`^word/(?:document|header\d+|footer\d+|footnotes|endnotes)\.xml$`)

func docxPartOrder(name string) int {
	switch {
	case name == "word/document.xml":
		return 0
	case strings.HasPrefix(name, "word/header"):
		return 1
	case strings.HasPrefix(name, "word/footer"):
		return 2
	case name == "word/footnotes.xml":
		return 3
	case name == "word/endnotes.xml":
		return 4
	}
	return 99
}

func Docx(ctx context.Context, path string) (string, []Chunk, error) {
	if path == "" {
		return "", nil, fmt.Errorf("docx: path is required")
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", nil, fmt.Errorf("docx: zip.OpenReader: %w", err)
	}
	defer zr.Close()

	type part struct {
		name string
		zf   *zip.File
	}
	var parts []part
	hasMain := false
	for _, f := range zr.File {
		if !docxPartPattern.MatchString(f.Name) {
			continue
		}
		parts = append(parts, part{name: f.Name, zf: f})
		if f.Name == "word/document.xml" {
			hasMain = true
		}
	}
	if !hasMain {
		return "", nil, fmt.Errorf("docx: word/document.xml missing")
	}
	slices.SortFunc(parts, func(a, b part) int {
		if d := docxPartOrder(a.name) - docxPartOrder(b.name); d != 0 {
			return d
		}
		return strings.Compare(a.name, b.name)
	})

	var full strings.Builder
	for _, p := range parts {
		if err := ctx.Err(); err != nil {
			return full.String(), nil, err
		}
		rc, oerr := p.zf.Open()
		if oerr != nil {
			return full.String(), nil, fmt.Errorf("docx: %s open: %w", p.name, oerr)
		}
		text, terr := docxText(ctx, rc)
		rc.Close()
		if terr != nil {
			return full.String(), nil, fmt.Errorf("docx: %s parse: %w", p.name, terr)
		}
		if full.Len() > 0 && text != "" {
			full.WriteString("\n\n")
		}
		full.WriteString(text)
	}

	text := full.String()
	docs, err := splitParagraphs(ctx, path, text)
	if err != nil {
		return text, docs, err
	}
	if len(docs) == 0 {
		return text, nil, fmt.Errorf("docx %q: %w", path, ErrEmpty)
	}
	return text, docs, nil
}

func docxText(ctx context.Context, r io.Reader) (string, error) {
	dec := xml.NewDecoder(r)
	var b strings.Builder
	tableDepth := 0
	pendingNewline := false
	pendingTab := false

	flush := func() {
		if pendingNewline {
			b.WriteByte('\n')
			pendingNewline = false
		}
		if pendingTab {
			b.WriteByte('\t')
			pendingTab = false
		}
	}

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
			case "tbl":
				tableDepth++
			case "t":
				var content string
				if err := dec.DecodeElement(&content, &t); err != nil {
					return b.String(), err
				}
				flush()
				b.WriteString(content)
			case "tab":
				flush()
				b.WriteByte('\t')
			case "br":
				flush()
				b.WriteByte('\n')
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if tableDepth > 0 {
					pendingNewline = true
				} else {
					b.WriteString("\n\n")
				}
			case "tc":
				if tableDepth > 0 {
					pendingNewline = false
					pendingTab = true
				}
			case "tr":
				if tableDepth > 0 {
					pendingTab = false
					b.WriteByte('\n')
				}
			case "tbl":
				if tableDepth > 0 {
					tableDepth--
				}
				if tableDepth == 0 {
					pendingNewline = false
					pendingTab = false
					b.WriteString("\n\n")
				}
			}
		}
	}
	return b.String(), nil
}
