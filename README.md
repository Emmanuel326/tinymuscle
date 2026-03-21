# TinyMuscle

TinyMuscle is a stateful web intelligence pipeline that turns any structured
page on the live internet into a monitored, deduplicated, queryable feed —
with no brittle CSS selectors, no headless Chrome configuration, and no
per-site maintenance burden.

It was built for African government procurement portals: sites running
decade-old PHP, behind Cloudflare, with inconsistent pagination, broken SSL,
and no public API. If it works there, it works anywhere.

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
does not know about portals. This is not accidental — it is the only way
to keep a system like this maintainable as the number of portals grows.

## The Deduplication Model

Most web intelligence tools key by URL or by a hash of the entire document.
Both are wrong.

Keying by URL breaks when a portal repaginates. Keying by full document hash
produces false positives when a timestamp in the page footer changes. Neither
tells you what changed or why it matters.

TinyMuscle uses a two-key model:

- **Primary key**: `portalID:referenceNumber` — stable identity derived from
  the source document's own reference system.
- **Version key**: SHA256 of title + issuing entity + deadline + estimated
  value. Only fields that matter to a bidder contribute to the hash.

On every crawl, three outcomes are possible:

1. Key missing → `status: new` → alert fired
2. Key present, hash unchanged → silent, no write
3. Key present, hash changed → `status: updated`, version incremented → alert fired

An addendum that extends a deadline is an update, not a new tender. A
re-crawl of an unchanged page produces zero writes.

## The Agent Model

TinyMuscle does not scrape. It issues goals.

Each portal registration includes a natural language goal:

```json
{
  "goal": "Navigate to the procurement notices page, extract all visible open tenders. For each tender extract: title, reference_number, issuing_entity, deadline, estimated_value, source_url. Return as JSON array."
}
```

TinyFish receives this goal and executes a stateful browser session:
JavaScript rendering, pagination clicks, Cloudflare bypass, session
management. It streams results back via SSE. TinyMuscle commits partial
results to BBolt as the stream arrives — a connection drop halfway through
a 200-tender paginated portal loses nothing already extracted.

## The Relevance Model

Raw extraction produces signal and noise in equal measure. A procurement
portal lists everything: office furniture, road construction, ICT
infrastructure, consultancy services. An ICT firm does not need to know
about the furniture tender.

When a portal is registered with a `business_profile`, every extracted
tender batch is scored by Gemini 2.5 Flash against that profile in a single
API call. Only tenders above `relevance_threshold` (default: 60/100) are
stored and surfaced. When no `business_profile` is provided, all tenders
are kept. The system degrades gracefully rather than failing.

## The Intelligence Model

Finding a tender is the beginning, not the end. Once a relevant tender is
detected, TinyMuscle can go deeper on demand.

A POST to `/tenders/{portalID}/{referenceNumber}/analyze` triggers a second
TinyFish session that navigates to the tender notice page, finds all attached
documents — PDFs, Word files, tender specifications — and extracts their full
text content. Gemini 2.5 Flash then reads that content against the registered
business profile and returns:

- Plain English summary of what the issuing entity wants
- Eligibility criteria — do you qualify?
- Required documents checklist
- Evaluation criteria — how they score bids
- Contact person and submission details
- A ready-to-edit draft bid response letter

The analysis is stored in BBolt and served via GET. The business owner opens
one screen and knows exactly what to do.

## Tradeoffs

**BBolt over Postgres.** BBolt is an embedded key-value store. It requires
no server, no connection pool, no migrations. The entire database is a
single file. For a pipeline that writes in batches and reads on API
requests, BBolt's sequential write performance is more than adequate.
The tradeoff is no SQL — but the query patterns here do not need it.

**Single binary.** There is no separate scheduler process, no message
queue, no worker pool. The scheduler, API server, and crawl pipeline run
in the same process. A single `./tinymuscle` starts everything. A single
SIGTERM stops it cleanly.

**SSE over WebSockets.** The dashboard receives tender events over
Server-Sent Events, not WebSockets. SSE is unidirectional, HTTP/1.1
compatible, and trivially reconnectable. The dashboard does not send data
to the server over the event channel — it has the REST API for that.

**Goal-based extraction over CSS selectors.** The extractor has no
knowledge of any specific portal's HTML structure. When a portal redesigns
its UI, nothing in TinyMuscle breaks. The tradeoff is dependency on
TinyFish's interpretation accuracy — a better failure mode than maintaining
hundreds of brittle selectors.

**On-demand analysis over automatic.** Document analysis is triggered
explicitly, not on every new tender. Not every tender warrants deep
analysis, and TinyFish sessions are not free. The business decides which
tenders are worth the deeper read.

## What This Is Good At

- Portals with aggressive anti-bot measures
- Sites that require multi-step navigation to reach listings
- Paginated results across an unknown number of pages
- Any situation where the cost of a missed opportunity exceeds the cost of running the pipeline
- Turning raw tender documents into actionable procurement intelligence

## What This Is Not Good At

- Real-time data with sub-minute latency — TinyFish sessions take 1-3 minutes per portal
- Portals requiring authenticated sessions with MFA
- High-frequency scenarios where milliseconds matter

