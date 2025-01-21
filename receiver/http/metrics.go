package http

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"gofire/metrics"
)

type Stats struct {
	metrics.ReceiverBasicMetrics
	statusCodes    map[int]float64
	requestLatency float64
}

func (s *Stats) GetBasicMetrics() metrics.ReceiverBasicMetrics {
	return s.ReceiverBasicMetrics
}

func (s *Stats) GetCustomMetrics() []metrics.CustomMetric {
	customMetrics := make([]metrics.CustomMetric, 0)

	// 添加状态码指标
	for code, count := range s.statusCodes {
		customMetrics = append(customMetrics, metrics.CustomMetric{
			Name:  "http_response_codes",
			Value: count,
			Type:  prometheus.CounterValue,
			Labels: map[string]string{
				"pipeline":      s.Pipeline,
				"receiver_type": s.ReceiverType,
				"code":          fmt.Sprintf("%d", code),
			},
		})
	}

	// 添加延迟指标
	customMetrics = append(customMetrics, metrics.CustomMetric{
		Name:  "http_request_duration_seconds",
		Value: s.requestLatency,
		Type:  prometheus.GaugeValue,
		Labels: map[string]string{
			"pipeline":      s.Pipeline,
			"receiver_type": s.ReceiverType,
		},
	})

	return customMetrics
}
