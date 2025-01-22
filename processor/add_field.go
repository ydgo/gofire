package processor

import (
	"errors"
	"gofire/component"
	"gofire/metrics"
	"time"
)

func init() {
	component.RegisterProcessor("add_field", NewAddField)
}

type AddField struct {
	pipeline string
	metrics  *metrics.ProcessorBasicMetrics
	field    string
	value    interface{}
}

func NewAddField(pipeName string, cfg map[string]interface{}, collector *metrics.Collector) (component.Processor, error) {
	field := cfg["field"].(string)
	if field == "" {
		return nil, errors.New("field is required")
	}
	value, ok := cfg["value"]
	if !ok {
		return nil, errors.New("value is required")
	}
	return &AddField{
		pipeline: pipeName,
		metrics:  collector.ProcessorBasicMetrics(),
		field:    field,
		value:    value,
	}, nil
}

func (p *AddField) Process(messages ...map[string]interface{}) ([]map[string]interface{}, error) {
	start := time.Now()
	for _, message := range messages {
		p.metrics.IncTotal(p.pipeline, "add_field")
		message[p.field] = p.value
	}
	p.metrics.AddProcessDuration(p.pipeline, "add_field", time.Since(start))
	return messages, nil
}
