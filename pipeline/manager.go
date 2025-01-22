package pipeline

import (
	"context"
	"fmt"
	"gofire/metrics"
	"gofire/pkg/logger"
	"sync"
	"time"
)

// Status 表示 Pipeline 的状态
type Status string

const (
	StatusCreated  Status = "created"  // 已创建
	StatusRunning  Status = "running"  // 运行中
	StatusStopping Status = "stopping" // 正在停止
	StatusStopped  Status = "stopped"  // 已停止
	StatusFailed   Status = "failed"   // 运行失败
)

// Info 包含 Pipeline 的信息
type Info struct {
	Name      string    // Pipeline 名称
	Status    Status    // Pipeline 状态
	StartTime time.Time // 启动时间
	Error     error     // 最后一次错误
}

// Manager 管理多个 Pipeline
type Manager struct {
	metrics   *metrics.PipelineManagerMetrics
	pipelines map[string]*Pipeline // Pipeline 映射表
	status    map[string]*Info     // Pipeline 状态信息
	mu        sync.RWMutex         // 读写锁
	ctx       context.Context      // 上下文
	cancel    context.CancelFunc   // 取消函数
}

// NewPipelineManager 创建新的 Pipeline 管理器
func NewPipelineManager(collector *metrics.Collector) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		metrics:   collector.PipelineManagerMetrics(),
		pipelines: make(map[string]*Pipeline),
		status:    make(map[string]*Info),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RegisterPipeline 注册一个新的 Pipeline
func (pm *Manager) RegisterPipeline(name string, pipeline *Pipeline) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.pipelines[name]; exists {
		return fmt.Errorf("pipeline %s 已存在", name)
	}

	pm.pipelines[name] = pipeline
	pm.status[name] = &Info{
		Name:   name,
		Status: StatusCreated,
	}
	pm.metrics.IncPipelineTotal()
	return nil
}

// StartPipeline 启动指定的 Pipeline
func (pm *Manager) StartPipeline(id string) error {
	pm.mu.Lock()
	pipeline, exists := pm.pipelines[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("pipeline %s 不存在", id)
	}

	info := pm.status[id]
	if info.Status == StatusRunning {
		pm.mu.Unlock()
		return fmt.Errorf("pipeline %s 已在运行", id)
	}

	info.Status = StatusRunning
	info.StartTime = time.Now()
	info.Error = nil
	pm.mu.Unlock()

	// 启动 Pipeline 并监控错误
	_ = pipeline.Start()
	// 监控 Pipeline 错误
	go pm.monitorPipeline(id, pipeline)

	return nil
}

// StartAll 启动所有 Pipeline
func (pm *Manager) StartAll() error {
	pm.mu.RLock()
	pipelineIDs := make([]string, 0, len(pm.pipelines))
	for id := range pm.pipelines {
		pipelineIDs = append(pipelineIDs, id)
	}
	pm.mu.RUnlock()

	var errors []error
	for _, id := range pipelineIDs {
		if err := pm.StartPipeline(id); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("启动部分 pipeline 失败: %v", errors)
	}
	return nil
}

// StopPipeline 停止指定的 Pipeline
func (pm *Manager) StopPipeline(id string) error {
	pm.mu.Lock()
	pipeline, exists := pm.pipelines[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("pipeline %s 不存在", id)
	}
	info := pm.status[id]
	if info.Status != StatusRunning && info.Status != StatusFailed {
		pm.mu.Unlock()
		return fmt.Errorf("pipeline %s 状态 [%s] 错误", id, info.Status)
	}
	info.Status = StatusStopping
	// 停止 Pipeline
	if err := pipeline.Stop(); err != nil {
		logger.Warnf("停止 pipeline %s 异常: %s", id, err)
	}
	pm.mu.Unlock()
	pm.updateStatus(id, StatusStopped, nil)
	return nil
}

// StopAll 停止所有 Pipeline
func (pm *Manager) StopAll() error {
	pm.mu.RLock()
	pipelineIDs := make([]string, 0, len(pm.pipelines))
	for id := range pm.pipelines {
		pipelineIDs = append(pipelineIDs, id)
	}
	pm.mu.RUnlock()

	var errors []error
	for _, id := range pipelineIDs {
		if err := pm.StopPipeline(id); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("停止部分 pipeline 失败: %v", errors)
	}
	return nil
}

// GetStatus 获取指定 Pipeline 的状态
func (pm *Manager) GetStatus(name string) (*Info, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info, exists := pm.status[name]
	if !exists {
		return nil, fmt.Errorf("pipeline %s 不存在", name)
	}

	// 返回副本
	return &Info{
		Name:      info.Name,
		Status:    info.Status,
		StartTime: info.StartTime,
		Error:     info.Error,
	}, nil
}

// ListPipelines 列出所有 Pipeline 的状态
func (pm *Manager) ListPipelines() []*Info {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*Info, 0, len(pm.status))
	for _, info := range pm.status {
		// 返回副本
		result = append(result, &Info{
			Name:      info.Name,
			Status:    info.Status,
			StartTime: info.StartTime,
			Error:     info.Error,
		})
	}
	return result
}

// monitorPipeline 监控 Pipeline 的错误
func (pm *Manager) monitorPipeline(name string, pipeline *Pipeline) {
	for {
		select {
		case err := <-pipeline.Errors():
			pm.updateStatus(name, StatusFailed, err)
			if err = pm.StopPipeline(name); err != nil {
				logger.Warnf("停止运行错误的 pipeline %s 错误: %s", name, err)
			} else {
				logger.Warnf("停止运行错误的 pipeline %s 成功", name)
			}
			return

		case <-pm.ctx.Done():
			return
		}
	}
}

// updateStatus 更新 Pipeline 状态
func (pm *Manager) updateStatus(name string, status Status, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if status == StatusStopped {
		delete(pm.pipelines, name)
		delete(pm.status, name)
		pm.metrics.DecPipelineTotal()
		return
	}

	if info, exists := pm.status[name]; exists {
		info.Status = status
		info.Error = err
	}
}

// Shutdown 关闭管理器
func (pm *Manager) Shutdown() error {
	pm.cancel() // 取消所有监控
	return pm.StopAll()
}
