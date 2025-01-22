package exporter

import (
	"fmt"
	"gofire/component"
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

func (e *StdoutExporter) Export(messages ...map[string]interface{}) error {
	start := time.Now()
	e.metrics.AddTotal(e.pipeline, "stdout", float64(len(messages)))
	for _, msg := range messages {
		fmt.Println(msg)
	}
	e.metrics.AddProcessDuration(e.pipeline, "stdout", time.Since(start))
	return nil
}

func (e *StdoutExporter) Shutdown() error {
	return nil
}
