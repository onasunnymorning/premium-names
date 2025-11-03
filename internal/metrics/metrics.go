package metrics

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RecordsPartitioned = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "zone_names",
		Name:      "records_partitioned_total",
		Help:      "Total DNS records seen during partitioning.",
	})
	DedupeInput = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "zone_names",
		Name:      "dedupe_input_total",
		Help:      "Total names processed in dedupe.",
	})
	DedupeUnique = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "zone_names",
		Name:      "dedupe_unique_total",
		Help:      "Total unique names emitted by dedupe.",
	})
	MergedEmitted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "zone_names",
		Name:      "merged_emitted_total",
		Help:      "Total unique names emitted by merge.",
	})
)

// Init registers collectors; call once from main.
func Init() {
	prometheus.MustRegister(RecordsPartitioned, DedupeInput, DedupeUnique, MergedEmitted)
}

// Serve starts a /metrics server on the given addr (e.g., ":9090"). Non-blocking when run in goroutine.
func Serve(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}

// AddrFromEnv returns listen address from METRICS_ADDR or default ":9090".
func AddrFromEnv() string {
	if v := os.Getenv("METRICS_ADDR"); v != "" { return v }
	return ":9090"
}
