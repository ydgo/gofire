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
	// todo 不应该保存一个map，因为 receiver 停止后还需要删除key，很不优雅
	// todo 全局应该只有一个 receiver collector，因为 receiver 相关指标是根据 label 来区分的，而不是根据 map 的key区分

	// todo 提供方法让用户在 具体的 receiver 中创建 collector 再注册到全局，然后全局进行暴露metrics
	// todo 这样对代码没有侵入性
	receiverMetrics map[string]ReceiverMetrics // key: pipelineID:receiverType

	// Pipeline 级别指标
	pipelineStats map[string]*PipelineStats

	// 自定义指标描述符缓存
	customMetricDescs map[string]*prometheus.Desc
}

// PipelineStats 存储单个 pipeline 的统计信息
type PipelineStats struct {
	ID string
	// 接收器统计
	ReceivedCount float64
	ReceiverType  string
	// 处理器统计
	ProcessedCount   float64
	ProcessingErrors float64
	// 导出器统计
	ExportedCount float64
	ExportErrors  float64
	// 性能统计
	ProcessingLatency float64
	BufferSize        float64
	// 状态
	IsRunning bool
}

// 定义所有指标描述符
var (
	// 程序级别指标描述符
	processStartTimeDesc = prometheus.NewDesc(
		"gofire_process_start_timestamp_seconds",
		"程序启动时间（Unix 时间戳）",
		nil, nil,
	)

	// Pipeline 级别指标描述符
	pipelineRunningDesc = prometheus.NewDesc(
		"gofire_pipeline_running",
		"Pipeline 运行状态（1表示运行中，0表示停止）",
		[]string{"pipeline_id"}, nil,
	)
	receivedCountDesc = prometheus.NewDesc(
		"gofire_received_records_total",
		"接收到的记录总数",
		[]string{"pipeline_id", "receiver_type"}, nil,
	)
	processedCountDesc = prometheus.NewDesc(
		"gofire_processed_records_total",
		"处理完成的记录总数",
		[]string{"pipeline_id"}, nil,
	)
	processingErrorsDesc = prometheus.NewDesc(
		"gofire_processing_errors_total",
		"处理错误总数",
		[]string{"pipeline_id"}, nil,
	)
	exportedCountDesc = prometheus.NewDesc(
		"gofire_exported_records_total",
		"导出的记录总数",
		[]string{"pipeline_id"}, nil,
	)
	exportErrorsDesc = prometheus.NewDesc(
		"gofire_export_errors_total",
		"导出错误总数",
		[]string{"pipeline_id"}, nil,
	)
	processingLatencyDesc = prometheus.NewDesc(
		"gofire_processing_latency_seconds",
		"数据处理延迟时间（秒）",
		[]string{"pipeline_id"}, nil,
	)
	bufferSizeDesc = prometheus.NewDesc(
		"gofire_buffer_size",
		"缓冲区当前大小",
		[]string{"pipeline_id"}, nil,
	)
)

// NewCollector 创建新的收集器
func NewCollector() *Collector {
	return &Collector{
		startTime:         time.Now(),
		pipelineStats:     make(map[string]*PipelineStats),
		receiverMetrics:   make(map[string]ReceiverMetrics),
		customMetricDescs: make(map[string]*prometheus.Desc),
		configMetrics: &ConfigMetrics{
			Total: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "gofire",
				Subsystem: "config",
				Name:      "total",
				Help:      "Config 总数",
			}),
			Reload: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: "gofire",
				Subsystem: "config",
				Name:      "reload",
				Help:      "Config 重载总次数",
			}, []string{"name"}),
		},
	}
}

var defaultCollector = NewCollector()

func Default() *Collector {
	return defaultCollector
}

// Describe 实现 prometheus.Collector 接口
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	// 配置管理指标
	ch <- c.configMetrics.Total.Desc()
	c.configMetrics.Reload.Describe(ch)

	// 程序级别指标
	ch <- processStartTimeDesc

	// Pipeline 级别指标
	ch <- pipelineRunningDesc
	ch <- receivedCountDesc
	ch <- processedCountDesc
	ch <- processingErrorsDesc
	ch <- exportedCountDesc
	ch <- exportErrorsDesc
	ch <- processingLatencyDesc
	ch <- bufferSizeDesc

	// Receiver 基础指标描述符
	ch <- receiverReceivedTotalDesc
	ch <- receiverFailuresTotalDesc
	ch <- receiverLastReceiveTimeDesc

	// 发送所有自定义指标的描述符
	c.mu.RLock()
	for _, desc := range c.customMetricDescs {
		ch <- desc
	}
	c.mu.RUnlock()
}

