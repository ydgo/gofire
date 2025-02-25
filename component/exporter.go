package component

import (
	"gofire/event"
	"gofire/pkg/logger"
)

type Exporter interface {
	Export(*event.Event) error
	Shutdown() error
}

type Exporters struct {
	exporters []Exporter
}

func BuildExporters(exporters ...Exporter) *Exporters {
	return &Exporters{exporters: exporters}
}

func (e *Exporters) Export(event *event.Event) error {
	for _, exporter := range e.exporters {
		if err := exporter.Export(event); err != nil {
			logger.Warnf("exporter export event failed: %s", err)
			continue
		}
	}
	// 一个事件被所有导出组件导出后被释放
	event.Release() // 释放 event
	return nil
}

func (e *Exporters) Shutdown() error {
	for _, exporter := range e.exporters {
		if err := exporter.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}
