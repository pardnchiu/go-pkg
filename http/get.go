package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

func GET[T any](ctx context.Context, client *http.Client, api string, header map[string]string) (T, int, error) {
	var result T

	if client == nil {
		client = &http.Client{}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", api, nil)
	if err != nil {
		return result, 0, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if s, ok := any(&result).(*string); ok {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, statusCode, err
		}
		*s = string(b)
		return result, statusCode, nil
	}

	ct := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "xml"):
		err = xml.NewDecoder(resp.Body).Decode(&result)
	default:
		err = json.NewDecoder(resp.Body).Decode(&result)
	}
	if err != nil {
		return result, statusCode, err
	}
	return result, statusCode, nil
}
