package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type serviceClient struct {
	httpClient   *http.Client
	baseURL      string
	apiKey       string
	apiKeyHeader string
}

type apiResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

const serviceUserIDHeader = "X-User-ID"

type serviceRequest struct {
	method  string
	path    string
	body    any
	headers map[string]string
	query   url.Values
}

func executeServiceRequest[T any](ctx context.Context, client *serviceClient, req serviceRequest) (T, error) {
	var zero T

	if client == nil || client.httpClient == nil {
		return zero, errors.New("service client is not configured")
	}

	var bodyReader io.Reader
	if req.body != nil {
		bodyBytes, err := json.Marshal(req.body)
		if err != nil {
			return zero, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, client.baseURL+req.path, bodyReader)
	if err != nil {
		return zero, err
	}
	if req.query != nil {
		httpReq.URL.RawQuery = req.query.Encode()
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(client.apiKeyHeader, client.apiKey)
	if req.body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	for key, value := range req.headers {
		httpReq.Header.Set(key, value)
	}

	res, err := client.httpClient.Do(httpReq)
	if err != nil {
		return zero, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return zero, err
	}

	bodyText := strings.TrimSpace(string(bodyBytes))
	if res.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("%s %s: %s: %s", req.method, httpReq.URL.RequestURI(), res.Status, bodyText)
	}
	if bodyText == "" {
		return zero, nil
	}

	var payload apiResponse[T]
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return zero, err
	}
	if !payload.Success {
		if payload.Error == "" {
			payload.Error = "upstream request was unsuccessful"
		}
		return zero, errors.New(payload.Error)
	}

	return payload.Data, nil
}
