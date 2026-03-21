package analyzer

import (
"context"
"encoding/json"
"fmt"
"strings"
"time"

"google.golang.org/genai"

"github.com/Emmanuel326/tinymuscle/store"
)

// Analysis is the structured intelligence extracted from a tender document
type Analysis struct {
TenderID            string    `json:"tender_id"`
PortalID            string    `json:"portal_id"`
Summary             string    `json:"summary"`
EligibilityCriteria []string  `json:"eligibility_criteria"`
RequiredDocuments   []string  `json:"required_documents"`
EvaluationCriteria  []string  `json:"evaluation_criteria"`
Deadline            time.Time `json:"deadline"`
EstimatedValue      string    `json:"estimated_value"`
ContactPerson       string    `json:"contact_person"`
Qualifies           bool      `json:"qualifies"`
QualifyReasons      []string  `json:"qualify_reasons"`
DraftResponse       string    `json:"draft_response"`
AnalyzedAt          time.Time `json:"analyzed_at"`
}

// Analyzer reads tender documents and produces structured intelligence
type Analyzer struct {
client *genai.Client
model  string
}

// New creates a new Analyzer
func New(apiKey string) (*Analyzer, error) {
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
APIKey:  apiKey,
Backend: genai.BackendGeminiAPI,
})
if err != nil {
return nil, fmt.Errorf("gemini client: %w", err)
}

return &Analyzer{
client: client,
model:  "gemini-2.5-flash",
}, nil
}

// Analyze takes raw document content extracted by TinyFish,
// reads it against the business profile, and returns
// structured intelligence plus a ready-to-edit draft response.
func (a *Analyzer) Analyze(
ctx context.Context,
tender store.Tender,
documentContent string,
businessProfile string,
) (*Analysis, error) {
if documentContent == "" {
return nil, fmt.Errorf("empty document content")
}

prompt := fmt.Sprintf(`You are a senior procurement specialist helping a business respond to tenders.

Business Profile:
%s

Tender Details:
- Title: %s
- Issuing Entity: %s
- Reference: %s
- Estimated Value: %s

Raw Document Content:
%s

Your job is to analyze this tender document and return a JSON object only.
No preamble. No markdown. No explanation outside the JSON.

Return exactly this structure:
{
  "summary": "2-3 sentence plain English explanation of what they want",
  "eligibility_criteria": ["criterion 1", "criterion 2"],
  "required_documents": ["document 1", "document 2"],
  "evaluation_criteria": ["criterion 1", "criterion 2"],
  "estimated_value": "extracted value or empty string",
  "contact_person": "name and email if found, else empty string",
  "qualifies": true or false,
  "qualify_reasons": ["reason 1", "reason 2"],
  "draft_response": "A professional bid expression of interest letter, ready to edit and submit. Address it to the issuing entity. Reference the tender number. State the company's relevant experience. Keep it under 300 words."
}`,
businessProfile,
tender.Title,
tender.IssuingEntity,
tender.ReferenceNumber,
tender.EstimatedValue,
documentContent,
)

result, err := a.client.Models.GenerateContent(
ctx,
a.model,
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

type geminiResponse struct {
Summary             string   `json:"summary"`
EligibilityCriteria []string `json:"eligibility_criteria"`
RequiredDocuments   []string `json:"required_documents"`
EvaluationCriteria  []string `json:"evaluation_criteria"`
EstimatedValue      string   `json:"estimated_value"`
ContactPerson       string   `json:"contact_person"`
Qualifies           bool     `json:"qualifies"`
QualifyReasons      []string `json:"qualify_reasons"`
DraftResponse       string   `json:"draft_response"`
}

var gr geminiResponse
if err := json.Unmarshal([]byte(raw), &gr); err != nil {
return nil, fmt.Errorf("unmarshal analysis: %w", err)
}

analysis := &Analysis{
TenderID:            tender.PortalID + ":" + tender.ReferenceNumber,
PortalID:            tender.PortalID,
Summary:             gr.Summary,
EligibilityCriteria: gr.EligibilityCriteria,
RequiredDocuments:   gr.RequiredDocuments,
EvaluationCriteria:  gr.EvaluationCriteria,
Deadline:            tender.Deadline,
EstimatedValue:      gr.EstimatedValue,
ContactPerson:       gr.ContactPerson,
Qualifies:           gr.Qualifies,
QualifyReasons:      gr.QualifyReasons,
DraftResponse:       gr.DraftResponse,
AnalyzedAt:          time.Now(),
}

return analysis, nil
}

func stripMarkdown(s string) string {
s = strings.TrimSpace(s)
for _, fence := range []string{"```json", "```"} {
if strings.HasPrefix(s, fence) {
s = s[len(fence):]
}
if strings.HasSuffix(s, fence) {
s = s[:len(s)-len(fence)]
}
}
return strings.TrimSpace(s)
}