// Collect 实现 prometheus.Collector 接口
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 收集配置管理指标
	c.configMetrics.Total.Collect(ch)
	c.configMetrics.Reload.Collect(ch)

	// 收集程序级别指标
	ch <- prometheus.MustNewConstMetric(
		processStartTimeDesc,
		prometheus.GaugeValue,
		float64(c.startTime.Unix()),
	)

	// 收集 Pipeline 级别指标
	for _, stats := range c.pipelineStats {
		running := 0.0
		if stats.IsRunning {
			running = 1.0
		}

		ch <- prometheus.MustNewConstMetric(
			pipelineRunningDesc,
			prometheus.GaugeValue,
			running,
			stats.ID,
		)
		ch <- prometheus.MustNewConstMetric(
			receivedCountDesc,
			prometheus.CounterValue,
			stats.ReceivedCount,
			stats.ID,
			stats.ReceiverType,
		)
		ch <- prometheus.MustNewConstMetric(
			processedCountDesc,
			prometheus.CounterValue,
			stats.ProcessedCount,
			stats.ID,
		)
		ch <- prometheus.MustNewConstMetric(
			processingErrorsDesc,
			prometheus.CounterValue,
			stats.ProcessingErrors,
			stats.ID,
		)
		ch <- prometheus.MustNewConstMetric(
			exportedCountDesc,
			prometheus.CounterValue,
			stats.ExportedCount,
			stats.ID,
		)
		ch <- prometheus.MustNewConstMetric(
			exportErrorsDesc,
			prometheus.CounterValue,
			stats.ExportErrors,
			stats.ID,
		)
		ch <- prometheus.MustNewConstMetric(
			processingLatencyDesc,
			prometheus.GaugeValue,
			stats.ProcessingLatency,
			stats.ID,
		)
		ch <- prometheus.MustNewConstMetric(
			bufferSizeDesc,
			prometheus.GaugeValue,
			stats.BufferSize,
			stats.ID,
		)
	}

	// 收集 Receiver 指标
	for _, stats := range c.receiverMetrics {
		basicStats := stats.GetBasicMetrics()

		// 收集基础指标
		ch <- prometheus.MustNewConstMetric(
			receiverReceivedTotalDesc,
			prometheus.CounterValue,
			basicStats.ReceivedTotal,
			basicStats.Pipeline,
			basicStats.ReceiverType,
		)

		ch <- prometheus.MustNewConstMetric(
			receiverFailuresTotalDesc,
			prometheus.CounterValue,
			basicStats.ReceivedFailures,
			basicStats.Pipeline,
			basicStats.ReceiverType,
		)

		ch <- prometheus.MustNewConstMetric(
			receiverLastReceiveTimeDesc,
			prometheus.GaugeValue,
			float64(basicStats.LastReceiveTime.Unix()),
			basicStats.Pipeline,
			basicStats.ReceiverType,
		)

		// 收集自定义指标
		for _, metric := range stats.GetCustomMetrics() {
			desc := c.getOrCreateCustomMetricDesc(metric)
			ch <- prometheus.MustNewConstMetric(
				desc,
				metric.Type,
				metric.Value,
				c.getLabelsValues(metric.Labels)...,
			)
		}
	}
}

// Pipeline 级别的指标更新方法
func (c *Collector) UpdatePipelineStats(id string, stats *PipelineStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pipelineStats[id] = stats
}

func (c *Collector) RemovePipelineStats(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pipelineStats, id)
}

// UpdateReceiverMetrics 更新 receiver 统计信息
func (c *Collector) UpdateReceiverMetrics(pipeline, receiverType string, metrics ReceiverMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%s:%s", pipeline, receiverType)
	c.receiverMetrics[key] = metrics
}

func (c *Collector) RemoveReceiverMetrics(pipeline, receiverType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.receiverMetrics, fmt.Sprintf("%s:%s", pipeline, receiverType))
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
