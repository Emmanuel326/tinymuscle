package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Emmanuel326/tenderwatchafrica/portals"
)

const (
	endpoint       = "https://agent.tinyfish.ai/v1/automation/run-sse"
	requestTimeout = 5 * time.Minute
	CrawlTimeout   = requestTimeout
)

// SSEEvent represents a single event from the TinyFish SSE stream
type SSEEvent struct {
	Type   string          `json:"type"`
	Status string          `json:"status"`
	Result json.RawMessage `json:"resultJson,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// RunRequest is the payload sent to TinyFish
type RunRequest struct {
	URL            string            `json:"url"`
	Goal           string            `json:"goal"`
	BrowserProfile string            `json:"browser_profile"`
	ProxyConfig    ProxyConfig       `json:"proxy_config"`
	Headers        map[string]string `json:"headers,omitempty"`
	Cookies        map[string]string `json:"cookies,omitempty"`
}

// ProxyConfig tells TinyFish which geo to proxy through
type ProxyConfig struct {
	Enabled     bool   `json:"enabled"`
	CountryCode string `json:"country_code"`
}

// Agent is the TinyFish SSE client
type Agent struct {
	apiKey     string
	httpClient *http.Client
}

// New creates a new Agent
func New(apiKey string) *Agent {
	return &Agent{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// Result is what the Agent hands back to the caller
type Result struct {
	PortalID string
	Raw      json.RawMessage
	Err      error
}

// Run executes a stateful multi-step crawl against a portal.
// It streams SSE events and calls onEvent for each intermediate
// event so the caller can commit partial results to BBolt
// before the stream completes.
func (a *Agent) Run(
	ctx context.Context,
	portal portals.Portal,
	onEvent func(SSEEvent),
) Result {
	payload := RunRequest{
		URL:            portal.URL,
		Goal:           portal.Goal,
		BrowserProfile: "stealth",
		ProxyConfig: ProxyConfig{
			Enabled:     true,
			CountryCode: "KE",
		},
		Headers: portal.Headers,
		Cookies: portal.Cookies,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Result{PortalID: portal.ID, Err: fmt.Errorf("marshal: %w", err)}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return Result{PortalID: portal.ID, Err: fmt.Errorf("request: %w", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return Result{PortalID: portal.ID, Err: fmt.Errorf("http: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{
			PortalID: portal.ID,
			Err:      fmt.Errorf("unexpected status: %d", resp.StatusCode),
		}
	}

	return a.consumeStream(ctx, portal.ID, resp.Body, onEvent)
}

// consumeStream reads the SSE stream line by line.
// It commits partial results via onEvent and returns
// the final resultJson on COMPLETE.
func (a *Agent) consumeStream(
	ctx context.Context,
	portalID string,
	body io.Reader,
	onEvent func(SSEEvent),
) Result {
	scanner := bufio.NewScanner(body)

	var dataBuffer strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return Result{
				PortalID: portalID,
				Err:      ctx.Err(),
			}
		default:
		}

		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			dataBuffer.WriteString(data)

		case line == "":
			// blank line = end of event, flush buffer
			raw := dataBuffer.String()
			dataBuffer.Reset()

			if raw == "" {
				continue
			}

			var event SSEEvent
			if err := json.Unmarshal([]byte(raw), &event); err != nil {
				// malformed event, skip
				continue
			}

			// hand every event to the caller for partial commits
			if onEvent != nil {
				onEvent(event)
			}

			// terminal signal
			if event.Type == "COMPLETE" && event.Status == "COMPLETED" {
				return Result{
					PortalID: portalID,
					Raw:      event.Result,
				}
			}

			// hard failure from TinyFish
			if event.Type == "ERROR" || event.Status == "FAILED" {
				return Result{
					PortalID: portalID,
					Err:      fmt.Errorf("tinyfish error: %s", event.Error),
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Result{PortalID: portalID, Err: fmt.Errorf("scanner: %w", err)}
	}

	return Result{
		PortalID: portalID,
		Err:      fmt.Errorf("stream ended without COMPLETE event"),
	}
}
