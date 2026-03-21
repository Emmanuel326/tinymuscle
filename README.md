# TinyMuscle

TinyMuscle is a stateful web intelligence pipeline that turns any structured
page on the live internet into a monitored, deduplicated, queryable feed —
with no brittle CSS selectors, no headless Chrome configuration, and no
per-site maintenance burden.

Built for African government procurement portals: sites running decade-old PHP,
behind Cloudflare, with inconsistent pagination, broken SSL, and no public API.
If it works there, it works anywhere.

---

## The Problem

Government tenders, UN procurement notices, grant opportunities — the data
exists. It is public. It is valuable. But it lives behind the worst UIs on
the internet: JavaScript-rendered tables, multi-step pagination, session
cookies, CAPTCHA walls, and PDF links three clicks deep. A Nairobi-based
construction firm misses a KSh 50M contract not because they were
unqualified but because the portal was down on the day it was posted and
nobody checked again.

Traditional scrapers break the moment a developer renames a CSS class.
Scheduled curl jobs get IP-banned. Manual monitoring does not scale past
two or three portals.

---

## The Architecture

TinyMuscle makes one architectural bet: delegate all browser complexity to
TinyFish and own everything else.

```
TinyFish (browser agent)
    ↓  SSE stream — partial results committed in real time
Extractor (raw JSON → Tender structs)
    ↓  shape-agnostic — handles flat arrays and nested objects
Gemini 2.5 Flash (relevance scoring against business profile)
    ↓  single LLM call per batch — not per tender
BBolt (two-key deduplication)
    ↓  primary key: portalID + referenceNumber
    ↓  version key: SHA256 of content fields
SSE broadcast → Dashboard
    ↓  on demand
TinyFish (document fetcher)
    ↓  navigates to tender notice, finds and extracts all documents
Gemini 2.5 Flash (analyzer)
    ↓  reads documents against business profile
Analysis: summary, eligibility, required docs, draft bid response
```

Each stage has one job. Nothing crosses boundaries. The scheduler does not
know about HTTP. The extractor does not know about storage. The notifier
does not know about portals.

---

## The Deduplication Model

TinyMuscle uses a two-key model:

- **Primary key**: `portalID:referenceNumber` — stable identity from the source
- **Version key**: SHA256 of title + issuing entity + deadline + estimated value

On every crawl:

1. Key missing → `status: new` → alert fired
2. Key present, hash unchanged → silent, no write
3. Key present, hash changed → `status: updated`, version incremented → alert fired

An addendum that extends a deadline is an update, not a new tender.
A re-crawl of an unchanged page produces zero writes.

---

## The Intelligence Model

Finding a tender is the beginning, not the end.

`POST /tenders/{portalID}/{referenceNumber}/analyze` triggers a TinyFish
session that navigates to the tender notice page, finds all attached documents,
and extracts their full text. Gemini 2.5 Flash reads that content against the
business profile and returns a structured analysis including a ready-to-edit
draft bid response.

**TinyMuscle never auto-submits.** The draft response is for human review.
The final decision is always the user's.

---

## Tradeoffs

**BBolt over Postgres.** No server, no migrations, single file on disk.
The query patterns here do not need SQL.

**Single binary.** No separate scheduler, no message queue, no worker pool.
One `./tinymuscle` starts everything. One SIGTERM stops it cleanly.

**SSE over WebSockets.** Unidirectional, HTTP/1.1 compatible, trivially
reconnectable. The dashboard does not need to send data over the event channel.

**Goal-based extraction over CSS selectors.** When a portal redesigns its UI,
nothing in TinyMuscle breaks.

**On-demand analysis over automatic.** Not every tender warrants deep analysis.
The business decides which tenders are worth the deeper read.

---

## What This Is Good At

- Portals with aggressive anti-bot measures
- Sites requiring multi-step navigation to reach listings
- Paginated results across an unknown number of pages
- Turning raw tender documents into actionable procurement intelligence

## What This Is Not Good At

