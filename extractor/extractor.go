package extractor

import (
"encoding/json"
"fmt"
"strings"
"time"

"github.com/Emmanuel326/tinymuscle/store"
)

type rawTender struct {
ReferenceNumber string `json:"reference_number"`
Title           string `json:"title"`
IssuingEntity   string `json:"issuing_entity"`
Deadline        string `json:"deadline"`
EstimatedValue  string `json:"estimated_value"`
SourceURL       string `json:"source_url"`
}

// Extract parses a raw TinyFish JSON result into a slice of Tenders.
// TinyFish may return results in multiple shapes:
// - flat array: [{"title": ...}, ...]
// - wrapped object: {"tenders": [...]} or {"jobs": [...]} or {"postings": [...]}
func Extract(portalID string, raw json.RawMessage) ([]store.Tender, error) {
if len(raw) == 0 {
return nil, fmt.Errorf("empty result from TinyFish")
}

normalized, err := normalize(raw)
if err != nil {
return nil, fmt.Errorf("normalize: %w", err)
}

var raws []rawTender
if err := json.Unmarshal(normalized, &raws); err != nil {
return nil, fmt.Errorf("unmarshal tenders: %w", err)
}

tenders := make([]store.Tender, 0, len(raws))
for _, r := range raws {
if r.ReferenceNumber == "" && r.Title == "" {
continue
}
// fall back to title as reference number if missing
ref := strings.TrimSpace(r.ReferenceNumber)
if ref == "" {
ref = strings.TrimSpace(r.Title)
}
tenders = append(tenders, store.Tender{
ReferenceNumber: ref,
PortalID:        portalID,
Title:           strings.TrimSpace(r.Title),
IssuingEntity:   strings.TrimSpace(r.IssuingEntity),
Deadline:        parseDeadline(r.Deadline),
EstimatedValue:  strings.TrimSpace(r.EstimatedValue),
SourceURL:       strings.TrimSpace(r.SourceURL),
})
}

return tenders, nil
}

// normalize ensures raw is always a JSON array.
// TinyFish returns results in various shapes depending on the goal.
// We try to find an array anywhere in the top-level object.
func normalize(raw json.RawMessage) (json.RawMessage, error) {
trimmed := strings.TrimSpace(string(raw))

// already a flat array
if strings.HasPrefix(trimmed, "[") {
return raw, nil
}

// wrapped object — find the first array value in the top-level keys
if strings.HasPrefix(trimmed, "{") {
var obj map[string]json.RawMessage
if err := json.Unmarshal(raw, &obj); err != nil {
return nil, fmt.Errorf("unmarshal wrapper: %w", err)
}

// priority keys first, then any array we find
priority := []string{
"tenders", "items", "results", "data",
"postings", "jobs", "listings", "opportunities",
}

for _, key := range priority {
if val, ok := obj[key]; ok {
if strings.HasPrefix(strings.TrimSpace(string(val)), "[") {
return val, nil
}
}
}

// fallback — find any array value
for _, val := range obj {
if strings.HasPrefix(strings.TrimSpace(string(val)), "[") {
return val, nil
}
}
}

return nil, fmt.Errorf("no array found in TinyFish result")
}

// parseDeadline attempts to parse a deadline string into a time.Time.
func parseDeadline(s string) time.Time {
    s = strings.TrimSpace(s)
    if s == "" {
        return time.Time{}
    }
    // strip timezone and time component — keep date only
    if idx := strings.Index(s, " "); idx != -1 {
        s = s[:idx]
    }
    // now s is something like "21-Mar-2026" or "2026-03-21"
    formats := []string{
        "2006-01-02",
        "02/01/2006",
        "02-01-2006",
        "02-Jan-2006",
        "January 2, 2006",
        "2 January 2006",
        "02 Jan 2006",
        "Jan 02, 2006",
        "Mon,",
    }
    for _, f := range formats {
        if t, err := time.Parse(f, s); err == nil {
            return t
        }
    }
    return time.Time{}
}
