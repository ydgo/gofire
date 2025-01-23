package exporter

import (
	"gofire/component"
	"gofire/event"
	"gofire/metrics"
	"time"
)

func init() {
	component.RegisterExporter("stdout", NewStdoutExporter)
}

type StdoutExporter struct {
	pipeline string
	metrics  *metrics.ExporterBasicMetrics
}

func NewStdoutExporter(pipeName string, config map[string]interface{}, collector *metrics.Collector) (component.Exporter, error) {
	return &StdoutExporter{
		pipeline: pipeName,
		metrics:  collector.ExporterBasicMetrics(),
	}, nil
}

func (e *StdoutExporter) Export(evt *event.Event) error {
	start := time.Now()
	e.metrics.IncTotal(e.pipeline, "stdout")
	e.metrics.AddProcessDuration(e.pipeline, "stdout", time.Since(start))
	return nil
}

func (e *StdoutExporter) Shutdown() error {
	return nil
}
