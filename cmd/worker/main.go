package main

import (
	"log"
	"os"
	"strings"

	tactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/yourorg/zone-names/internal/activities"
	znmetrics "github.com/yourorg/zone-names/internal/metrics"
	"github.com/yourorg/zone-names/internal/workflow"
)

func main() {
	// Support both TEMPORAL_TARGET_HOST and TEMPORAL_ADDRESS for compatibility
	taddr := getenv("TEMPORAL_TARGET_HOST", getenv("TEMPORAL_ADDRESS", "localhost:7233"))
	ns := getenv("TEMPORAL_NAMESPACE", "default")
	q := getenv("TEMPORAL_TASK_QUEUE", "zone-names")
	tmpDir := getenv("ZN_TMP_DIR", "/var/zone-names")
	// Ensure scratch dir exists and is writable
	_ = os.MkdirAll(tmpDir, 0o777)

	// Structured logger (zap)
	zl := newZap(getenv("LOG_LEVEL", "info"))
	defer zl.Sync()

	// Metrics server
	znmetrics.Init()
	go func() {
		addr := znmetrics.AddrFromEnv()
		_ = znmetrics.Serve(addr)
	}()

	c, err := client.Dial(client.Options{HostPort: taddr, Namespace: ns})
	if err != nil {
		log.Fatal("temporal client:", err)
	}
	defer c.Close()

	w := worker.New(c, q, worker.Options{})
	acts := activities.New(activities.Config{ScratchDir: tmpDir})
	// Register activities with explicit names matching workflow.ExecuteActivity calls
	w.RegisterActivityWithOptions(acts.StreamPartition, tactivity.RegisterOptions{Name: "Activities.StreamPartition"})
	w.RegisterActivityWithOptions(acts.ShardDedupeBadger, tactivity.RegisterOptions{Name: "Activities.ShardDedupeBadger"})
	w.RegisterActivityWithOptions(acts.MergeSortedAndWriteManifest, tactivity.RegisterOptions{Name: "Activities.MergeSortedAndWriteManifest"})
	w.RegisterActivityWithOptions(acts.CleanupScratch, tactivity.RegisterOptions{Name: "Activities.CleanupScratch"})
	w.RegisterWorkflow(workflow.ZoneNamesWorkflow)

	zl.Info("worker started", zap.String("namespace", ns), zap.String("taskQueue", q), zap.String("tmp", tmpDir), zap.String("metrics", getenv("METRICS_ADDR", ":9090")))
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatal("worker failed:", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func newZap(level string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	switch strings.ToLower(level) {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return logger
}
