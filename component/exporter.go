package component

type Exporter interface {
	Export(...map[string]interface{}) error
	Shutdown() error
}

type Exporters struct {
	exporters []Exporter
}

func BuildExporters(exporters ...Exporter) *Exporters {
	return &Exporters{exporters: exporters}
}

func (e *Exporters) Export(messages ...map[string]interface{}) error {
	for _, exporter := range e.exporters {
		if err := exporter.Export(messages...); err != nil {
			return err
		}
	}
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
