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

## Features

-**AI-Powered Extraction** - Natural language goals, not brittle CSS selectors -**Smart Relevance Filtering** - Gemini AI scores tenders against your business profile -**Intelligent Deduplication** - Two-key model detects real changes, ignores noise -**Real-time Updates** - SSE streams tenders live to your dashboard -**Zero Configuration Database** - Embedded BBolt, no separate database server

- **Single Binary** - Everything in one executable, easy deployment
- **Modern Dashboard** - Beautiful React frontend with real-time updates
- **Live Web Agents** - TinyFish handles JavaScript, pagination, anti-bot measures

## The Architecture

TinyMuscle makes one architectural bet: delegate all browser complexity to
TinyFish and own everything else.

```
TinyFish (browser agent)
    ↓  SSE stream — partial results committed in real time
Extractor (raw JSON → Tender structs)
    ↓  shape-agnostic — handles flat arrays and nested objects
Gemini Flash (relevance scoring against business profile)
    ↓  single LLM call per batch — not per tender
BBolt (two-key deduplication)
    ↓  primary key: portalID + referenceNumber
    ↓  version key: SHA256 of content fields
SSE broadcast → Dashboard
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
tender batch is scored by Gemini Flash against that profile in a single
API call. Only tenders above `relevance_threshold` (default: 60/100) are
stored and surfaced. When no `business_profile` is provided, all tenders
are kept. The system degrades gracefully rather than failing.

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

## What This Is Good At

- Portals with aggressive anti-bot measures
- Sites that require multi-step navigation to reach listings
- Paginated results across an unknown number of pages
- Any situation where the cost of a missed opportunity exceeds the cost of running the pipeline

## What This Is Not Good At

- Real-time data with sub-minute latency — TinyFish sessions take 1-3 minutes per portal
- Portals requiring authenticated sessions with MFA
- Extracting data from PDFs linked within tender listings

## Quick Start

### Prerequisites

- **Go 1.25+** - [Download](https://golang.org/dl/)
- **Node.js 18+** - [Download](https://nodejs.org/)
- **TinyFish API Key** - Get one at [tinyfish.io](https://tinyfish.io) (required for live agents)
- **Gemini API Key** - Get one at [makersuite.google.com/app/apikey](https://makersuite.google.com/app/apikey) (optional, for AI relevance scoring)

### Installation

```bash
# Clone the repository
git clone https://github.com/Emmanuel326/tinymuscle
cd tinymuscle

# Configure environment
cp .env.example .env
# Edit .env and add your API keys

# Build the backend
go build -o tinymuscle ./cmd/main.go

# Install frontend dependencies
cd tenderwatch-frontend
npm install

# Return to project root
cd ..
```

### Environment Variables

```bash
TINYFISH_API_KEY=required       # TinyFish Web Agent API key
GEMINI_API_KEY=optional         # Gemini API key for relevance scoring
DB_PATH=tinymuscle.db           # BBolt database path
ADDR=:8080                      # HTTP listen address
USE_MOCK=false                  # Set to true to bypass TinyFish for local dev
```

**Note:** `TINYFISH_API_KEY` is not required when `USE_MOCK=true`

### Running the Application

```bash
# Terminal 1: Start the backend
export $(cat .env | xargs)
./tinymuscle

# Terminal 2: Start the frontend development server
cd tenderwatch-frontend
npm run dev
```

Access the dashboard at `http://localhost:5173`

## API Reference

### Register a Portal

Enable AI relevance scoring by providing a business profile:

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

### Register a Portal Without AI Filtering

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

### List All Portals

```bash
curl -s http://localhost:8080/portals
```

### Delete a Portal

```bash
curl -s -X DELETE http://localhost:8080/portals/ungm
```

### List All Tenders

```bash
curl -s http://localhost:8080/tenders | python3 -m json.tool
```

### List Tenders by Portal

```bash
curl -s http://localhost:8080/tenders/ungm | python3 -m json.tool
```

### Subscribe to Live Events (Server-Sent Events)

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

Status values: `new` | `updated`

## Tested Portals

```
UN Global Marketplace   https://www.ungm.org/Public/Notice   15 tenders in ~1 min
Kenya National Treasury https://www.treasury.go.ke/tenders   15 tenders in ~3 min
Hacker News Jobs        https://news.ycombinator.com/jobs     10 items   in ~1 min
```

## Project Structure

```
tinymuscle/
├── agent/                   # TinyFish SSE client + mock
│   ├── agent.go            # Real TinyFish agent
│   └── mock.go             # Mock agent for testing
├── api/                     # Chi router, REST handlers
│   └── api.go              # Server setup
├── cmd/                     # Binary entrypoint
│   └── main.go             # App wiring, graceful shutdown
├── extractor/               # Shape-agnostic JSON → Tender
│   └── extractor.go        # Field mapping & validation
├── matcher/                 # Gemini relevance scoring
│   └── matcher.go          # AI scoring integration
├── notifier/                # Fan-out SSE broadcaster
│   └── notifier.go         # Event distribution
├── portals/                 # Portal type definition
│   └── portals.go          # Portal struct
├── scheduler/               # Cron engine, crawl pipeline
│   └── scheduler.go        # Scheduling & execution
├── store/                   # BBolt, two-key deduplication
│   └── store.go            # Database operations
├── tenderwatch-frontend/    # React + Vite dashboard
│   ├── src/
│   │   ├── components/     # TenderCard, PortalManager
│   │   ├── hooks/          # useSSE for real-time events
│   │   ├── services/       # API client
│   │   ├── App.jsx         # Main dashboard
│   │   ├── main.jsx        # Entry point
│   │   ├── index.css       # Tailwind styles
│   │   └── assets/         # Static files
│   ├── package.json        # Frontend dependencies
│   ├── tailwind.config.js  # Tailwind CSS config
│   ├── vite.config.js      # Vite build config
│   └── index.html          # HTML template
├── .env.example             # Environment template
├── go.mod                   # Go dependencies
├── go.sum                   # Go checksums
├── add_test_data.go         # Utility to seed test data
└── README.md                # This file
```