- Sub-minute latency — TinyFish sessions take 1-3 minutes per portal
- Portals requiring MFA-authenticated sessions
- High-frequency scenarios where milliseconds matter

---

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 18+ (for frontend)
- TinyFish API key — from [tinyfish.ai](https://tinyfish.ai)
- Gemini API key — from [aistudio.google.com](https://aistudio.google.com) (optional)

### Setup

```bash
git clone https://github.com/Emmanuel326/tinymuscle
cd tinymuscle
cp .env.example .env
# edit .env — set TINYFISH_API_KEY and GEMINI_API_KEY
```

### Build and run backend

```bash
go build -o tinymuscle ./cmd/main.go
export $(cat .env | xargs) && ./tinymuscle
```

### Run frontend

```bash
cd tenderwatch-frontend
npm install
npm run dev
```

Dashboard available at `http://localhost:5173`
Backend API at `http://localhost:8080`

### Cross-compile

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o tinymuscle ./cmd/main.go

# macOS
GOOS=darwin GOARCH=arm64 go build -o tinymuscle ./cmd/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o tinymuscle.exe ./cmd/main.go
```

---

## Environment Variables

```
TINYFISH_API_KEY   required*   TinyFish Web Agent API key
GEMINI_API_KEY     optional    Gemini 2.5 Flash API key
DB_PATH            optional    BBolt database path (default: tinymuscle.db)
ADDR               optional    HTTP listen address (default: :8080)
USE_MOCK           optional    true = bypass TinyFish, use mock agent
```

*not required when USE_MOCK=true

---

## API Reference

All endpoints return JSON. All POST bodies are JSON.
Reference numbers containing `/` must be URL-encoded as `%2F` in the path.

---

### Portals

#### POST /portals — Register a portal

**Frontend:** Render a form with these fields. `interval_min` defaults to 60.
`business_profile` and `relevance_threshold` are optional — when omitted,
all tenders are kept without AI filtering.

```bash
curl -s -X POST http://localhost:8080/portals \
  -H "Content-Type: application/json" \
  -d '{
    "id": "ungm",
    "name": "UN Global Marketplace",
    "url": "https://www.ungm.org/Public/Notice",
    "goal": "Navigate to the procurement notices page, extract all visible open tenders. For each tender extract: title, reference_number, issuing_entity, deadline, estimated_value, source_url. Return as JSON array.",
    "interval_min": 60,
    "business_profile": "We are a Nairobi-based ICT firm specialising in network infrastructure and government systems integration",
    "relevance_threshold": 60
  }'
```

**Response: 201 Created**

```json
{
  "id": "ungm",
  "name": "UN Global Marketplace",
  "url": "https://www.ungm.org/Public/Notice",
  "goal": "...",
  "interval_min": 60,
  "business_profile": "...",
  "relevance_threshold": 60
}
```

**Frontend notes:**
- `id` — user-defined slug, no spaces, e.g. `ungm`, `treasury_ke`
- `goal` — natural language instruction to TinyFish. Keep it specific.
- `interval_min` — how often to crawl. Minimum recommended: 30.
- `business_profile` — plain English description of the business. Gemini uses this to score relevance.
- `relevance_threshold` — 0-100. Tenders scoring below this are dropped. Default: 60.
- On success, the backend immediately fires a crawl in the background. No user action needed.

---

#### GET /portals — List all portals

**Frontend:** Render a portal management list. Show name, URL, interval, and a delete button.

```bash
curl -s http://localhost:8080/portals
```

**Response: 200 OK**

```json
[
  {
    "id": "ungm",
    "name": "UN Global Marketplace",
    "url": "https://www.ungm.org/Public/Notice",
    "goal": "...",
    "interval_min": 60,
    "business_profile": "...",
    "relevance_threshold": 60
  }
]
```

---

#### DELETE /portals/{id} — Remove a portal

**Frontend:** Confirm dialog → call this → remove from list.

```bash
curl -s -X DELETE http://localhost:8080/portals/ungm
```

**Response: 204 No Content**

---

### Tenders

#### GET /tenders — All tenders across all portals

**Frontend:** Main tender feed. Poll on page load. Combine with SSE for live updates.

```bash
curl -s http://localhost:8080/tenders | python3 -m json.tool
```

**Response: 200 OK** — array of tender objects (see Tender Object below)

**Frontend notes:**
- `status: "new"` — show a green NEW badge
- `status: "updated"` — show a blue UPDATED badge with version number
- `deadline` — if `0001-01-01T00:00:00Z` the deadline was not parseable, show "—"
- `estimated_value` — may be empty string, show "—" if so
- `source_url` — always link directly to the source portal page

---

#### GET /tenders/{portalID} — Tenders for a specific portal

**Frontend:** Filtered view per portal. Use `portal_id` from the tender object.

```bash
curl -s http://localhost:8080/tenders/ungm | python3 -m json.tool
```

**Response: 200 OK** — array of tender objects

---

### Analysis

Analysis is a two-step process. Always POST first, then GET.

#### POST /tenders/{portalID}/{referenceNumber}/analyze — Trigger analysis

**Frontend:** "Analyze" button on tender detail card. Disable the button after
click and show a loading spinner. Poll the GET endpoint every 30 seconds.

**Important:** Reference numbers containing `/` must be URL-encoded.
Use `encodeURIComponent(tender.reference_number)` in JavaScript — never encode manually.

```bash
# reference number with no slashes
curl -s -X POST \
  http://localhost:8080/tenders/ungm/30000022713/analyze

# reference number with slashes — encode / as %2F
curl -s -X POST \
  "http://localhost:8080/tenders/ungm/RFP%2FHCR%2FSYR%2F2026%2F2390/analyze"
```

**Response: 202 Accepted**

```json
{
  "message": "analysis started — poll GET /tenders/ungm/30000022713/analysis"
}
```

**Frontend notes:**
- 202 means the job started, not that it's done
- TinyFish fetches the documents, Gemini reads them — takes 2-5 minutes
- Show a "Analyzing..." state on the button
- Poll `GET /tenders/{portalID}/{referenceNumber}/analysis` every 30s
- If GET returns 404, analysis is still running — keep polling
- If GET returns 200, analysis is ready — render the Analysis Card

**JavaScript pattern:**

```javascript
const analyze = async (tender) => {
  const ref = encodeURIComponent(tender.reference_number)
  await fetch(`/tenders/${tender.portal_id}/${ref}/analyze`, { method: 'POST' })

  // poll until ready
  const poll = setInterval(async () => {
    const res = await fetch(`/tenders/${tender.portal_id}/${ref}/analysis`)
    if (res.ok) {
      clearInterval(poll)
      const analysis = await res.json()
      renderAnalysis(analysis)
    }
  }, 30000)
}
```

---

#### GET /tenders/{portalID}/{referenceNumber}/analysis — Get analysis result

**Frontend:** Poll this after triggering analyze. Render the Analysis Card when 200 returns.

```bash
curl -s \
  http://localhost:8080/tenders/ungm/30000022713/analysis \
  | python3 -m json.tool
```

**Response: 200 OK** — analysis object (see Analysis Object below)
**Response: 404** — analysis not yet ready, keep polling

---

### Live Events (SSE)

#### GET /events — Live tender stream

**Frontend:** Connect once on page load. Append new tenders to the feed in real time.
Reconnect automatically on disconnect.

```bash
curl -s http://localhost:8080/events
```

**Events arrive as:**

```
data: {"type":"new","tender":{...tender object...}}

data: {"type":"updated","tender":{...tender object...}}

: heartbeat
```

Heartbeat comment sent every 30 seconds — ignore it, it just keeps the connection alive.

**JavaScript pattern:**

```javascript
const es = new EventSource('http://localhost:8080/events')

es.onmessage = (e) => {
  const { type, tender } = JSON.parse(e.data)
  if (type === 'new') prependToFeed(tender)
  if (type === 'updated') updateInFeed(tender)
}

es.onerror = () => {
  // EventSource reconnects automatically — no action needed
}
```

---

## Data Objects

### Tender Object

```json
{
  "reference_number": "30000022713",
  "portal_id": "ungm",
  "title": "Servicios de Digitalización de documentos",
  "issuing_entity": "IOM",
  "deadline": "2026-03-22T00:00:00Z",
  "estimated_value": "",
  "source_url": "https://www.ungm.org/Public/Notice/294509",
  "content_hash": "05351fd3...",
  "version": 1,
  "last_updated": "2026-03-21T19:45:26Z",
  "status": "new"
}
```

| Field | Type | Notes |
|---|---|---|
| reference_number | string | Use with encodeURIComponent in URLs |
| portal_id | string | Matches the portal id it came from |
| title | string | Display as card heading |
| issuing_entity | string | Display as subtitle |
| deadline | string (ISO 8601) | `0001-01-01T00:00:00Z` means unparseable — show "—" |
| estimated_value | string | May be empty — show "—" |
| source_url | string | Always link to this |
| content_hash | string | Internal — do not display |
| version | int | Show if > 1, indicates tender was updated |
| last_updated | string (ISO 8601) | Show as relative time |
| status | string | `new` or `updated` |

---

### Analysis Object

```json
{
  "tender_id": "ungm:30000022713",
  "portal_id": "ungm",
  "summary": "IOM requires a firm to digitize documents...",
  "eligibility_criteria": [
    "Valid company registration",
    "Experience in document digitization"
  ],
  "required_documents": [
    "Technical proposal",
    "Company registration certificate",
    "Price quotation"
  ],
  "evaluation_criteria": [
    "Technical compliance 40%",
    "Price 60%"
  ],
  "deadline": "2026-03-22T00:00:00Z",
  "estimated_value": "",
  "contact_person": "procurement@iom.int",
  "qualifies": true,
  "qualify_reasons": [
    "ICT background aligns with digitization work",
    "Government systems integration experience is relevant"
  ],
  "draft_response": "Dear IOM Procurement Unit, We write in response to...",
  "analyzed_at": "2026-03-21T19:50:00Z"
}
```

| Field | Type | Frontend guidance |
|---|---|---|
| summary | string | Show as paragraph at top of Analysis Card |
| eligibility_criteria | []string | Render as checklist |
| required_documents | []string | Render as checklist with checkboxes |
| evaluation_criteria | []string | Render as numbered list |
| qualifies | bool | Show green YES / red NO badge prominently |
| qualify_reasons | []string | Show below the qualifies badge |
| draft_response | string | Render in a textarea. Add "Copy" button. Add disclaimer: "Review before sending — TinyMuscle never auto-submits." |
| contact_person | string | Show as mailto link if it contains @ |
| analyzed_at | string | Show as "Analyzed X minutes ago" |

---

## Tested Portals

```
UN Global Marketplace   https://www.ungm.org/Public/Notice        15 tenders  ~1 min
Kenya National Treasury https://www.treasury.go.ke/tenders        15 tenders  ~3 min
Hacker News Jobs        https://news.ycombinator.com/jobs          10 items    ~1 min
```

---

## Project Structure

```
tinymuscle/
├── agent/                TinyFish SSE client, document fetcher, mock
├── analyzer/             Gemini document intelligence + draft response
├── api/                  Chi router, REST handlers, SSE endpoint
├── cmd/                  Binary entrypoint, wiring, graceful shutdown
├── extractor/            Shape-agnostic JSON → Tender normalizer
├── matcher/              Gemini relevance scoring
├── notifier/             Fan-out SSE broadcaster, drop semantics for slow clients
├── portals/              Portal type definition
├── scheduler/            Cron engine, immediate-fire on registration, crawl pipeline
├── store/                BBolt, two-key deduplication, version tracking, analyses
└── tenderwatch-frontend/ React dashboard
```

---

## Built for the TinyFish $2M Pre-Accelerator Hackathon 2026

