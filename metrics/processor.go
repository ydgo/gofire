package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

type ProcessorBasicMetrics struct {
	mutex            sync.RWMutex
	Total            *prometheus.CounterVec
	Failures         *prometheus.CounterVec
	DurationInMillis *prometheus.CounterVec
	LastProcessTime  *prometheus.GaugeVec
	// todo 只存储单个组件类型的自定义指标，指定固定的 namespace
	CustomMetric map[string]prometheus.Collector
}

func NewProcessorMetrics() *ProcessorBasicMetrics {
	return &ProcessorBasicMetrics{
		CustomMetric: make(map[string]prometheus.Collector),
		Total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "processor",
			Name:      "total",
			Help:      "处理数据总量",
		}, []string{"pipeline", "processor_type"}),
		Failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "processor",
			Name:      "failures",
			Help:      "处理数据失败总量",
		}, []string{"pipeline", "processor_type"}),
		DurationInMillis: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "processor",
			Name:      "process_duration",
			Help:      "处理数据执行总毫秒",
		}, []string{"pipeline", "processor_type"}),
		LastProcessTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gofire",
			Subsystem: "processor",
			Name:      "last_receive_time",
			Help:      "上次处理数据时间",
		}, []string{"pipeline", "processor_type"}),
	}
}

func (m *ProcessorBasicMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.Total.Describe(ch)
	m.Failures.Describe(ch)
	m.DurationInMillis.Describe(ch)
	m.LastProcessTime.Describe(ch)
	for _, collector := range m.CustomMetric {
		collector.Describe(ch)
	}
}

func (m *ProcessorBasicMetrics) Collect(ch chan<- prometheus.Metric) {
	m.Total.Collect(ch)
	m.Failures.Collect(ch)
	m.DurationInMillis.Collect(ch)
	m.LastProcessTime.Collect(ch)
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, collector := range m.CustomMetric {
		collector.Collect(ch)
	}
}

func (m *ProcessorBasicMetrics) IncTotal(pipeline, processorType string) {
	m.Total.WithLabelValues(pipeline, processorType).Inc()
}

func (m *ProcessorBasicMetrics) AddTotal(pipeline, processorType string, value float64) {
	m.Total.WithLabelValues(pipeline, processorType).Add(value)
}

func (m *ProcessorBasicMetrics) IncFailures(pipeline, processorType string) {
	m.Failures.WithLabelValues(pipeline, processorType).Inc()
}

func (m *ProcessorBasicMetrics) AddFailures(pipeline, processorType string, value float64) {
	m.Failures.WithLabelValues(pipeline, processorType).Add(value)
}

func (m *ProcessorBasicMetrics) AddProcessDuration(pipeline, processorType string, duration time.Duration) {
	m.DurationInMillis.WithLabelValues(pipeline, processorType).Add(float64(duration.Milliseconds()))
}

func (m *ProcessorBasicMetrics) SetLastReceiveTime(pipeline, processorType string, t time.Time) {
	m.LastProcessTime.WithLabelValues(pipeline, processorType).Set(float64(t.Unix()))
}

func (m *ProcessorBasicMetrics) RegisterCustomMetrics(name string, collector prometheus.Collector) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.CustomMetric[name]; ok {
		return fmt.Errorf("metric %s already exists", name)
	}
	m.CustomMetric[name] = collector
	return nil
}

func (m *ProcessorBasicMetrics) UnregisterCustomMetrics(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.CustomMetric, name)
}