## Running

```bash
git clone https://github.com/Emmanuel326/tinymuscle
cd tinymuscle
cp .env.example .env
# set TINYFISH_API_KEY and GEMINI_API_KEY
go build -o tinymuscle ./cmd/main.go
export $(cat .env | xargs) && ./tinymuscle
```

## Environment Variables

```
TINYFISH_API_KEY   required*   TinyFish Web Agent API key
GEMINI_API_KEY     optional    Gemini 2.5 Flash API key
DB_PATH            optional    BBolt database path (default: tinymuscle.db)
ADDR               optional    HTTP listen address (default: :8080)
USE_MOCK           optional    Set to true to bypass TinyFish for local dev
```

*not required when USE_MOCK=true

## API

### Register a portal

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

### Register a portal (mock mode — no AI filtering)

```bash
curl -s -X POST http://localhost:8080/portals \
  -H "Content-Type: application/json" \
  -d '{
    "id": "ungm",
    "name": "UN Global Marketplace",
    "url": "https://www.ungm.org/Public/Notice",
    "goal": "Navigate to the procurement notices page, extract all visible open tenders. For each tender extract: title, reference_number, issuing_entity, deadline, estimated_value, source_url. Return as JSON array.",
    "interval_min": 60
  }'
```

### List all portals

```bash
curl -s http://localhost:8080/portals
```

### Delete a portal

```bash
curl -s -X DELETE http://localhost:8080/portals/ungm
```

### List all tenders

```bash
curl -s http://localhost:8080/tenders | python3 -m json.tool
```

### List tenders by portal

```bash
curl -s http://localhost:8080/tenders/ungm | python3 -m json.tool
```

### Trigger deep analysis on a tender

```bash
curl -s -X POST \
  http://localhost:8080/tenders/ungm/UNDP-LBR-00870/analyze
```

Returns 202 immediately. TinyFish fetches the documents, Gemini reads them.
Poll the GET endpoint until the analysis appears.

### Get tender analysis

```bash
curl -s \
  http://localhost:8080/tenders/ungm/UNDP-LBR-00870/analysis \
  | python3 -m json.tool
```

### Connect to the live event stream

```bash
curl -s http://localhost:8080/events
```

Events arrive as:

```json
{"type":"new","tender":{...}}
{"type":"updated","tender":{...}}
```

A heartbeat comment is sent every 30 seconds to survive proxy timeouts.

## Tender Object

```json
{
  "reference_number": "UNDP-LBR-00870",
  "portal_id": "ungm",
  "title": "Procurement of server and UPS for PPCC Liberia",
  "issuing_entity": "UNDP",
  "deadline": "2026-03-21T10:00:00Z",
  "estimated_value": "",
  "source_url": "https://www.ungm.org/Public/Notice/293698",
  "content_hash": "0cf3bb44...",
  "version": 1,
  "last_updated": "2026-03-21T04:57:45Z",
  "status": "new"
}
```

## Analysis Object

```json
{
  "tender_id": "ungm:UNDP-LBR-00870",
  "portal_id": "ungm",
  "summary": "UNDP Liberia requires a vendor to supply a server and UPS unit for the PPCC office. The procurement is a direct supply of ICT hardware.",
  "eligibility_criteria": [
    "Registered company with valid tax compliance certificate",
    "Proven experience supplying ICT hardware to UN or government entities",
    "Valid business registration"
  ],
  "required_documents": [
    "Technical proposal",
    "Company registration certificate",
    "Tax compliance certificate",
    "Itemised price quotation"
  ],
  "evaluation_criteria": [
    "Technical compliance 40%",
    "Price 60%"
  ],
  "estimated_value": "",
  "contact_person": "procurement.lbr@undp.org",
  "qualifies": true,
  "qualify_reasons": [
    "Business profile matches ICT hardware supply",
    "Network infrastructure experience is directly relevant"
  ],
  "draft_response": "Dear UNDP Procurement Team, We write in response to tender UNDP-LBR-00870...",
  "analyzed_at": "2026-03-21T06:30:00Z"
}
```

## Tested Portals

```
UN Global Marketplace   https://www.ungm.org/Public/Notice   15 tenders in ~1 min
Kenya National Treasury https://www.treasury.go.ke/tenders   15 tenders in ~3 min
Hacker News Jobs        https://news.ycombinator.com/jobs     10 items   in ~1 min
```

## Project Structure

```
tinymuscle/
├── agent/       TinyFish SSE client, document fetcher, mock
├── analyzer/    Gemini document intelligence + draft response
├── api/         Chi router, REST handlers, SSE endpoint
├── cmd/         Binary entrypoint, wiring, graceful shutdown
├── extractor/   Shape-agnostic JSON → Tender normalizer
├── matcher/     Gemini relevance scoring
├── notifier/    Fan-out SSE broadcaster, drop semantics for slow clients
├── portals/     Portal type definition
├── scheduler/   Cron engine, immediate-fire on registration, crawl pipeline
└── store/       BBolt, two-key deduplication, version tracking, analyses
```

## Built for the TinyFish $2M Pre-Accelerator Hackathon 2026

