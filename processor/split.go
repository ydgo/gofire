package processor

import (
	"errors"
	"gofire/component"
	"gofire/metrics"
	"strings"
	"time"
)

func init() {
	component.RegisterProcessor("split", NewSplitProcessor)
}

type SplitProcessor struct {
	pipeline string
	sep      string
	metrics  *metrics.ProcessorBasicMetrics
}

func NewSplitProcessor(pipeName string, cfg map[string]interface{}, collector *metrics.Collector) (component.Processor, error) {
	if sep, ok := cfg["separator"].(string); ok {
		return &SplitProcessor{sep: sep, pipeline: pipeName, metrics: collector.ProcessorBasicMetrics()}, nil
	}
	return nil, errors.New("separator not found in config")
}

func (p *SplitProcessor) Process(messages ...map[string]interface{}) ([]map[string]interface{}, error) {
	start := time.Now()
	data := make([]map[string]interface{}, 0)
	for _, message := range messages {
		// todo 每个 pipeline 可能会有多个同类型的 processor
		p.metrics.IncTotal(p.pipeline, "split")
		raw, ok := message["message"].(string)
		if !ok {
			// todo error ?
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
	p.metrics.AddProcessDuration(p.pipeline, "split", time.Since(start))
	return data, nil
}
