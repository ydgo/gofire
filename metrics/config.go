package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ConfigMetrics struct {
	Total  prometheus.Gauge
	Reload *prometheus.CounterVec
}

func NewConfigMetrics() *ConfigMetrics {
	return &ConfigMetrics{
		Total: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "gofire",
			Subsystem: "config",
			Name:      "total",
			Help:      "Config 总数",
		}),
		Reload: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "config",
			Name:      "reload",
			Help:      "Config 重载总次数",
		}, []string{"name"}),
	}
}
func (c *ConfigMetrics) Describe(ch chan<- *prometheus.Desc) {
	c.Total.Describe(ch)
	c.Reload.Describe(ch)
}

func (c *ConfigMetrics) Collect(ch chan<- prometheus.Metric) {
	c.Total.Collect(ch)
	c.Reload.Collect(ch)
}
