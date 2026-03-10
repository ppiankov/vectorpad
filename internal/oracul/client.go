package oracul

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	consultTimeout   = 480 * time.Second
	preflightTimeout = 10 * time.Second
	authHeader       = "X-Oracul-Key"
)

// Client is an HTTP client for the Oracul API.
type Client struct {
	endpoint string
	apiKey   string
	http     *http.Client
}

// NewClient creates an Oracul API client.
func NewClient(endpoint, apiKey string) *Client {
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		http:     &http.Client{},
	}
}

// Consult sends a case to Oracul for deliberation.
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

// APIError represents a non-2xx response from the Oracul API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("oracul API %d: %s", e.StatusCode, e.Message)
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
