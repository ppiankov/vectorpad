package vectorcourt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	consultTimeout   = 480 * time.Second
	preflightTimeout = 10 * time.Second
	accountTimeout   = 5 * time.Second
	precedentTimeout = 10 * time.Second
	authHeader       = "X-VC-Key"
)

// Client is an HTTP client for the VectorCourt API.
type Client struct {
	endpoint string
	apiKey   string
	http     *http.Client
}

// NewClient creates a VectorCourt API client.
func NewClient(endpoint, apiKey string) *Client {
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		http:     &http.Client{},
	}
}

// Consult sends a case to VectorCourt for deliberation.
// Returns the raw JSON response envelope (VP passes through without parsing).
func (c *Client) Consult(ctx context.Context, req *ConsultRequest) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, consultTimeout)
	defer cancel()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/consult", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set(authHeader, c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("consult request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	return json.RawMessage(respBody), nil
}

// Preflight validates a case without starting deliberation.
func (c *Client) Preflight(ctx context.Context, question string, filing *CaseFiling) (*PreflightResult, error) {
	ctx, cancel := context.WithTimeout(ctx, preflightTimeout)
	defer cancel()

	req := ConsultRequest{
		Question: question,
		Filing:   filing,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/preflight", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set(authHeader, c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("preflight request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var result PreflightResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse preflight response: %w", err)
	}

	return &result, nil
}

// PreflightGate runs preflight and returns a gate decision.
func (c *Client) PreflightGate(ctx context.Context, question string, filing *CaseFiling) (*GateResult, error) {
	result, err := c.Preflight(ctx, question, filing)
	if err != nil {
		return nil, err
	}

	gate := &GateResult{
		Verdict:  result.Verdict,
		Tier:     result.Tier,
		Quality:  result.FilingQuality,
		Warnings: result.Warnings,
		Reason:   result.Reason,
	}

	gate.Allowed = result.Verdict == "ACCEPTED"
	return gate, nil
}

// Account fetches the current account status (tier, quota, reset time).
func (c *Client) Account(ctx context.Context) (*AccountStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, accountTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v1/account", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.apiKey != "" {
		httpReq.Header.Set(authHeader, c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("account request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var status AccountStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("parse account response: %w", err)
	}

	return &status, nil
}

// SearchPrecedents searches for similar past decisions.
func (c *Client) SearchPrecedents(ctx context.Context, question string, limit int) (*PrecedentSearch, error) {
	ctx, cancel := context.WithTimeout(ctx, precedentTimeout)
	defer cancel()

	u := c.endpoint + "/v1/precedents/search?q=" + url.QueryEscape(question) + "&limit=" + strconv.Itoa(limit)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.apiKey != "" {
		httpReq.Header.Set(authHeader, c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("precedent search request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var result PrecedentSearch
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse precedent response: %w", err)
	}

	return &result, nil
}

const outcomeTimeout = 10 * time.Second

// ReportOutcome submits an outcome for a previously decided case.
func (c *Client) ReportOutcome(ctx context.Context, caseID string, req *OutcomeRequest) (*OutcomeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, outcomeTimeout)
	defer cancel()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/cases/"+caseID+"/outcome", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set(authHeader, c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("outcome request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var result OutcomeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse outcome response: %w", err)
	}

	return &result, nil
}

// APIError represents a non-2xx response from the VectorCourt API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("vectorcourt API %d: %s", e.StatusCode, e.Message)
}

func parseAPIError(statusCode int, body []byte) *APIError {
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return &APIError{StatusCode: statusCode, Message: errResp.Error}
	}

	msg := http.StatusText(statusCode)
	switch statusCode {
	case http.StatusUnauthorized:
		msg = "invalid or missing API key"
	case http.StatusTooManyRequests:
		msg = "rate limited"
	}
	return &APIError{StatusCode: statusCode, Message: msg}
}
