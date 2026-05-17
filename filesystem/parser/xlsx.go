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

var xlsxSheetRegex = regexp.MustCompile(`^xl/worksheets/sheet(\d+)\.xml$`)

type xlsxCell struct {
	Ref     string   `xml:"r,attr"`
	Type    string   `xml:"t,attr"`
	V       string   `xml:"v"`
	InlineT []string `xml:"is>t"`
	InlineR []struct {
		T string `xml:"t"`
	} `xml:"is>r"`
}

func XLSX(ctx context.Context, path string, offset, limit int) (string, error) {
	if path == "" {
		return "", fmt.Errorf("xlsx: path is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("xlsx: zip.OpenReader: %w", err)
	}
	defer zr.Close()

	var sstFile *zip.File
	type sheetEntry struct {
		num int
		zf  *zip.File
	}
	var sheets []sheetEntry
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			sstFile = f
			continue
		}
		m := xlsxSheetRegex.FindStringSubmatch(f.Name)
		if m == nil {
			continue
		}
		n, perr := strconv.Atoi(m[1])
		if perr != nil {
			continue
		}
		sheets = append(sheets, sheetEntry{num: n, zf: f})
	}
	if len(sheets) == 0 {
		return "", fmt.Errorf("xlsx: no worksheets found in %q", path)
	}
	slices.SortFunc(sheets, func(a, b sheetEntry) int { return a.num - b.num })

	var sst []string
	if sstFile != nil {
		rc, oerr := sstFile.Open()
		if oerr != nil {
			return "", fmt.Errorf("xlsx: sharedStrings open: %w", oerr)
		}
		sst, err = xlsxSharedStrings(ctx, rc)
		rc.Close()
		if err != nil {
			return "", fmt.Errorf("xlsx: sharedStrings parse: %w", err)
		}
	}

	rc, err := sheets[0].zf.Open()
	if err != nil {
		return "", fmt.Errorf("xlsx: sheet%d open: %w", sheets[0].num, err)
	}
	defer rc.Close()
	rows, err := xlsxSheetRows(ctx, rc, sst)
	if err != nil {
		return "", fmt.Errorf("xlsx: sheet%d parse: %w", sheets[0].num, err)
	}

	if len(rows) == 0 {
		return fmt.Sprintf("%s is empty", path), nil
	}

	header := rows[0]
	cols := len(header)
	dataRows := rows[1:]

	if len(dataRows) == 0 {
		return csvMarshalRows([][]string{header})
	}

	skip := max(offset-1, 0)
	if skip >= len(dataRows) {
		return fmt.Sprintf("offset %d exceeds data rows %d in %s", offset, len(dataRows), path), nil
	}

	end := min(skip+limit, len(dataRows))
	out := make([][]string, 0, 1+(end-skip))
	out = append(out, csvNormalizeRow(header, cols))
	for _, r := range dataRows[skip:end] {
		out = append(out, csvNormalizeRow(r, cols))
	}
	return csvMarshalRows(out)
}

func xlsxSharedStrings(ctx context.Context, r io.Reader) ([]string, error) {
	dec := xml.NewDecoder(r)
	var sst []string
	var current strings.Builder
	inSI := false
	for {
		if err := ctx.Err(); err != nil {
			return sst, err
		}
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return sst, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				current.Reset()
				inSI = true
			case "t":
				if inSI {
					var content string
					if derr := dec.DecodeElement(&content, &t); derr != nil {
						return sst, derr
					}
					current.WriteString(content)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "si" {
				sst = append(sst, current.String())
				inSI = false
			}
		}
	}
	return sst, nil
}

func xlsxSheetRows(ctx context.Context, r io.Reader, sst []string) ([][]string, error) {
	dec := xml.NewDecoder(r)
	var rows [][]string
	var currentRow []string
	for {
		if err := ctx.Err(); err != nil {
			return rows, err
		}
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return rows, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				currentRow = currentRow[:0]
			case "c":
				var cell xlsxCell
				if derr := dec.DecodeElement(&cell, &t); derr != nil {
					return rows, derr
				}
				col := parseXLSXColRef(cell.Ref)
				if col <= 0 {
					continue
				}
				for len(currentRow) < col-1 {
					currentRow = append(currentRow, "")
				}
				currentRow = append(currentRow, xlsxCellValue(cell, sst))
			}
		case xml.EndElement:
			if t.Name.Local == "row" {
				row := make([]string, len(currentRow))
				copy(row, currentRow)
				rows = append(rows, row)
				currentRow = currentRow[:0]
			}
		}
	}
	return rows, nil
}

func parseXLSXColRef(ref string) int {
	col := 0
	for _, ch := range ref {
		switch {
		case ch >= 'A' && ch <= 'Z':
			col = col*26 + int(ch-'A'+1)
		case ch >= 'a' && ch <= 'z':
			col = col*26 + int(ch-'a'+1)
		default:
			return col
		}
	}
	return col
}

func xlsxCellValue(c xlsxCell, sst []string) string {
	switch c.Type {
	case "s":
		idx, err := strconv.Atoi(c.V)
		if err != nil || idx < 0 || idx >= len(sst) {
			return ""
		}
		return sst[idx]
	case "inlineStr":
		var b strings.Builder
		for _, t := range c.InlineT {
			b.WriteString(t)
		}
		for _, r := range c.InlineR {
			b.WriteString(r.T)
		}
		return b.String()
	case "b":
		if c.V == "1" {
			return "TRUE"
		}
		return "FALSE"
	default:
		return c.V
	}
}
