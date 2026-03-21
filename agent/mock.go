package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Emmanuel326/tinymuscle/portals"
)

// MockAgent simulates TinyFish responses for local development
// and testing without consuming API credits.
type MockAgent struct{}

// NewMock creates a MockAgent
func NewMock() *MockAgent {
	return &MockAgent{}
}

// Run simulates a TinyFish SSE crawl and returns fake tenders
func (m *MockAgent) Run(
	ctx context.Context,
	portal portals.Portal,
	onEvent func(SSEEvent),
) Result {
	// simulate network latency
	time.Sleep(500 * time.Millisecond)

	// fire a fake intermediate event
	if onEvent != nil {
		onEvent(SSEEvent{
			Type:   "PROGRESS",
			Status: "RUNNING",
		})
	}

	tenders := mockTenders(portal.ID)

	raw, err := json.Marshal(tenders)
	if err != nil {
		return Result{PortalID: portal.ID, Err: err}
	}

	return Result{
		PortalID: portal.ID,
		Raw:      json.RawMessage(raw),
	}
}

// mockTenders returns realistic fake tenders per portal
func mockTenders(portalID string) []map[string]string {
	now := time.Now()
	deadline := now.AddDate(0, 0, 14).Format("2006-01-02")

	return []map[string]string{
		{
			"reference_number": portalID + "/001/2026",
			"title":            "Supply of Office Furniture and Equipment",
			"issuing_entity":   "Ministry of Public Service",
			"deadline":         deadline,
			"estimated_value":  "KES 2,500,000",
			"source_url":       "https://example.go.ke/tenders/001",
		},
		{
			"reference_number": portalID + "/002/2026",
			"title":            "Construction of Access Roads — Phase II",
			"issuing_entity":   "Kenya National Highways Authority",
			"deadline":         deadline,
			"estimated_value":  "KES 45,000,000",
			"source_url":       "https://example.go.ke/tenders/002",
		},
		{
			"reference_number": portalID + "/003/2026",
			"title":            "Provision of ICT Infrastructure Support Services",
			"issuing_entity":   "Communications Authority of Kenya",
			"deadline":         deadline,
			"estimated_value":  "KES 8,750,000",
			"source_url":       "https://example.go.ke/tenders/003",
		},
	}
}
