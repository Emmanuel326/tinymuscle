package api

import (
"context"
"encoding/json"
"fmt"
"net/http"
"time"

"github.com/go-chi/chi/v5"
"github.com/go-chi/chi/v5/middleware"

"github.com/Emmanuel326/tinymuscle/agent"
"github.com/Emmanuel326/tinymuscle/analyzer"
"github.com/Emmanuel326/tinymuscle/notifier"
"github.com/Emmanuel326/tinymuscle/portals"
"github.com/Emmanuel326/tinymuscle/scheduler"
"github.com/Emmanuel326/tinymuscle/store"
)

// Server holds all dependencies
type Server struct {
store     *store.Store
scheduler *scheduler.Scheduler
notifier  *notifier.Notifier
agent     *agent.Agent
analyzer  *analyzer.Analyzer
router    *chi.Mux
}

// New wires up the router and returns a Server
func New(
s *store.Store,
sc *scheduler.Scheduler,
n *notifier.Notifier,
a *agent.Agent,
az *analyzer.Analyzer,
) *Server {
srv := &Server{
store:     s,
scheduler: sc,
notifier:  n,
agent:     a,
analyzer:  az,
router:    chi.NewRouter(),
}
srv.routes()
return srv
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
s.router.ServeHTTP(w, r)
}

func (s *Server) routes() {
r := s.router

r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(cors)

// portals
r.Post("/portals", s.handleCreatePortal)
r.Get("/portals", s.handleListPortals)
r.Delete("/portals/{id}", s.handleDeletePortal)

// tenders
r.Get("/tenders", s.handleListTenders)
r.Get("/tenders/{portalID}", s.handleTendersByPortal)

// analysis
r.Post("/tenders/{portalID}/{referenceNumber}/analyze", s.handleAnalyze)
r.Get("/tenders/{portalID}/{referenceNumber}/analysis", s.handleGetAnalysis)

// SSE stream
r.Get("/events", s.handleEvents)
}

func (s *Server) handleCreatePortal(w http.ResponseWriter, r *http.Request) {
var p portals.Portal
if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
http.Error(w, "invalid payload", http.StatusBadRequest)
return
}

if p.ID == "" || p.URL == "" || p.Goal == "" {
http.Error(w, "id, url and goal are required", http.StatusBadRequest)
return
}

if p.IntervalMin <= 0 {
p.IntervalMin = 60
}

raw, err := json.Marshal(p)
if err != nil {
http.Error(w, "marshal error", http.StatusInternalServerError)
return
}

if err := s.store.SavePortal(raw, p.ID); err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

if err := s.scheduler.Register(p); err != nil {
http.Error(w, "scheduler error", http.StatusInternalServerError)
return
}

w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(p)
}

func (s *Server) handleListPortals(w http.ResponseWriter, r *http.Request) {
raws, err := s.store.GetAllPortals()
if err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

result := make([]portals.Portal, 0, len(raws))
for _, raw := range raws {
var p portals.Portal
if err := json.Unmarshal(raw, &p); err != nil {
continue
}
result = append(result, p)
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(result)
}

func (s *Server) handleDeletePortal(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")
if id == "" {
http.Error(w, "id required", http.StatusBadRequest)
return
}

if err := s.store.DeletePortal(id); err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

s.scheduler.Deregister(id)
w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListTenders(w http.ResponseWriter, r *http.Request) {
tenders, err := s.store.GetAllTenders()
if err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(tenders)
}

func (s *Server) handleTendersByPortal(w http.ResponseWriter, r *http.Request) {
portalID := chi.URLParam(r, "portalID")
tenders, err := s.store.GetTendersByPortal(portalID)
if err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(tenders)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
portalID := chi.URLParam(r, "portalID")
refNum := chi.URLParam(r, "referenceNumber")

tenders, err := s.store.GetTendersByPortal(portalID)
if err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

var target *store.Tender
for i := range tenders {
if tenders[i].ReferenceNumber == refNum {
target = &tenders[i]
break
}
}

if target == nil {
http.Error(w, "tender not found", http.StatusNotFound)
return
}

if target.SourceURL == "" {
http.Error(w, "tender has no source URL", http.StatusBadRequest)
return
}

portalsRaw, err := s.store.GetAllPortals()
if err != nil {
http.Error(w, "store error", http.StatusInternalServerError)
return
}

var businessProfile string
for _, raw := range portalsRaw {
var p portals.Portal
if err := json.Unmarshal(raw, &p); err != nil {
continue
}
if p.ID == portalID {
businessProfile = p.BusinessProfile
break
}
}

go func() {
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

content, err := s.agent.FetchDocument(ctx, target.SourceURL)
if err != nil {
return
}

analysis, err := s.analyzer.Analyze(ctx, *target, content, businessProfile)
if err != nil {
return
}

data, err := json.Marshal(analysis)
if err != nil {
return
}

s.store.SaveAnalysis(portalID+":"+refNum, data)
}()

w.WriteHeader(http.StatusAccepted)
json.NewEncoder(w).Encode(map[string]string{
"message": "analysis started — poll GET /tenders/" + portalID + "/" + refNum + "/analysis",
})
}

func (s *Server) handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
portalID := chi.URLParam(r, "portalID")
refNum := chi.URLParam(r, "referenceNumber")

data, err := s.store.GetAnalysis(portalID + ":" + refNum)
if err != nil {
http.Error(w, "analysis not found — trigger POST first", http.StatusNotFound)
return
}

w.Header().Set("Content-Type", "application/json")
w.Write(data)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("X-Accel-Buffering", "no")

flusher, ok := w.(http.Flusher)
if !ok {
http.Error(w, "streaming unsupported", http.StatusInternalServerError)
return
}

ch := s.notifier.Subscribe()
defer s.notifier.Unsubscribe(ch)

ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
select {
case payload, ok := <-ch:
if !ok {
return
}
fmt.Fprintf(w, "data: %s\n\n", payload)
flusher.Flush()

case <-ticker.C:
fmt.Fprintf(w, ": heartbeat\n\n")
flusher.Flush()

case <-r.Context().Done():
return
}
}
}

func cors(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
if r.Method == http.MethodOptions {
w.WriteHeader(http.StatusNoContent)
return
}
next.ServeHTTP(w, r)
})
}
