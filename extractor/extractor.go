package extractor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Emmanuel326/tenderwatchafrica/store"
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
		if r.ReferenceNumber == "" || r.Title == "" {
			continue
		}
		tenders = append(tenders, store.Tender{
			ReferenceNumber: strings.TrimSpace(r.ReferenceNumber),
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
		return json.RawMessage("[" + trimmed + "]"), nil
	}
	return nil, fmt.Errorf("unexpected JSON shape: %s", trimmed[:min(len(trimmed), 100)])
}

func parseDeadline(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"January 2, 2006",
		"2 January 2006",
		"02 Jan 2006",
		"Jan 02, 2006",
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
