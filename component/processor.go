package component

import "gofire/event"

type Processor interface {
	// Process 如果丢弃或者生成新的 event，需要释放原来的 event
	Process(...*event.Event) ([]*event.Event, error)
}

type ProcessorNode struct {
	processor Processor
	next      *ProcessorNode
}

type ProcessorLink struct {
	root *ProcessorNode
}

func (l *ProcessorLink) Process(evt *event.Event) ([]*event.Event, error) {
	events := make([]*event.Event, 0, 1)
	events = append(events, evt)
	for node := l.root; node != nil; node = node.next {
		var err error
		events, err = node.processor.Process(events...) // evt 的释放由 Process 决定，如果仍需使用请复制一份
		if err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (l *ProcessorLink) addProcessor(processor Processor) {
	if l.root == nil {
		l.root = &ProcessorNode{processor: processor}
	} else {
		// find last node
		var last = l.root
		for last.next != nil {
			last = last.next
		}
		last.next = &ProcessorNode{processor: processor}
	}
}

func BuildProcessorLink(processors ...Processor) *ProcessorLink {
	link := &ProcessorLink{}
	for _, processor := range processors {
		link.addProcessor(processor)
	}
	return link
}
