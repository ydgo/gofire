package component

import (
	"fmt"
	"gofire/metrics"
	"sync"
)

type Opts struct {
	Pipeline      string
	ComponentType string
	Setting       map[string]any // 配置
}

type ReceiverOpts Opts

// 定义组件创建器的函数类型

type ReceiverCreator func(opts ReceiverOpts) (Receiver, error)
type ProcessorCreator func(pipeName string, config map[string]interface{}, metrics *metrics.Collector) (Processor, error)
type ExporterCreator func(pipeName string, config map[string]interface{}, metrics *metrics.Collector) (Exporter, error)

// Factory 组件工厂，用于创建各种组件
type Factory struct {
	receivers  map[string]ReceiverCreator
	processors map[string]ProcessorCreator
	exporters  map[string]ExporterCreator
	mu         sync.RWMutex
	metrics    *metrics.Collector
}

func (f *Factory) Metrics() *metrics.Collector {
	return f.metrics
}

var factory = NewComponentFactory(metrics.Default())

// NewComponentFactory 创建新的组件工厂
func NewComponentFactory(metrics *metrics.Collector) *Factory {
	return &Factory{
		metrics:    metrics,
		receivers:  make(map[string]ReceiverCreator),
		processors: make(map[string]ProcessorCreator),
		exporters:  make(map[string]ExporterCreator),
	}
}

// RegisterReceiver 注册接收器创建器
func (f *Factory) RegisterReceiver(typeName string, creator ReceiverCreator) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.receivers[typeName] = creator
}

// RegisterProcessor 注册处理器创建器
func (f *Factory) RegisterProcessor(typeName string, creator ProcessorCreator) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.processors[typeName] = creator
}

// RegisterExporter 注册导出器创建器
func (f *Factory) RegisterExporter(typeName string, creator ExporterCreator) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exporters[typeName] = creator
}

func RegisterReceiver(typeName string, creator ReceiverCreator) {
	factory.RegisterReceiver(typeName, creator)
}

func RegisterProcessor(typeName string, creator ProcessorCreator) {
	factory.RegisterProcessor(typeName, creator)
}

func RegisterExporter(typeName string, creator ExporterCreator) {
	factory.RegisterExporter(typeName, creator)
}

// CreateReceiver 创建接收器实例
func (f *Factory) CreateReceiver(opts ReceiverOpts) (Receiver, error) {
	f.mu.RLock()
	creator, exists := f.receivers[opts.ComponentType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("未知的接收器类型: %s", opts.ComponentType)
	}
	return creator(opts)
}

// CreateProcessor 创建处理器实例
func (f *Factory) CreateProcessor(pipeName string, typeName string, config map[string]interface{}) (Processor, error) {
	f.mu.RLock()
	creator, exists := f.processors[typeName]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("未知的处理器类型: %s", typeName)
	}

	return creator(pipeName, config, f.metrics)
}

// CreateExporter 创建导出器实例
func (f *Factory) CreateExporter(pipeName string, typeName string, config map[string]interface{}) (Exporter, error) {
	f.mu.RLock()
	creator, exists := f.exporters[typeName]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("未知的导出器类型: %s", typeName)
	}

	return creator(pipeName, config, f.metrics)
}

// GetReceiverTypes 获取所有已注册的接收器类型
func (f *Factory) GetReceiverTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.receivers))
	for typeName := range f.receivers {
		types = append(types, typeName)
	}
	return types
}

// GetProcessorTypes 获取所有已注册的处理器类型
func (f *Factory) GetProcessorTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.processors))
	for typeName := range f.processors {
		types = append(types, typeName)
	}
	return types
}

// GetExporterTypes 获取所有已注册的导出器类型
func (f *Factory) GetExporterTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.exporters))
	for typeName := range f.exporters {
		types = append(types, typeName)
	}
	return types
}

func GetReceiverTypes() []string {
	return factory.GetReceiverTypes()
}
func GetProcessorTypes() []string {
	return factory.GetProcessorTypes()
}
func GetExporterTypes() []string {
	return factory.GetExporterTypes()
}

func DefaultFactory() *Factory {
	return factory
}
