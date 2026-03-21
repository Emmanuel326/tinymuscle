package scheduler

import (
"context"
"encoding/json"
"fmt"
"log"
"sync"

"github.com/robfig/cron/v3"

"github.com/Emmanuel326/tenderwatchafrica/agent"
"github.com/Emmanuel326/tenderwatchafrica/extractor"
"github.com/Emmanuel326/tenderwatchafrica/notifier"
"github.com/Emmanuel326/tenderwatchafrica/portals"
"github.com/Emmanuel326/tenderwatchafrica/store"
)

// Runner is the interface both Agent and MockAgent satisfy
type Runner interface {
Run(ctx context.Context, portal portals.Portal, onEvent func(agent.SSEEvent)) agent.Result
}

// Scheduler manages cron jobs per portal
type Scheduler struct {
cron     *cron.Cron
runner   Runner
store    *store.Store
notifier *notifier.Notifier
mu       sync.Mutex
jobs     map[string]cron.EntryID
}

// New creates a new Scheduler
func New(
r Runner,
s *store.Store,
n *notifier.Notifier,
) *Scheduler {
return &Scheduler{
cron:     cron.New(),
runner:   r,
store:    s,
notifier: n,
jobs:     make(map[string]cron.EntryID),
}
}

// Start begins the cron engine and rehydrates
// all portals already stored in BBolt
func (s *Scheduler) Start() error {
portalsRaw, err := s.store.GetAllPortals()
if err != nil {
return fmt.Errorf("load portals: %w", err)
}

for _, raw := range portalsRaw {
var p portals.Portal
if err := json.Unmarshal(raw, &p); err != nil {
log.Printf("skip malformed portal: %v", err)
continue
}
if err := s.Register(p); err != nil {
log.Printf("skip portal %s: %v", p.ID, err)
}
}

s.cron.Start()
return nil
}

// Stop halts the cron engine gracefully
func (s *Scheduler) Stop() {
s.cron.Stop()
}

// Register adds a new portal to the cron engine
func (s *Scheduler) Register(p portals.Portal) error {
spec := fmt.Sprintf("@every %dm", p.IntervalMin)

s.mu.Lock()
defer s.mu.Unlock()

if id, ok := s.jobs[p.ID]; ok {
s.cron.Remove(id)
}

id, err := s.cron.AddFunc(spec, func() {
s.crawl(p)
})
if err != nil {
return fmt.Errorf("add cron job for %s: %w", p.ID, err)
}

s.jobs[p.ID] = id
return nil
}

// Deregister removes a portal's cron job
func (s *Scheduler) Deregister(portalID string) {
s.mu.Lock()
defer s.mu.Unlock()

if id, ok := s.jobs[portalID]; ok {
s.cron.Remove(id)
delete(s.jobs, portalID)
}
}

// crawl is the full pipeline for a single portal
func (s *Scheduler) crawl(p portals.Portal) {
log.Printf("[%s] crawl started", p.ID)

ctx, cancel := context.WithTimeout(context.Background(), agent.CrawlTimeout)
defer cancel()

result := s.runner.Run(ctx, p, func(event agent.SSEEvent) {
log.Printf("[%s] sse event: %s", p.ID, event.Type)
})

if result.Err != nil {
log.Printf("[%s] crawl failed: %v", p.ID, result.Err)
return
}

tenders, err := extractor.Extract(p.ID, result.Raw)
if err != nil {
log.Printf("[%s] extract failed: %v", p.ID, err)
return
}

log.Printf("[%s] extracted %d tenders", p.ID, len(tenders))

events, err := s.store.UpsertTenders(tenders)
if err != nil {
log.Printf("[%s] upsert failed: %v", p.ID, err)
return
}

log.Printf("[%s] %d new/updated tenders", p.ID, len(events))

s.notifier.Broadcast(events)
}
