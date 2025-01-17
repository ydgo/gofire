package processor

import (
	"errors"
	"gofire/component"
	"strings"
)

func init() {
	component.RegisterProcessor("split", NewSplitProcessor)
}

type SplitProcessor struct {
	sep string
}

func NewSplitProcessor(cfg map[string]interface{}) (component.Processor, error) {
	if sep, ok := cfg["separator"].(string); ok {
		return &SplitProcessor{sep: sep}, nil
	}
	return nil, errors.New("separator not found in config")
}

func (p *SplitProcessor) Process(messages ...map[string]interface{}) ([]map[string]interface{}, error) {
	data := make([]map[string]interface{}, 0)
	for _, message := range messages {
		raw, ok := message["message"].(string)
		if !ok {
			continue
		}
		raws := strings.Split(raw, p.sep)
		for _, raw = range raws {
			tmp := make(map[string]interface{})
			for k, v := range message {
				tmp[k] = v
			}
			tmp["message"] = raw
			data = append(data, tmp)
		}
	}
	return data, nil
}
