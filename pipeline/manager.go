package pipeline

import (
	"context"
	"fmt"
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
	ID        string    // Pipeline 唯一标识
	Name      string    // Pipeline 名称
	Status    Status    // Pipeline 状态
	StartTime time.Time // 启动时间
	Error     error     // 最后一次错误
}

// Manager 管理多个 Pipeline
type Manager struct {
	pipelines map[string]*Pipeline // Pipeline 映射表
	status    map[string]*Info     // Pipeline 状态信息
	mu        sync.RWMutex         // 读写锁
	ctx       context.Context      // 上下文
	cancel    context.CancelFunc   // 取消函数
}

// NewPipelineManager 创建新的 Pipeline 管理器
func NewPipelineManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		pipelines: make(map[string]*Pipeline),
		status:    make(map[string]*Info),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RegisterPipeline 注册一个新的 Pipeline
func (pm *Manager) RegisterPipeline(id string, name string, pipeline *Pipeline) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.pipelines[id]; exists {
		return fmt.Errorf("pipeline %s 已存在", id)
	}

	pm.pipelines[id] = pipeline
	pm.status[id] = &Info{
		ID:     id,
		Name:   name,
		Status: StatusCreated,
	}

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
	if err := pipeline.Start(); err != nil {
		pm.updateStatus(id, StatusFailed, err)
		return fmt.Errorf("启动 pipeline %s 失败: %w", id, err)
	}

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
	if info.Status != StatusRunning {
		pm.mu.Unlock()
		return fmt.Errorf("pipeline %s 未在运行", id)
	}

	info.Status = StatusStopping
	pm.mu.Unlock()

	// 停止 Pipeline
	if err := pipeline.Stop(); err != nil {
		pm.updateStatus(id, StatusFailed, err)
		return fmt.Errorf("停止 pipeline %s 失败: %w", id, err)
	}

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
func (pm *Manager) GetStatus(id string) (*Info, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info, exists := pm.status[id]
	if !exists {
		return nil, fmt.Errorf("pipeline %s 不存在", id)
	}

	// 返回副本
	return &Info{
		ID:        info.ID,
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
			ID:        info.ID,
			Name:      info.Name,
			Status:    info.Status,
			StartTime: info.StartTime,
			Error:     info.Error,
		})
	}
	return result
}

// monitorPipeline 监控 Pipeline 的错误
func (pm *Manager) monitorPipeline(id string, pipeline *Pipeline) {
	for {
		select {
		case err := <-pipeline.Errors():
			if err != nil {
				pm.updateStatus(id, StatusFailed, err)
				return
			}
		case <-pm.ctx.Done():
			return
		}
	}
}

// updateStatus 更新 Pipeline 状态
func (pm *Manager) updateStatus(id string, status Status, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if info, exists := pm.status[id]; exists {
		info.Status = status
		info.Error = err
	}
}

// Shutdown 关闭管理器
func (pm *Manager) Shutdown() error {
	pm.cancel() // 取消所有监控
	return pm.StopAll()
}
