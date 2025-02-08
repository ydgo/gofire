package processor

import (
	"errors"
	"gofire/component"
	"gofire/event"
	"strings"
)

func init() {
	component.RegisterProcessor("split", NewSplitProcessor)
}

type SplitProcessor struct {
	sep string
}

func NewSplitProcessor(opts component.ProcessorOpts) (component.Processor, error) {
	if sep, ok := opts.Setting["separator"].(string); ok {
		return &SplitProcessor{sep: sep}, nil
	}
	return nil, errors.New("separator not found in config")
}

func (p *SplitProcessor) Process(events ...*event.Event) ([]*event.Event, error) {
	data := make([]*event.Event, 0)
	for _, evt := range events {
		raws := strings.Split(evt.Message, p.sep)
		for _, raw := range raws {
			newEvent := evt.Clone()
			newEvent.SetMessage(raw)
			data = append(data, newEvent)
		}
		// 丢弃并释放旧的 event
		evt.Release()
	}
	return data, nil
}
