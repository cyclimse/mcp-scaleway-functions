package cockpit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type LokiClient interface {
	Query(ctx context.Context, query string, start time.Time, end time.Time) ([]Log, error)
}

type lokiClient struct {
	httpClient http.Client
	url        string
}

func NewLokiClient(url string, secretKey string) LokiClient {
	return &lokiClient{
		httpClient: http.Client{
			Transport: &roundTripper{
				base:      http.DefaultTransport,
				secretKey: secretKey,
			},
		},
		url: url,
	}
}

func (c *lokiClient) Query(ctx context.Context, query string, start time.Time, end time.Time) ([]Log, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/loki/api/v1/query_range", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", fmt.Sprintf("%d", start.UnixNano()))
	q.Add("end", fmt.Sprintf("%d", end.UnixNano()))

	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code (%d) from Loki", resp.StatusCode)
	}

	var queryResp QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if queryResp.Status != "success" {
		return nil, fmt.Errorf("query failed with status: %s", queryResp.Status)
	}

	var logs []Log
	for _, stream := range queryResp.Data.Result {
		for _, entry := range stream.Entries {
			logs = append(logs, Log{
				Timestamp: entry.Timestamp,
				Message:   entry.Line,
			})
		}
	}

	return logs, nil
}

type QueryResponse struct {
	Status string            `json:"status"`
	Data   QueryResponseData `json:"data"`
}

type QueryResponseData struct {
	ResultType string `json:"resultType"`
	// We only support stream result type for now.
	Result Streams `json:"result"`
}

type Streams []Stream

type Stream struct {
	Stream  StreamMetadata `json:"stream"`
	Entries []Entry        `json:"values"`
}

type StreamMetadata struct {
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	ResourceType string `json:"resource_type"`
}

type roundTripper struct {
	base      http.RoundTripper
	secretKey string
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Token", rt.secretKey)
	return rt.base.RoundTrip(req)
}
