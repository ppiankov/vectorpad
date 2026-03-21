package vectorcourt

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// StreamSpar connects to the SSE endpoint and returns a channel of spar events.
// The channel is closed when the stream ends (final event or context cancel).
func StreamSpar(ctx context.Context, endpoint, submissionID string) (<-chan SparEvent, error) {
	u := endpoint + "/v1/submissions/" + submissionID + "/stream"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect to stream: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("stream returned %d", resp.StatusCode)
	}

	ch := make(chan SparEvent, 16)
	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			var ev SparEvent
			if json.Unmarshal([]byte(data), &ev) != nil {
				continue
			}
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
			if ev.Final {
				return
			}
		}
	}()

	return ch, nil
}
