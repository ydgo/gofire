package processor

import (
	"errors"
	"gofire/component"
	"gofire/event"
)

func init() {
	component.RegisterProcessor("add_field", NewAddField)
}

type AddField struct {
	pipeline string
	field    string
	value    interface{}
}

func NewAddField(opts component.ProcessorOpts) (component.Processor, error) {
	field := opts.Setting["field"].(string)
	if field == "" {
		return nil, errors.New("field is required")
	}
	value, ok := opts.Setting["value"]
	if !ok {
		return nil, errors.New("value is required")
	}
	return &AddField{
		pipeline: opts.Pipeline,
		field:    field,
		value:    value,
	}, nil
}

func (p *AddField) Process(events ...*event.Event) ([]*event.Event, error) {
	for _, evt := range events {
		evt.AddField(p.field, p.value)
	}
	return events, nil
}
