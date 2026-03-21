package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/Emmanuel326/tinymuscle/agent"
	"github.com/Emmanuel326/tinymuscle/extractor"
	"github.com/Emmanuel326/tinymuscle/matcher"
	"github.com/Emmanuel326/tinymuscle/notifier"
	"github.com/Emmanuel326/tinymuscle/portals"
	"github.com/Emmanuel326/tinymuscle/store"
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
	matcher  *matcher.Matcher
	mu       sync.Mutex
	jobs     map[string]cron.EntryID
}

// New creates a new Scheduler
func New(
	r Runner,
	s *store.Store,
	n *notifier.Notifier,
	m *matcher.Matcher,
) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		runner:   r,
		store:    s,
		notifier: n,
		matcher:  m,
		jobs:     make(map[string]cron.EntryID),
	}
}

// Start begins the cron engine and rehydrates portals from BBolt
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

// Register adds a new portal to the cron engine.
// It also fires an immediate crawl in a goroutine so
// the first result appears without waiting for the first tick.
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

	// fire immediately on registration
	go s.crawl(p)

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

// crawl is the full pipeline for a single portal:
// Agent → Extractor → Matcher → Store → Notifier
func (s *Scheduler) crawl(p portals.Portal) {
	log.Printf("[%s] crawl started", p.ID)

	ctx, cancel := context.WithTimeout(context.Background(), agent.CrawlTimeout)
	defer cancel()

	result := s.runner.Run(ctx, p, func(event agent.SSEEvent) {
		if event.Type == "STREAMING_URL" {
			log.Printf("[%s] live browser: %s", p.ID, event.Result)
		} else {
			log.Printf("[%s] sse event: %s", p.ID, event.Type)
		}
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

	// run AI matcher if business profile is set
	if s.matcher != nil && p.BusinessProfile != "" {
		threshold := p.RelevanceThreshold
		if threshold == 0 {
			threshold = 60
		}

		scored, err := s.matcher.Score(ctx, p.BusinessProfile, tenders, threshold)
		if err != nil {
			log.Printf("[%s] matcher failed: %v — using all tenders", p.ID, err)
		} else {
			log.Printf("[%s] matcher kept %d/%d relevant tenders", p.ID, len(scored), len(tenders))
			tenders = make([]store.Tender, len(scored))
			for i, sc := range scored {
				tenders[i] = sc.Tender
			}
		}
	}

	events, err := s.store.UpsertTenders(tenders)
	if err != nil {
		log.Printf("[%s] upsert failed: %v", p.ID, err)
		return
	}

	log.Printf("[%s] %d new/updated tenders", p.ID, len(events))
	s.notifier.Broadcast(events)
}
