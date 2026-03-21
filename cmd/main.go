package main

import (
"context"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/Emmanuel326/tinymuscle/agent"
"github.com/Emmanuel326/tinymuscle/analyzer"
"github.com/Emmanuel326/tinymuscle/api"
"github.com/Emmanuel326/tinymuscle/matcher"
"github.com/Emmanuel326/tinymuscle/notifier"
"github.com/Emmanuel326/tinymuscle/scheduler"
"github.com/Emmanuel326/tinymuscle/store"
)

func main() {
apiKey := os.Getenv("TINYFISH_API_KEY")
useMock := os.Getenv("USE_MOCK") == "true"
geminiKey := os.Getenv("GEMINI_API_KEY")
dbPath := os.Getenv("DB_PATH")
addr := os.Getenv("ADDR")

if !useMock && apiKey == "" {
log.Fatal("TINYFISH_API_KEY is required unless USE_MOCK=true")
}
if dbPath == "" {
dbPath = "tinymuscle.db"
}
if addr == "" {
addr = ":8080"
}

// store
s, err := store.New(dbPath)
if err != nil {
log.Fatalf("store: %v", err)
}
defer s.Close()

// notifier
n := notifier.New()

// matcher
var m *matcher.Matcher
if geminiKey != "" {
m, err = matcher.New(geminiKey)
if err != nil {
log.Fatalf("matcher: %v", err)
}
log.Println("AI matcher enabled")
} else {
log.Println("AI matcher disabled — set GEMINI_API_KEY to enable")
}

// analyzer
var az *analyzer.Analyzer
if geminiKey != "" {
az, err = analyzer.New(geminiKey)
if err != nil {
log.Fatalf("analyzer: %v", err)
}
log.Println("AI analyzer enabled")
}

// agent
var a *agent.Agent
var sc *scheduler.Scheduler
if useMock {
log.Println("running with mock agent")
sc = scheduler.New(agent.NewMock(), s, n, m)
a = nil
} else {
a = agent.New(apiKey)
sc = scheduler.New(a, s, n, m)
}

if err := sc.Start(); err != nil {
log.Fatalf("scheduler: %v", err)
}
defer sc.Stop()

// api
srv := &http.Server{
Addr:         addr,
Handler:      api.New(s, sc, n, a, az),
ReadTimeout:  10 * time.Second,
WriteTimeout: 0,
IdleTimeout:  60 * time.Second,
}

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("TinyMuscle listening on %s", addr)
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("server: %v", err)
}
}()

<-quit
log.Println("shutting down...")

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
log.Fatalf("shutdown: %v", err)
}

log.Println("clean exit")
}
