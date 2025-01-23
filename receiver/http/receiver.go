package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"gofire/component"
	"gofire/event"
	"gofire/metrics"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func init() {
	component.RegisterReceiver("http", NewReceiver)
}

// Receiver 实现了基于 HTTP 的数据接收器
type Receiver struct {
	pipeline string
	server   *http.Server
	output   chan *event.Event
	metrics  *metrics.ReceiverBasicMetrics
	mu       sync.RWMutex

	statusCodes *prometheus.CounterVec

	ctx    context.Context
	cancel context.CancelFunc

	// 配置项
	port               int
	path               string
	maxPendingRequests int
}

// Config HTTP 接收器的配置
type Config struct {
	Port               int    `yaml:"port"`
	Path               string `yaml:"path"`
	maxPendingRequests int    `yaml:"batch_size"`
}

// NewReceiver 创建新的 HTTP 接收器
func NewReceiver(pipeName string, config map[string]interface{}, collector *metrics.Collector) (component.Receiver, error) {
	// 解析配置
	var cfg Config

	// 从 map 中获取配置值
	if port, ok := config["port"].(int); ok {
		cfg.Port = port
	}
	if path, ok := config["path"].(string); ok {
		cfg.Path = path
	}
	if maxPendingRequests, ok := config["max_pending_requests"].(int); ok {
		cfg.maxPendingRequests = maxPendingRequests
	}

	// 验证和设置默认值
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("无效的端口号: %d", cfg.Port)
	}

	if cfg.Path == "" {
		cfg.Path = "/receive"
	}

	if cfg.maxPendingRequests <= 0 {
		cfg.maxPendingRequests = 100
	}
	ctx, cancel := context.WithCancel(context.Background())

	// 创建自定义指标
	statusCodes := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "receiver_http_status_code_total",
		Help: "Http receiver response status code total",
	}, []string{"pipeline", "status"})

	receiver := &Receiver{
		pipeline:           pipeName,
		ctx:                ctx,
		cancel:             cancel,
		output:             make(chan *event.Event, cfg.maxPendingRequests),
		port:               cfg.Port,
		path:               cfg.Path,
		maxPendingRequests: cfg.maxPendingRequests,
		metrics:            collector.ReceiverBasicMetrics(),
		statusCodes:        statusCodes,
	}
	// 注册自定义指标
	if err := receiver.metrics.RegisterCustomMetrics("receiver_http_status_code_total", receiver.statusCodes); err != nil {
		return nil, fmt.Errorf("metrics register custom metrics: %w", err)
	}

	// 创建 HTTP 服务器
	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Path, receiver.handleRequest)

	receiver.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
	// 启动 HTTP 服务器
	go func() {
		if err := receiver.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// 实际项目中应该有更好的错误处理机制
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()
	return receiver, nil
}

// Shutdown 实现 component.Receiver 接口
func (r *Receiver) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case <-r.ctx.Done():
		return errors.New("receiver has been shut down")
	default:
		r.cancel()
	}
	err := r.server.Shutdown(context.TODO())
	if err != nil {
		log.Printf("Shutdown error: %v\n", err)
	}
	r.metrics.Delete(r.pipeline)
	close(r.output)
	return err

}

// handleRequest 处理 HTTP 请求
func (r *Receiver) handleRequest(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	r.metrics.SetLastReceiveTime(r.pipeline, "http", startTime)

	// 确保是 POST 请求
	if req.Method != http.MethodPost {
		r.statusCodes.WithLabelValues(r.pipeline, strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.statusCodes.WithLabelValues(r.pipeline, strconv.Itoa(http.StatusBadRequest)).Inc()
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// todo codec text|json

	evt := event.NewEvent()
	evt.SetMessage(string(body))

	r.metrics.IncTotal(r.pipeline, "http") // 指标更新
	// 尝试发送数据到输出通道
	select {
	case r.output <- evt:
		r.statusCodes.WithLabelValues(r.pipeline, strconv.Itoa(http.StatusOK)).Inc()
		w.WriteHeader(http.StatusOK)
	case <-time.After(time.Second * 3):
		evt.Release() // 释放 event
		r.statusCodes.WithLabelValues(r.pipeline, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		r.metrics.IncFailures(r.pipeline, "http") // 指标更新
		http.Error(w, "Pipeline busy", http.StatusServiceUnavailable)
	}
	r.metrics.AddProcessDuration(r.pipeline, "http", time.Since(startTime)) // 指标更新
}

func (r *Receiver) ReadMessage(ctx context.Context) (*event.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-r.output:
		if !ok {
			return nil, errors.New("receiver has been shut down")
		}
		return data, nil

	}

}

// 确保 HTTPReceiver 实现了 component.Receiver 接口
var _ component.Receiver = (*Receiver)(nil)
