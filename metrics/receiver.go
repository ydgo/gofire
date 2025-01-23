package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

type ReceiverBasicMetrics struct {
	mutex           sync.RWMutex
	Total           *prometheus.CounterVec
	Failures        *prometheus.CounterVec
	ProcessDuration *prometheus.CounterVec
	LastReceiveTime *prometheus.GaugeVec
	CustomMetric    map[string]prometheus.Collector
}

func NewReceiverBasicMetrics() *ReceiverBasicMetrics {
	return &ReceiverBasicMetrics{
		CustomMetric: make(map[string]prometheus.Collector),
		Total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "receiver",
			Name:      "total",
			Help:      "接收数据总量",
		}, []string{"pipeline", "receiver_type"}),
		Failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "receiver",
			Name:      "failures",
			Help:      "接收数据失败总量",
		}, []string{"pipeline", "receiver_type"}),
		ProcessDuration: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gofire",
			Subsystem: "receiver",
			Name:      "process_duration",
			Help:      "接收数据执行总时间",
		}, []string{"pipeline", "receiver_type"}),
		LastReceiveTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gofire",
			Subsystem: "receiver",
			Name:      "last_receive_time",
			Help:      "上次接收数据时间",
		}, []string{"pipeline", "receiver_type"}),
	}
}

func (m *ReceiverBasicMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.Total.Describe(ch)
	m.Failures.Describe(ch)
	m.ProcessDuration.Describe(ch)
	m.LastReceiveTime.Describe(ch)
	for _, collector := range m.CustomMetric {
		collector.Describe(ch)
	}
}

func (m *ReceiverBasicMetrics) Collect(ch chan<- prometheus.Metric) {
	m.Total.Collect(ch)
	m.Failures.Collect(ch)
	m.ProcessDuration.Collect(ch)
	m.LastReceiveTime.Collect(ch)
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, collector := range m.CustomMetric {
		collector.Collect(ch)
	}
}

func (m *ReceiverBasicMetrics) IncTotal(pipeline, receiverType string) {
	m.Total.WithLabelValues(pipeline, receiverType).Inc()
}

func (m *ReceiverBasicMetrics) AddTotal(pipeline, receiverType string, value float64) {
	m.Total.WithLabelValues(pipeline, receiverType).Add(value)
}

func (m *ReceiverBasicMetrics) IncFailures(pipeline, receiverType string) {
	m.Failures.WithLabelValues(pipeline, receiverType).Inc()
}

func (m *ReceiverBasicMetrics) AddFailures(pipeline, receiverType string, value float64) {
	m.Failures.WithLabelValues(pipeline, receiverType).Add(value)
}

func (m *ReceiverBasicMetrics) AddProcessDuration(pipeline, receiverType string, duration time.Duration) {
	m.ProcessDuration.WithLabelValues(pipeline, receiverType).Add(float64(duration.Milliseconds()))
}

func (m *ReceiverBasicMetrics) SetLastReceiveTime(pipeline, receiverType string, t time.Time) {
	m.LastReceiveTime.WithLabelValues(pipeline, receiverType).Set(float64(t.Unix()))
}

// todo 感觉这个方法应该挂在全局的 collector 上

func (m *ReceiverBasicMetrics) RegisterCustomMetrics(name string, collector prometheus.Collector) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.CustomMetric[name]; ok {
		return fmt.Errorf("metric %s already exists", name)
	}
	m.CustomMetric[name] = collector
	return nil
}

func (m *ReceiverBasicMetrics) UnregisterCustomMetrics(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.CustomMetric, name)
}

func (m *ReceiverBasicMetrics) Delete(pipeline string) {
	m.Total.DeleteLabelValues(pipeline)
	m.Failures.DeleteLabelValues(pipeline)
	m.ProcessDuration.DeleteLabelValues(pipeline)
	m.LastReceiveTime.DeleteLabelValues(pipeline)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	clear(m.CustomMetric)

}
