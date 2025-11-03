package main

import (
	"log"
	"os"
	"strings"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/yourorg/zone-names/internal/activities"
	znmetrics "github.com/yourorg/zone-names/internal/metrics"
	"github.com/yourorg/zone-names/internal/workflow"
)

func main() {
	taddr := getenv("TEMPORAL_TARGET_HOST", "localhost:7233")
	ns := getenv("TEMPORAL_NAMESPACE", "default")
	q := getenv("TEMPORAL_TASK_QUEUE", "zone-names")
	tmpDir := getenv("ZN_TMP_DIR", "/tmp/zone-names")

	// Structured logger (zap)
	zl := newZap(getenv("LOG_LEVEL", "info"))
	defer zl.Sync()

	// Metrics server
	znmetrics.Init()
	go func() {
		addr := znmetrics.AddrFromEnv()
		_ = znmetrics.Serve(addr)
	}()

	c, err := client.Dial(client.Options{ HostPort: taddr, Namespace: ns })
	if err != nil { log.Fatal("temporal client:", err) }
	defer c.Close()

	w := worker.New(c, q, worker.Options{})
	acts := activities.New(activities.Config{ ScratchDir: tmpDir })
	w.RegisterActivity(acts.StreamPartition)
	w.RegisterActivity(acts.ShardDedupeBadger)
	w.RegisterActivity(acts.MergeSortedAndWriteManifest)
	w.RegisterWorkflow(workflow.ZoneNamesWorkflow)

	zl.Info("worker started", zap.String("namespace", ns), zap.String("taskQueue", q), zap.String("tmp", tmpDir), zap.String("metrics", getenv("METRICS_ADDR", ":9090")))
	if err := w.Run(worker.InterruptCh()); err != nil { log.Fatal("worker failed:", err) }
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

func newZap(level string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	switch strings.ToLower(level) {
	case "debug": cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info": cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn": cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error": cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default: cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := cfg.Build()
	if err != nil { return zap.NewNop() }
	return logger
}
