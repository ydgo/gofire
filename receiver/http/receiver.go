package http

import (
	"context"
	"errors"
	"fmt"
	"gofire/component"
	"gofire/metrics"
	"gofire/pkg/logger"
	"io"
	"log"
	"net/http"
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
	output   chan map[string]interface{}
	stats    *Stats
	metrics  *metrics.Collector
	mu       sync.RWMutex

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
func NewReceiver(config map[string]interface{}, collector *metrics.Collector) (component.Receiver, error) {
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
	pipeline := config["@pipeline"].(string)
	receiver := &Receiver{
		pipeline:           pipeline, // todo 怎么拿到这个 id ？
		ctx:                ctx,
		cancel:             cancel,
		output:             make(chan map[string]interface{}, cfg.maxPendingRequests),
		port:               cfg.Port,
		path:               cfg.Path,
		maxPendingRequests: cfg.maxPendingRequests,
		metrics:            collector,
		stats: &Stats{
			ReceiverBasicMetrics: metrics.ReceiverBasicMetrics{
				Pipeline:     pipeline,
				ReceiverType: "http",
			},
			statusCodes: make(map[int]float64),
		},
	}

	// 创建 HTTP 服务器
	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Path, receiver.handleRequest)

	receiver.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
	// 更新初始统计信息
	receiver.metrics.UpdateReceiverMetrics(receiver.pipeline, "http", receiver.stats)
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
	close(r.output)
	r.metrics.RemoveReceiverMetrics(r.pipeline, "http")
	return err

}

// handleRequest 处理 HTTP 请求
func (r *Receiver) handleRequest(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	// 确保是 POST 请求
	if req.Method != http.MethodPost {
		r.updateStats(http.StatusMethodNotAllowed, startTime)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.updateStats(http.StatusBadRequest, startTime)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// todo codec text|json

	data := map[string]interface{}{
		"message": string(body),
		"host":    req.RemoteAddr,
	}

	// 尝试发送数据到输出通道
	select {
	case r.output <- data:
		r.updateStats(http.StatusOK, startTime)
		w.WriteHeader(http.StatusOK)

	case <-time.After(5 * time.Second): // 添加超时处理
		r.updateStats(http.StatusServiceUnavailable, startTime)
		http.Error(w, "Pipeline busy", http.StatusServiceUnavailable)
	}
}

func (r *Receiver) ReadMessage(ctx context.Context) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		logger.Debug(18)
		return nil, ctx.Err()
	case data, ok := <-r.output:
		if !ok {
			return nil, errors.New("receiver has been shut down")
		}
		return data, nil

	}

}

// updateStats 更新统计信息
func (r *Receiver) updateStats(statusCode int, startTime time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// todo 考虑比锁更好的实现方式，比如原子指令
	// 更新基础统计信息
	r.stats.ReceivedTotal++
	if statusCode >= 400 {
		r.stats.ReceivedFailures++
	}
	r.stats.LastReceiveTime = time.Now()

	// 更新 HTTP 特定统计信息
	r.stats.statusCodes[statusCode]++
	r.stats.requestLatency = time.Since(startTime).Seconds()

	// 更新收集器
	r.metrics.UpdateReceiverMetrics(r.pipeline, "http", r.stats)
}

// 确保 HTTPReceiver 实现了 component.Receiver 接口
var _ component.Receiver = (*Receiver)(nil)
