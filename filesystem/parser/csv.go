package parser

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const csvMaxReadSize = 1 << 20

func CSV(ctx context.Context, path string, offset, limit int) (string, error) {
	if path == "" {
		return "", fmt.Errorf("csv: path is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("csv: os.Stat: %w", err)
	}
	if info.Size() > csvMaxReadSize {
		return "", fmt.Errorf("csv: file too large (%d bytes, max 1 MB)", info.Size())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("csv: os.ReadFile: %w", err)
	}
	raw := bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})

	reader := csv.NewReader(bytes.NewReader(raw))
	if strings.ToLower(filepath.Ext(path)) == ".tsv" {
		reader.Comma = '\t'
	}
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Sprintf("%s is empty", path), nil
		}
		return "", fmt.Errorf("csv: read header: %w", err)
	}

	skip := max(offset-1, 0)

	rows := make([][]string, 0, limit+1)
	rows = append(rows, header)
	dataCount := 0
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("csv: read row %d: %w", dataCount+1, err)
		}
		dataCount++
		if dataCount <= skip {
			continue
		}
		if len(rows)-1 >= limit {
			break
		}
		rows = append(rows, csvNormalizeRow(record, len(header)))
	}

	if dataCount == 0 {
		return csvMarshalRows(rows)
	}
	if len(rows) == 1 {
		return fmt.Sprintf("offset %d exceeds data rows %d in %s", offset, dataCount, path), nil
	}

	return csvMarshalRows(rows)
}

func csvNormalizeRow(row []string, cols int) []string {
	if len(row) == cols {
		return row
	}
	if len(row) > cols {
		return row[:cols]
	}
	out := make([]string, cols)
	copy(out, row)
	return out
}

func csvMarshalRows(rows [][]string) (string, error) {
	b, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("csv: json.Marshal: %w", err)
	}
	return string(b), nil
}
