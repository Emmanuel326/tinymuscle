package matcher

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	"github.com/Emmanuel326/tinymuscle/store"
)

// ScoredTender wraps a tender with an AI relevance score
type ScoredTender struct {
	store.Tender
	Relevance int    `json:"relevance"` // 0-100
	Reason    string `json:"reason"`
}

// Matcher scores tenders against a business profile using Gemini
type Matcher struct {
	client *genai.Client
	model  string
}

// New creates a new Matcher
func New(apiKey string) (*Matcher, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}

	return &Matcher{
		client: client,
		model:  "gemini-2.5-flash",
	}, nil
}

// Score takes a business profile and a batch of tenders,
// returns only those with relevance >= threshold.
// It makes a single LLM call for the entire batch.
func (m *Matcher) Score(
	ctx context.Context,
	businessProfile string,
	tenders []store.Tender,
	threshold int,
) ([]ScoredTender, error) {
	if len(tenders) == 0 {
		return nil, nil
	}

	// build a minimal representation of tenders for the prompt
	type tenderSummary struct {
		Index          int    `json:"index"`
		Title          string `json:"title"`
		IssuingEntity  string `json:"issuing_entity"`
		EstimatedValue string `json:"estimated_value"`
	}

	summaries := make([]tenderSummary, len(tenders))
	for i, t := range tenders {
		summaries[i] = tenderSummary{
			Index:          i,
			Title:          t.Title,
			IssuingEntity:  t.IssuingEntity,
			EstimatedValue: t.EstimatedValue,
		}
	}

	summaryJSON, err := json.Marshal(summaries)
	if err != nil {
		return nil, fmt.Errorf("marshal summaries: %w", err)
	}

	prompt := fmt.Sprintf(`You are a procurement analyst.

Business profile: %s

Score each tender for relevance to this business on a scale of 0-100.
Return ONLY a JSON array. No preamble, no markdown, no explanation outside the array.

Format:
[{"index": 0, "relevance": 85, "reason": "Direct match for ICT infrastructure work"}]

Tenders:
%s`, businessProfile, string(summaryJSON))

	result, err := m.client.Models.GenerateContent(
		ctx,
		m.model,
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	raw := result.Candidates[0].Content.Parts[0].Text
	raw = stripMarkdown(raw)

	type scoreResult struct {
		Index     int    `json:"index"`
		Relevance int    `json:"relevance"`
		Reason    string `json:"reason"`
	}

	var scores []scoreResult
	if err := json.Unmarshal([]byte(raw), &scores); err != nil {
		return nil, fmt.Errorf("unmarshal scores: %w", err)
	}

	var matched []ScoredTender
	for _, s := range scores {
		if s.Index < 0 || s.Index >= len(tenders) {
			continue
		}
		if s.Relevance >= threshold {
			matched = append(matched, ScoredTender{
				Tender:    tenders[s.Index],
				Relevance: s.Relevance,
				Reason:    s.Reason,
			})
		}
	}

	return matched, nil
}

// stripMarkdown removes ```json fences if Gemini wraps its response
func stripMarkdown(s string) string {
	if len(s) == 0 {
		return s
	}
	// trim ```json ... ``` wrappers
	for _, fence := range []string{"```json", "```"} {
		if len(s) > len(fence) {
			if s[:len(fence)] == fence {
				s = s[len(fence):]
			}
			if s[len(s)-len(fence):] == fence {
				s = s[:len(s)-len(fence)]
			}
		}
	}
	return s
}
