package agent

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
)

// FetchDocument uses TinyFish to navigate to a tender notice URL,
// find all relevant documents, and extract their full text content.
func (a *Agent) FetchDocument(ctx context.Context, noticeURL string) (string, error) {
payload := RunRequest{
URL: noticeURL,
Goal: "Navigate to this tender notice page. Find all attached documents, " +
"PDF links, or downloadable files. Extract and return the complete " +
"text content of all tender documents including: scope of work, " +
"eligibility criteria, required documents, evaluation criteria, " +
"submission instructions, and contact details. " +
"Return as a single JSON object with key 'content' containing all extracted text.",
BrowserProfile: "lite",
}

body, err := json.Marshal(payload)
if err != nil {
return "", fmt.Errorf("marshal: %w", err)
}

req, err := http.NewRequestWithContext(
ctx,
http.MethodPost,
endpoint,
bytes.NewReader(body),
)
if err != nil {
return "", fmt.Errorf("request: %w", err)
}

req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-API-Key", a.apiKey)
req.Header.Set("Accept", "text/event-stream")
req.Header.Set("Cache-Control", "no-cache")
req.Header.Set("Connection", "keep-alive")

resp, err := a.httpClient.Do(req)
if err != nil {
return "", fmt.Errorf("http: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
errBody, _ := io.ReadAll(resp.Body)
return "", fmt.Errorf("unexpected status: %d — %s", resp.StatusCode, string(errBody))
}

result := a.consumeStream(ctx, "doc_fetch", resp.Body, nil)
if result.Err != nil {
return "", result.Err
}

// extract content field from result
var wrapper map[string]json.RawMessage
if err := json.Unmarshal(result.Raw, &wrapper); err != nil {
// if not wrapped just return raw as string
return string(result.Raw), nil
}

for _, key := range []string{"content", "text", "document", "result"} {
if val, ok := wrapper[key]; ok {
var s string
if err := json.Unmarshal(val, &s); err == nil {
return s, nil
}
return string(val), nil
}
}

return string(result.Raw), nil
}
