package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PipelineMetrics struct {
	In        *prometheus.CounterVec
	Processed *prometheus.CounterVec
	Out       *prometheus.CounterVec
	Failures  *prometheus.CounterVec
}

func NewPipelineMetrics() *PipelineMetrics {
	return &PipelineMetrics{
		In: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "pipeline",
			Name:      "in",
			Help:      "数据流入总量",
		}, []string{"pipeline"}),
		Processed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "pipeline",
			Name:      "processed",
			Help:      "数据处理总量",
		}, []string{"pipeline"}),
		Out: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "pipeline",
			Name:      "out",
			Help:      "数据流出总量",
		}, []string{"pipeline"}),
		Failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "pipeline",
			Name:      "failures",
			Help:      "数据失败总量",
		}, []string{"pipeline"}),
	}
}
func (m *PipelineMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.In.Describe(ch)
	m.Processed.Describe(ch)
	m.Out.Describe(ch)
	m.Failures.Describe(ch)
}

func (m *PipelineMetrics) Collect(ch chan<- prometheus.Metric) {
	m.In.Collect(ch)
	m.Processed.Collect(ch)
	m.Out.Collect(ch)
	m.Failures.Collect(ch)
}

func (m *PipelineMetrics) AddIn(pipelineName string, n float64) {
	m.In.WithLabelValues(pipelineName).Add(n)
}

func (m *PipelineMetrics) AddProcessed(pipelineName string, n float64) {
	m.Processed.WithLabelValues(pipelineName).Add(n)
}

func (m *PipelineMetrics) AddOut(pipelineName string, n float64) {
	m.Out.WithLabelValues(pipelineName).Add(n)
}

func (m *PipelineMetrics) AddFailure(pipelineName string, n float64) {
	m.Failures.WithLabelValues(pipelineName).Add(n)
}

func (m *PipelineMetrics) Delete(pipelineName string) {
	m.In.DeleteLabelValues(pipelineName)
	m.Processed.DeleteLabelValues(pipelineName)
	m.Out.DeleteLabelValues(pipelineName)
	m.Failures.DeleteLabelValues(pipelineName)
}
