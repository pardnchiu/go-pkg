package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"strings"
)

type File struct {
	Name        string
	ContentType string
	Data        []byte
	Reader      io.Reader
}

func buildBody(body map[string]any, contentType string) (io.Reader, string, error) {
	switch contentType {
	case "form":
		values := url.Values{}
		for k, v := range body {
			values.Set(k, fmt.Sprint(v))
		}
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil
	case "multipart":
		return buildMultipart(body)
	default:
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal body: %w", err)
		}
		return bytes.NewReader(raw), "application/json", nil
	}
}

func buildMultipart(body map[string]any) (io.Reader, string, error) {
	var buffer bytes.Buffer
	form := multipart.NewWriter(&buffer)

	for k, v := range body {
		var file File
		switch value := v.(type) {
		case File:
			file = value
		case *File:
			if value == nil {
				return nil, "", fmt.Errorf("field %q: File is nil", k)
			}
			file = *value
		case []byte:
			file = File{Data: value}
		case io.Reader:
			file = File{Reader: value}
		default:
			if err := form.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", fmt.Errorf("failed to write field %q: %w", k, err)
			}
			continue
		}

		if err := writeFile(form, k, file); err != nil {
			return nil, "", err
		}
	}

	if err := form.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}
	return &buffer, form.FormDataContentType(), nil
}

func writeFile(form *multipart.Writer, field string, file File) error {
	name := file.Name
	if name == "" {
		name = field
	}

	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(field), escapeQuotes(name)))
	if file.ContentType != "" {
		header.Set("Content-Type", file.ContentType)
	} else {
		header.Set("Content-Type", "application/octet-stream")
	}

	part, err := form.CreatePart(header)
	if err != nil {
		return fmt.Errorf("failed to create part %q: %w", field, err)
	}

	if file.Reader != nil {
		if _, err := io.Copy(part, file.Reader); err != nil {
			return fmt.Errorf("failed to copy part %q: %w", field, err)
		}
		return nil
	}
	if _, err := part.Write(file.Data); err != nil {
		return fmt.Errorf("failed to write part %q: %w", field, err)
	}
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
