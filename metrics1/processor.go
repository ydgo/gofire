package metrics1

import "github.com/prometheus/client_golang/prometheus"

type ProcessorCollector interface {
	IncProcessed(labels prometheus.Labels)
	IncProcessedFailures(labels prometheus.Labels)
	RegisterCustomMetrics(CustomMetric)
}

// CustomMetric 定义自定义指标
type CustomMetric struct {
	Name   string
	Value  float64
	Type   prometheus.ValueType
	Labels map[string]string
}

type processorCollector struct {
	total    *prometheus.CounterVec
	failures *prometheus.CounterVec
}

func NewProcessorCollector() ProcessorCollector {
	collector := &processorCollector{
		total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "processor",
			Name:      "total",
			Subsystem: "",
			Help:      "Total number of processed metrics",
		}, []string{"pipeline", "type"}),
		failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "processor",
			Name:      "failures",
			Help:      "Total number of failed metrics",
		}, []string{"pipeline", "type"}),
	}
	return collector
}

func (collector *processorCollector) IncProcessed(labels prometheus.Labels) {
	collector.total.With(labels).Inc()
}

func (collector *processorCollector) IncProcessedFailures(labels prometheus.Labels) {
	collector.failures.With(labels).Inc()
}

func (collector *processorCollector) RegisterCustomMetrics(customMetric CustomMetric) {}
