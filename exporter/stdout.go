package exporter

import (
	"fmt"
	"gofire/component"
	"gofire/event"
)

func init() {
	component.RegisterExporter("stdout", NewStdoutExporter)
}

type StdoutExporter struct {
}

func NewStdoutExporter(opts component.ExporterOpts) (component.Exporter, error) {
	return &StdoutExporter{}, nil
}

func (e *StdoutExporter) Export(evt *event.Event) error {
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
