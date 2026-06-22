package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func POST[T any](ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (T, int, error) {
	return send[T](ctx, client, "POST", api, header, body, contentType)
}

func PUT[T any](ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (T, int, error) {
	return send[T](ctx, client, "PUT", api, header, body, contentType)
}

func PATCH[T any](ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (T, int, error) {
	return send[T](ctx, client, "PATCH", api, header, body, contentType)
}

func DELETE[T any](ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (T, int, error) {
	return send[T](ctx, client, "DELETE", api, header, body, contentType)
}

func send[T any](ctx context.Context, client *http.Client, method, api string, header map[string]string, body map[string]any, contentType string) (T, int, error) {
	var result T

	if contentType == "" {
		contentType = "json"
	}

	var req *http.Request
	var err error
	if contentType == "form" {
		requestBody := url.Values{}
		for k, v := range body {
			requestBody.Set(k, fmt.Sprint(v))
		}

		req, err = http.NewRequestWithContext(ctx, method, api, strings.NewReader(requestBody.Encode()))
		if err != nil {
			return result, 0, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		requestBody, err := json.Marshal(body)
		if err != nil {
			return result, 0, fmt.Errorf("failed to marshal body: %w", err)
		}

		req, err = http.NewRequestWithContext(ctx, method, api, strings.NewReader(string(requestBody)))
		if err != nil {
			return result, 0, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")

	for k, v := range header {
		req.Header.Set(k, v)
	}

	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, 0, fmt.Errorf("failed to send: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if statusCode < 200 || statusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return result, statusCode, fmt.Errorf("HTTP %d: %s", statusCode, strings.TrimSpace(string(raw)))
	}

	if s, ok := any(&result).(*string); ok {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, statusCode, fmt.Errorf("failed to read: %w", err)
		}
		*s = string(b)
		return result, statusCode, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, statusCode, fmt.Errorf("failed to read: %w", err)
	}
	return result, statusCode, nil
}
