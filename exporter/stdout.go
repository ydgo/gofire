package exporter

import (
	"fmt"
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
	if data, err := evt.ToJSON(); err == nil {
		fmt.Println(data)
		return nil
	} else {
		return err
	}
}

func (e *StdoutExporter) Shutdown() error {
	return nil
}