## Dashboard Features

The **TenderWatch** frontend is a modern React application with real-time updates:

### Dashboard Home

- **Live Statistics** - Total tenders, new opportunities, and updated tenders at a glance
- **Real-time Notifications** - Toast alerts for new and updated tenders
- **Live Feed Status** - Visual indicator showing connection status to the SSE event stream

### Search & Filter

- **Full-text Search** - Search by tender title, reference number, or issuing entity
- **Status Filtering** - Filter by All / New / Updated tenders
- **Live Mode Toggle** - Switch between viewing all tenders or just the latest 20

### Portal Management

- **Add Portals** - Register new procurement portals with a clean form interface
- **AI Relevance Filtering** - (Optional) Configure business profile and relevance threshold
- **Portal List** - View active portals with crawl intervals and filtering status
- **Delete Portals** - Remove portals you no longer need

### Tender Viewing

- **Tender Cards** - Rich cards displaying:
  - Title and Issuing Entity
  - Deadline with visual warnings for approaching deadlines
  - Estimated Value (if available)
  - Relevance Score (if AI filtering enabled)
  - Direct link to source
  - Version and update timestamp
- **Status Badges** - Visual indicators for new (🆕) and updated (📝) tenders

### Front-End Tech Stack

- **React 18** - Modern UI framework
- **Vite** - Lightning-fast build tool
- **Tailwind CSS** - Utility-first styling
- **Lucide React** - Beautiful SVG icons
- **react-hot-toast** - Non-intrusive notifications
- **date-fns** - Date formatting and manipulation
- **Axios** - HTTP client for API calls
- **Server-Sent Events (SSE)** - Real-time bidirectional updates

## Built for the TinyFish $2M Pre-Accelerator Hackathon 2026

## Development

### Local Development with Mock Data

To test without a TinyFish API key, use mock mode:

```bash
# Set USE_MOCK=true in your .env
export USE_MOCK=true
export $(cat .env | xargs)./tinymuscle
```

The mock agent returns sample tender data for testing portal registration and the dashboard.

### Adding Test Data

Add sample tenders to the database:

```bash
# Run the add_test_data script
go run add_test_data.go
```

This populates the BBolt database with test portals and tenders for frontend development.

### Frontend Development

The frontend runs with hot module reloading (HMR):

```bash
cd tenderwatch-frontend
npm run dev
```

Access the dashboard at `http://localhost:5173`. Changes to React components reload instantly.

### Building for Production

**Backend:**

```bash
go build -o tinymuscle ./cmd/main.go
```

## Troubleshooting

### Backend Issues

**"TINYFISH_API_KEY is required"**

- Set `USE_MOCK=true` if you don't have a TinyFish key, or get one at [tinyfish.io](https://tinyfish.io)

**"Failed to connect to event stream"**

- Ensure the backend is running on port 8080
- Check firewall rules are not blocking localhost connections
- Verify `ADDR` environment variable is set correctly

**"Database locked"**

- BBolt can only have one writer at a time. Ensure only one instance of tinymuscle is running.
- Check `DB_PATH` environment variable — multiple instances might be using different database files.

### Frontend Issues

**"Failed to load tenders" or "Disconnected" status**

- Verify the backend is running: `curl http://localhost:8080/portals`
- Check that `ADDR=:8080` matches the backend server address
- If running on a different machine, update the API URL in `tenderwatch-frontend/src/services/api.js`

**SSE connection keeps failing**

- Check browser console for CORS errors
- Verify backend is listening on 8080: `lsof -i :8080` or `netstat -an | grep 8080`
- Ensure no firewall/proxy is blocking SSE connections

**Node modules issues**

- Clear and reinstall: `rm -rf node_modules package-lock.json && npm install`

## Performance Notes

- **Crawl Time**: TinyFish sessions typically take 1-3 minutes per portal depending on site complexity
- **Database** - BBolt is single-file and has no server overhead, but is optimized for batch writes and read-on-request patterns
- **Relevance Scoring**: Gemini API calls are batched per crawl cycle, not per tender (single call for all new tenders)
- **Memory**: Typical runtime uses <100MB for the backend with 10K+ tenders in the database

## Contributing

TinyMuscle is designed for extensibility:

- **New Portals** - Register via API, no code changes needed
- **Custom Extractors** - Modify `extractor/extractor.go` for new field patterns
- **Custom Matchers** - Beyond Gemini, implement the `Matcher` interface in `matcher/matcher.go`
- **Frontend Customization** - All React components in `tenderwatch-frontend/src/components/`

## Support

- **Issues & Features** - GitHub Issues
- **Questions** - Start a Discussion
- **TinyFish Docs** - [docs.tinyfish.io](https://docs.tinyfish.io)
- **Gemini API Docs** - [ai.google.dev](https://ai.google.dev)
