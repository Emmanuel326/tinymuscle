# TenderWatchAfrica

An AI agent that hunts government tenders and contracts across African procurement portals — so businesses never miss an opportunity because some broken government website didn't load that day.

Powered by [TinyFish](https://www.tinyfish.ai).

---

## 🚀 How It Works

1. You register a portal (URL + crawl goal) via the API  
2. The scheduler sends a TinyFish agent to crawl it on your interval  
3. The agent navigates the live web — pagination, JavaScript, anti-bot — and extracts tenders  
4. New and updated tenders are stored, deduplicated, and streamed live to the dashboard  

---

## 🧱 Stack

- **Go 1.26** — single binary, zero runtime dependencies  
- **TinyFish Web Agent API** — browser automation and anti-bot  
- **BBolt** — embedded key/value store, no database server needed  
- **Chi** — lightweight HTTP router  
- **Server-Sent Events (SSE)** — live tender feed to the frontend  

---

## 🛠️ Running Locally

```bash
# clone
git clone https://github.com/Emmanuel326/tenderwatchafrica
cd tenderwatchafrica

# configure
cp .env.example .env
# edit .env and set TINYFISH_API_KEY

# build
go build -o tenderwatchafrica ./cmd/main.go

# run
export $(cat .env | xargs) && ./tenderwatchafrica
```

---

## 🔐 Environment Variables

| Variable           | Required | Default           | Description                          |
|------------------|----------|------------------|--------------------------------------|
| TINYFISH_API_KEY | yes*     | —                | TinyFish API key                     |
| DB_PATH          | no       | tenderwatch.db   | BBolt database file path             |
| ADDR             | no       | :8080            | HTTP server address                  |
| USE_MOCK         | no       | false            | Use mock agent for local dev         |

\* Not required when `USE_MOCK=true`

---

## 📡 API Reference

### 📂 Portals

| Method | Endpoint      | Description           |
|--------|--------------|-----------------------|
| POST   | /portals     | Register a new portal |
| GET    | /portals     | List all portals      |
| DELETE | /portals/:id | Remove a portal       |

#### Example Request (POST /portals)

```json
{
  "id": "ppra_ke",
  "name": "PPRA Kenya",
  "url": "https://www.ppra.go.ke",
  "goal": "Navigate to the tenders section, paginate through all pages, extract all open tenders as JSON array with fields: title, reference_number, issuing_entity, deadline, estimated_value, source_url",
  "interval_min": 60,
  "headers": {},
  "cookies": {}
}
```

---

### 📑 Tenders

| Method | Endpoint            | Description                    |
|--------|--------------------|--------------------------------|
| GET    | /tenders           | All tenders across all portals |
| GET    | /tenders/:portalID | Tenders for a specific portal  |

#### Tender Object

```json
{
  "reference_number": "PPRA/001/2026",
  "portal_id": "ppra_ke",
  "title": "Supply of Office Furniture",
  "issuing_entity": "Ministry of Public Service",
  "deadline": "2026-04-03T00:00:00Z",
  "estimated_value": "KES 2,500,000",
  "source_url": "https://www.ppra.go.ke/tenders/001",
  "content_hash": "0cf3bb44...",
  "version": 1,
  "last_updated": "2026-03-20T23:46:39Z",
  "status": "new"
}
```

---

## 🔴 Live Events (SSE)

**Endpoint:** `GET /events`

- Connect once and receive a live stream of tender events  
- Heartbeat sent every 30 seconds  

#### Event Example

```json
{
  "type": "new",
  "tender": { ...tender object }
}
```

#### Event Types

- `new` — First time seen  
- `updated` — Content changed  
- `closed` — Past deadline  

---

## 📊 Tender Status

| Status  | Meaning                           |
|--------|-----------------------------------|
| new    | First time this tender is seen     |
| updated| Tender exists but content changed |
| closed | Tender past deadline              |

---

## 🌍 Demo Portals

| Portal                    | ID              | Interval |
|--------------------------|-----------------|----------|
| PPRA Kenya               | ppra_ke         | 60 min   |
| KeNHA                    | kenha           | 60 min   |
| Nairobi City County      | nairobi_county  | 120 min  |
| African Development Bank | afdb            | 120 min  |
| UN Kenya                 | un_kenya        | 120 min  |

---

## 🗂️ Project Structure

```
tenderwatchafrica/
├── agent/        # TinyFish SSE client + mock
├── api/          # Chi HTTP server + SSE endpoint
├── cmd/          # Binary entrypoint
├── extractor/    # Raw JSON → Tender structs
├── notifier/     # Fan-out SSE broadcaster
├── portals/      # Portal data type
├── scheduler/    # Cron engine + crawl pipeline
└── store/        # BBolt persistence + deduplication
```

---

## 🏁 Hackathon

Built for **TinyFish $2M Pre-Accelerator Hackathon 2026**
