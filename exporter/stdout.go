package exporter

import (
	"fmt"
	"gofire/component"
)

func init() {
	component.RegisterExporter("stdout", NewStdoutExporter)
}

type StdoutExporter struct{}

func NewStdoutExporter(config map[string]interface{}) (component.Exporter, error) {
	return &StdoutExporter{}, nil
}

func (e *StdoutExporter) Export(messages ...map[string]interface{}) error {
	for _, msg := range messages {
		fmt.Println(msg)
	}
	return nil
}

func (e *StdoutExporter) Shutdown() error {
	return nil
}
