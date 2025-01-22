package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

type ExporterBasicMetrics struct {
	mutex            sync.RWMutex
	Total            *prometheus.CounterVec
	Failures         *prometheus.CounterVec
	DurationInMillis *prometheus.CounterVec
	LastExportTime   *prometheus.GaugeVec
	// todo 只存储单个组件类型的自定义指标，指定固定的 namespace
	CustomMetric map[string]prometheus.Collector
}

// todo 不可导出

func NewExporterBasicMetrics() *ExporterBasicMetrics {
	return &ExporterBasicMetrics{
		CustomMetric: make(map[string]prometheus.Collector),
		Total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "exporter",
			Name:      "total",
			Help:      "转发数据总量",
		}, []string{"pipeline", "exporter_type"}),
		Failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "exporter",
			Name:      "failures",
			Help:      "转发数据失败总量",
		}, []string{"pipeline", "exporter_type"}),
		DurationInMillis: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "exporter",
			Name:      "process_duration",
			Help:      "转发数据执行总毫秒",
		}, []string{"pipeline", "exporter_type"}),
		LastExportTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gofire",
			Subsystem: "exporter",
			Name:      "last_export_time",
			Help:      "上次转发数据时间",
		}, []string{"pipeline", "exporter_type"}),
	}
}

func (m *ExporterBasicMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.Total.Describe(ch)
	m.Failures.Describe(ch)
	m.DurationInMillis.Describe(ch)
	m.LastExportTime.Describe(ch)
	for _, collector := range m.CustomMetric {
		collector.Describe(ch)
	}
}

func (m *ExporterBasicMetrics) Collect(ch chan<- prometheus.Metric) {
	m.Total.Collect(ch)
	m.Failures.Collect(ch)
	m.DurationInMillis.Collect(ch)
	m.LastExportTime.Collect(ch)
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, collector := range m.CustomMetric {
		collector.Collect(ch)
	}
}

func (m *ExporterBasicMetrics) IncTotal(pipeline, exporterType string) {
	m.Total.WithLabelValues(pipeline, exporterType).Inc()
}

func (m *ExporterBasicMetrics) AddTotal(pipeline, exporterType string, value float64) {
	m.Total.WithLabelValues(pipeline, exporterType).Add(value)
}

func (m *ExporterBasicMetrics) IncFailures(pipeline, exporterType string) {
	m.Failures.WithLabelValues(pipeline, exporterType).Inc()
}

func (m *ExporterBasicMetrics) AddFailures(pipeline, exporterType string, value float64) {
	m.Failures.WithLabelValues(pipeline, exporterType).Add(value)
}

func (m *ExporterBasicMetrics) AddProcessDuration(pipeline, exporterType string, duration time.Duration) {
	m.DurationInMillis.WithLabelValues(pipeline, exporterType).Add(float64(duration.Milliseconds()))
}

func (m *ExporterBasicMetrics) SetLastExportTime(pipeline, exporterType string, t time.Time) {
	m.LastExportTime.WithLabelValues(pipeline, exporterType).Set(float64(t.Unix()))
}

func (m *ExporterBasicMetrics) RegisterCustomMetrics(name string, collector prometheus.Collector) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.CustomMetric[name]; ok {
		return fmt.Errorf("metric %s already exists", name)
	}
	m.CustomMetric[name] = collector
	return nil
}

func (m *ExporterBasicMetrics) UnregisterCustomMetrics(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.CustomMetric, name)
}
