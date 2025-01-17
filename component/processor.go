package component

type Processor interface {
	Process(...map[string]interface{}) ([]map[string]interface{}, error)
}

type ProcessorNode struct {
	processor Processor
	next      *ProcessorNode
}

type ProcessorLink struct {
	root *ProcessorNode
}

func (l *ProcessorLink) Process(message map[string]interface{}) ([]map[string]interface{}, error) {
	messages := make([]map[string]interface{}, 0, 1)
	messages = append(messages, message)
	for node := l.root; node != nil; node = node.next {
		var err error
		messages, err = node.processor.Process(messages...)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
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
