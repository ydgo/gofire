package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ConfigMetrics struct {
	Total  prometheus.Gauge
	Reload *prometheus.CounterVec
}
