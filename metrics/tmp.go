package metrics

import "github.com/prometheus/client_golang/prometheus"

type TmpMetrics struct {
}

func NewTmpMetrics() *TmpMetrics {
	return &TmpMetrics{}
}
func (m *TmpMetrics) Describe(ch chan<- *prometheus.Desc) {

}

func (m *TmpMetrics) Collect(ch chan<- prometheus.Metric) {
}
