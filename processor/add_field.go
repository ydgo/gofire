package processor

import (
	"errors"
	"gofire/component"
)

func init() {
	component.RegisterProcessor("add_field", NewAddField)
}

type AddField struct {
	field string
	value interface{}
}

func NewAddField(cfg map[string]interface{}) (component.Processor, error) {
	field := cfg["field"].(string)
	if field == "" {
		return nil, errors.New("field is required")
	}
	value, ok := cfg["value"]
	if !ok {
		return nil, errors.New("value is required")
	}
	return &AddField{field, value}, nil
}

func (p *AddField) Process(messages ...map[string]interface{}) ([]map[string]interface{}, error) {
	for _, message := range messages {
		message[p.field] = p.value
	}
	// 指标上报
	// 自定义指标上报
	return messages, nil
}
