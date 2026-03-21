package extractor

import (
"encoding/json"
"fmt"
"log"
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

func normalize(raw json.RawMessage) (json.RawMessage, error) {
trimmed := strings.TrimSpace(string(raw))

if strings.HasPrefix(trimmed, "[") {
return raw, nil
}

if strings.HasPrefix(trimmed, "{") {
var obj map[string]json.RawMessage
if err := json.Unmarshal(raw, &obj); err != nil {
return nil, fmt.Errorf("unmarshal wrapper: %w", err)
}

priority := []string{
"tenders", "items", "results", "data",
"postings", "jobs", "listings", "opportunities",
"notices", "contracts", "procurements",
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

// last resort — log the keys we got so we can add them
keys := make([]string, 0, len(obj))
for k := range obj {
keys = append(keys, k)
}
log.Printf("normalize: no array found, top-level keys: %v", keys)
}

return nil, fmt.Errorf("no array found in TinyFish result")
}

func parseDeadline(s string) time.Time {
s = strings.TrimSpace(s)
if s == "" {
return time.Time{}
}
if idx := strings.Index(s, " "); idx != -1 {
s = s[:idx]
}
formats := []string{
"2006-01-02",
"02/01/2006",
"02-01-2006",
"02-Jan-2026",
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

func min(a, b int) int {
if a < b {
return a
}
return b
}
