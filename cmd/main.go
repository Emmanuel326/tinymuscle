package main

import (
"context"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/Emmanuel326/tenderwatchafrica/agent"
"github.com/Emmanuel326/tenderwatchafrica/api"
"github.com/Emmanuel326/tenderwatchafrica/notifier"
"github.com/Emmanuel326/tenderwatchafrica/scheduler"
"github.com/Emmanuel326/tenderwatchafrica/store"
)

func main() {
apiKey := os.Getenv("TINYFISH_API_KEY")
useMock := os.Getenv("USE_MOCK") == "true"

if !useMock && apiKey == "" {
log.Fatal("TINYFISH_API_KEY is required unless USE_MOCK=true")
}

dbPath := os.Getenv("DB_PATH")
if dbPath == "" {
dbPath = "tenderwatch.db"
}

addr := os.Getenv("ADDR")
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

// scheduler — accepts either real or mock agent
var sc *scheduler.Scheduler
if useMock {
log.Println("running with mock agent")
sc = scheduler.New(agent.NewMock(), s, n)
} else {
sc = scheduler.New(agent.New(apiKey), s, n)
}

if err := sc.Start(); err != nil {
log.Fatalf("scheduler: %v", err)
}
defer sc.Stop()

// api
srv := &http.Server{
Addr:         addr,
Handler:      api.New(s, sc, n),
ReadTimeout:  10 * time.Second,
WriteTimeout: 0,
IdleTimeout:  60 * time.Second,
}

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("TenderWatchAfrica listening on %s", addr)
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
