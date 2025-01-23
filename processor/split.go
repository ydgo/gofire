package processor

import (
	"errors"
	"gofire/component"
	"gofire/event"
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

func (p *SplitProcessor) Process(events ...*event.Event) ([]*event.Event, error) {
	start := time.Now()
	data := make([]*event.Event, 0)
	for _, evt := range events {
		// todo 每个 pipeline 可能会有多个同类型的 processor
		p.metrics.IncTotal(p.pipeline, "split")

		raws := strings.Split(evt.Message, p.sep)
		for _, raw := range raws {
			newEvent := evt.Clone()
			newEvent.SetMessage(raw)
			data = append(data, newEvent)
		}
		// 丢弃并释放旧的 event
		evt.Release()
	}
	p.metrics.AddProcessDuration(p.pipeline, "split", time.Since(start))
	return data, nil
}
