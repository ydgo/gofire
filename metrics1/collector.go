package metrics1

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

// Collector 管理所有指标的收集和注册
type Collector struct {
	// prometheus 注册表
	registry  *prometheus.Registry
	processor ProcessorCollector

	// 防止并发访问的锁
	mu sync.RWMutex
}
