package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	// Receiver 基础指标
	receiverReceivedTotalDesc = prometheus.NewDesc(
		"gofire_receiver_received_total",
		"Receiver 接收到的数据总数",
		[]string{"pipeline", "receiver_type"}, nil,
	)
	receiverFailuresTotalDesc = prometheus.NewDesc(
		"gofire_receiver_failures_total",
		"Receiver 接收失败的数据总数",
		[]string{"pipeline", "receiver_type"}, nil,
	)
	receiverLastReceiveTimeDesc = prometheus.NewDesc(
		"gofire_receiver_last_receive_timestamp_seconds",
		"Receiver 最后一次接收数据的时间戳",
		[]string{"pipeline", "receiver_type"}, nil,
	)
)

type ReceiverMetrics interface {
	// GetBasicMetrics 返回基础统计信息
	GetBasicMetrics() ReceiverBasicMetrics
	// GetCustomMetrics 返回自定义指标
	GetCustomMetrics() []CustomMetric
}

type ReceiverBasicMetrics struct {
	Pipeline         string
	ReceiverType     string
	ReceivedTotal    float64
	ReceivedFailures float64
	LastReceiveTime  time.Time
}
