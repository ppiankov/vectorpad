package stash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultOllamaURL  = "http://localhost:11434"
	defaultEmbedModel = "nomic-embed-text"
	embedTimeout      = 5 * time.Second
)

// Embedder computes text embeddings via Ollama's local API.
type Embedder struct {
	baseURL   string
	model     string
	client    *http.Client
	available *bool // nil = unchecked, true/false = cached
}

// NewEmbedder creates an embedder with default Ollama settings.
func NewEmbedder() *Embedder {
	return &Embedder{
		baseURL: defaultOllamaURL,
		model:   defaultEmbedModel,
		client: &http.Client{
			Timeout: embedTimeout,
		},
	}
}

// NewEmbedderWithURL creates an embedder pointing at a custom Ollama URL.
func NewEmbedderWithURL(baseURL, model string) *Embedder {
	return &Embedder{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: embedTimeout,
		},
	}
}

// Available checks if Ollama is reachable and the model is pulled.
// Result is cached for the session.
func (e *Embedder) Available() bool {
	if e.available != nil {
		return *e.available
	}

	resp, err := e.client.Get(e.baseURL + "/api/tags")
	if err != nil {
		e.cacheAvailable(false)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		e.cacheAvailable(false)
		return false
	}

	// Check if our model is in the list.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.cacheAvailable(false)
		return false
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tags); err != nil {
		e.cacheAvailable(false)
		return false
	}

	for _, m := range tags.Models {
		if m.Name == e.model || m.Name == e.model+":latest" {
			e.cacheAvailable(true)
			return true
		}
	}

	e.cacheAvailable(false)
	return false
}

func (e *Embedder) cacheAvailable(v bool) {
	e.available = &v
}

// Embed computes the embedding vector for the given text.
func (e *Embedder) Embed(text string) ([]float32, error) {
	reqBody, err := json.Marshal(map[string]string{
		"model":  e.model,
		"prompt": text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	resp, err := e.client.Post(e.baseURL+"/api/embeddings", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ollama embed request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed returned status %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	// Convert float64 (JSON default) to float32 for storage.
	vec := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}
