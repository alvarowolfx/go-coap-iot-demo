package coap

import (
	"log"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	MLatencyMs = stats.Float64("gateway/coap/latency", "The latency in milliseconds per request", "ms")

	MRequests = stats.Int64("gateway/coap/requests", "Number of requests", "By")

	MMessageBytes = stats.Int64("gateway/coap/bytes", "Number of bytes received", "bytes")
)

var (
	LatencyView = &view.View{
		Name:        "gateway/coap/latency",
		Measure:     MLatencyMs,
		Description: "The distribution of the latencies",

		Aggregation: view.Distribution(0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000),
		TagKeys:     []tag.Key{KeyMethod},
	}

	RequestsCountView = &view.View{
		Name:        "gateway/coap/requests",
		Measure:     MRequests,
		Description: "Number of requests",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyMethod, KeyFormat},
	}

	MessageSizeView = &view.View{
		Name:        "gateway/coap/bytes",
		Measure:     MMessageBytes,
		Description: "Bytes received",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyMethod, KeyFormat},
	}
)

var (
	KeyMethod, _ = tag.NewKey("method")
	KeyStatus, _ = tag.NewKey("status")
	KeyFormat, _ = tag.NewKey("format")
	KeyError, _  = tag.NewKey("error")
)

func registerMetrics() {
	err := view.Register(LatencyView, RequestsCountView, MessageSizeView)
	if err != nil {
		log.Fatalf("Failed to register views: %v", err)
	}
}

func sinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}
