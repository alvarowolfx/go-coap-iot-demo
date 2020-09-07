package metrics

import (
	"log"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
)

func StartMetricsExporter() {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "iot_demo",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	// Now finally run the Prometheus exporter as a scrape endpoint.
	// We'll run the server on port 8888.
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(":8888", mux); err != nil {
			log.Fatalf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()
}
