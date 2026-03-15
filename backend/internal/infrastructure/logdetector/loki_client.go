package logdetector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type QueryMatch struct {
	Timestamp time.Time
	Line      string
	Labels    map[string]string
}

type QueryResult struct {
	Matches []QueryMatch
}

type LokiClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewLokiClient(baseURL string, httpClient *http.Client) *LokiClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &LokiClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *LokiClient) QueryRange(ctx context.Context, query string, start time.Time, end time.Time, limit int) (*QueryResult, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("loki base URL is required")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("loki query is required")
	}
	if limit <= 0 {
		limit = 20
	}

	endpoint, err := url.Parse(c.baseURL + "/loki/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("parse loki base URL: %w", err)
	}
	values := endpoint.Query()
	values.Set("query", query)
	values.Set("start", strconv.FormatInt(start.UTC().UnixNano(), 10))
	values.Set("end", strconv.FormatInt(end.UTC().UnixNano(), 10))
	values.Set("limit", strconv.Itoa(limit))
	values.Set("direction", "backward")
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build loki request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute loki query: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("loki query failed with status %d", resp.StatusCode)
	}

	var decoded lokiQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode loki query response: %w", err)
	}
	if decoded.Status != "success" {
		return nil, fmt.Errorf("loki query status %q", decoded.Status)
	}

	out := &QueryResult{Matches: make([]QueryMatch, 0)}
	for _, stream := range decoded.Data.Result {
		for _, entry := range stream.Values {
			if len(entry) < 2 {
				continue
			}
			timestamp, err := parseLokiTimestamp(entry[0])
			if err != nil {
				continue
			}
			out.Matches = append(out.Matches, QueryMatch{
				Timestamp: timestamp,
				Line:      entry[1],
				Labels:    cloneStringMap(stream.Stream),
			})
		}
	}
	return out, nil
}

type lokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string               `json:"resultType"`
		Result     []lokiStreamResponse `json:"result"`
	} `json:"data"`
}

type lokiStreamResponse struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func parseLokiTimestamp(raw string) (time.Time, error) {
	nanos, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, nanos).UTC(), nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
