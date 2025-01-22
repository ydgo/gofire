package metrics

import "github.com/prometheus/client_golang/prometheus"

type PipelineManagerMetrics struct {
	PipelineTotal prometheus.Gauge
	// todo pipeline status vector
}

func NewPipelineManagerMetrics() *PipelineManagerMetrics {
	return &PipelineManagerMetrics{
		PipelineTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "gofire_pipeline_total",
			Help: "Total number of pipelines.",
		}),
	}
}

func (m *PipelineManagerMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.PipelineTotal.Describe(ch)
}

func (m *PipelineManagerMetrics) Collect(ch chan<- prometheus.Metric) {
	m.PipelineTotal.Collect(ch)
}

func (m *PipelineManagerMetrics) IncPipelineTotal() {
	m.PipelineTotal.Inc()
}

func (m *PipelineManagerMetrics) DecPipelineTotal() {
	m.PipelineTotal.Dec()
}
