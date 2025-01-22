package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

// CustomMetric 定义自定义指标
type CustomMetric struct {
	Name   string
	Value  float64
	Type   prometheus.ValueType
	Labels map[string]string
}

// Collector 包含程序级别和 pipeline 级别的指标收集
type Collector struct {
	mu sync.RWMutex
	// 程序级别指标
	startTime time.Time

	// 配置管理模块指标
	configMetrics *ConfigMetrics

	// Receiver 级别指标
	receiverBasicMetrics *ReceiverBasicMetrics
	// todo 不应该保存一个map，因为 receiver 停止后还需要删除key，很不优雅
	// todo 全局应该只有一个 receiver collector，因为 receiver 相关指标是根据 label 来区分的，而不是根据 map 的key区分

	// todo 提供方法让用户在 具体的 receiver 中创建 collector 再注册到全局，然后全局进行暴露metrics
	// todo 这样对代码没有侵入性

	// Pipeline Manager Metrics
	pipelineManagerMetrics *PipelineManagerMetrics

	processorBasicMetrics *ProcessorBasicMetrics

	exporterBasicMetrics *ExporterBasicMetrics

	pipelineMetrics *PipelineMetrics
	// todo 这样也是可以的
	tmpMetrics prometheus.Collector
	// 自定义指标描述符缓存
	customMetricDescs map[string]*prometheus.Desc
}

// NewCollector 创建新的收集器
func NewCollector() *Collector {
	return &Collector{
		startTime:              time.Now(),
		configMetrics:          NewConfigMetrics(),
		receiverBasicMetrics:   NewReceiverBasicMetrics(),
		pipelineManagerMetrics: NewPipelineManagerMetrics(),
		processorBasicMetrics:  NewProcessorMetrics(),
		exporterBasicMetrics:   NewExporterBasicMetrics(),
		pipelineMetrics:        NewPipelineMetrics(),
		tmpMetrics:             NewTmpMetrics(),
	}
}

var defaultCollector = NewCollector()

func Default() *Collector {
	return defaultCollector
}

// Describe 实现 prometheus.Collector 接口
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	// 配置管理指标
	c.configMetrics.Describe(ch)

	// Receiver 接收数据指标
	c.receiverBasicMetrics.Describe(ch)

	// Pipeline Manager 指标
	c.pipelineManagerMetrics.Describe(ch)

	// Processor 指标
	c.processorBasicMetrics.Describe(ch)

	c.exporterBasicMetrics.Describe(ch)

	c.pipelineMetrics.Describe(ch)
}

// Collect 实现 prometheus.Collector 接口
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 收集配置管理指标
	c.configMetrics.Collect(ch)

	// 收集 Receiver 指标
	c.receiverBasicMetrics.Collect(ch)

	// 收集 Pipeline Manager 指标
	c.pipelineManagerMetrics.Collect(ch)

	// 收集 Processor 指标
	c.processorBasicMetrics.Collect(ch)

	c.exporterBasicMetrics.Collect(ch)

	c.pipelineMetrics.Collect(ch)

}

// getOrCreateCustomMetricDesc 获取或创建自定义指标的描述符
func (c *Collector) getOrCreateCustomMetricDesc(metric CustomMetric) *prometheus.Desc {
	key := fmt.Sprintf("%s_%s", metric.Name, metric.Type.ToDTO().String())
	desc, exists := c.customMetricDescs[key]
	if !exists {
		desc = prometheus.NewDesc(
			fmt.Sprintf("gofire_receiver_%s", metric.Name),
			fmt.Sprintf("Custom metric: %s", metric.Name),
			c.getLabelsKeys(metric.Labels),
			nil,
		)
		c.customMetricDescs[key] = desc
	}
	return desc
}

// 辅助方法
func (c *Collector) getLabelsKeys(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	return keys
}

func (c *Collector) getLabelsValues(labels map[string]string) []string {
	values := make([]string, 0, len(labels))
	for _, v := range labels {
		values = append(values, v)
	}
	return values
}

func (c *Collector) ConfigMetrics() *ConfigMetrics {
	return c.configMetrics
}

func (c *Collector) ReceiverBasicMetrics() *ReceiverBasicMetrics {
	return c.receiverBasicMetrics
}
func (c *Collector) PipelineManagerMetrics() *PipelineManagerMetrics {
	return c.pipelineManagerMetrics
}
func (c *Collector) ProcessorBasicMetrics() *ProcessorBasicMetrics {
	return c.processorBasicMetrics
}
func (c *Collector) ExporterBasicMetrics() *ExporterBasicMetrics {
	return c.exporterBasicMetrics
}

func (c *Collector) PipelineMetrics() *PipelineMetrics {
	return c.pipelineMetrics
}

func (c *Collector) TmpMetrics() *TmpMetrics {
	return c.tmpMetrics.(*TmpMetrics)
}
