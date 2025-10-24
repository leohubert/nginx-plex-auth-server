package plex

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/leohubert/nginx-plex-auth-server/pkg/errtb"
)

type Request struct {
	Method  string
	URL     string
	Body    any
	Headers map[string]string
}

func do[T any](client *http.Client, req *Request) (*T, error) {
	request, err := http.NewRequest(req.Method, req.URL, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		request.Header.Set(k, v)
	}

	if req.Body != nil {
		request.Header.Set("Content-Type", "application/json")
		jsonBody := errtb.Must(json.Marshal(req.Body))
		request.Body = io.NopCloser(bytes.NewReader(jsonBody))
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var result T

	switch response.Header.Get("Content-Type") {
	case "application/json", "application/json; charset=utf-8":
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode JSON response: %w", err)
		}
	case "text/xml; charset=utf-8", "application/xml; charset=utf-8":
		if err := xml.NewDecoder(response.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode XML response: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported content type: %s", response.Header.Get("Content-Type"))
	}

	return &result, nil
}
