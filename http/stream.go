package http

import (
	"context"
	"fmt"
	"net/http"
)

func GETStream(ctx context.Context, client *http.Client, api string, header map[string]string) (*http.Response, error) {
	if client == nil {
		client = &http.Client{}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", api, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}

	return client.Do(req)
}

func POSTStream(ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (*http.Response, error) {
	return sendStream(ctx, client, "POST", api, header, body, contentType)
}

func PUTStream(ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (*http.Response, error) {
	return sendStream(ctx, client, "PUT", api, header, body, contentType)
}

func PATCHStream(ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (*http.Response, error) {
	return sendStream(ctx, client, "PATCH", api, header, body, contentType)
}

func DELETEStream(ctx context.Context, client *http.Client, api string, header map[string]string, body map[string]any, contentType string) (*http.Response, error) {
	return sendStream(ctx, client, "DELETE", api, header, body, contentType)
}

func sendStream(ctx context.Context, client *http.Client, method, api string, header map[string]string, body map[string]any, contentType string) (*http.Response, error) {
	if contentType == "" {
		contentType = "json"
	}

	requestBody, requestType, err := buildBody(body, contentType)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, api, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", requestType)

	for k, v := range header {
		req.Header.Set(k, v)
	}

	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send: %w", err)
	}
	return resp, nil
}
