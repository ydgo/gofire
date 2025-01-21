package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"gofire/component"
	"gofire/metrics"
	"gofire/pipeline"
	"gofire/pkg/logger"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// PipelineConfig 表示单个 Pipeline 的配置
type PipelineConfig struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Receiver struct {
		Type   string                 `yaml:"type"`
		Config map[string]interface{} `yaml:"config"`
	} `yaml:"receiver"`
	Processors []struct {
		Type   string                 `yaml:"type"`
		Config map[string]interface{} `yaml:"config"`
	} `yaml:"processors"`
	Exporters []struct {
		Type   string                 `yaml:"type"`
		Config map[string]interface{} `yaml:"config"`
	} `yaml:"exporters"`
}

// Manager 管理 Pipeline 配置
type Manager struct {
	configDir   string
	pipelineMgr *pipeline.Manager
	watcher     *fsnotify.Watcher
	configs     map[string]*PipelineConfig
	mu          sync.RWMutex
	factory     *component.Factory // 用于创建组件的工厂
	metrics     *metrics.ConfigMetrics
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(configDir string, pipelineMgr *pipeline.Manager, factory *component.Factory) (*Manager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建文件监控失败: %w", err)
	}

	cm := &Manager{
		configDir:   configDir,
		pipelineMgr: pipelineMgr,
		watcher:     watcher,
		configs:     make(map[string]*PipelineConfig),
		factory:     factory,
		metrics:     factory.Metrics().ConfigMetrics(),
	}

	return cm, nil
}

// Start 启动配置管理器
func (m *Manager) Start() error {
	// 首次加载所有配置
	if err := m.loadAllConfigs(); err != nil {
		return err
	}

	// 监控配置目录
	if err := m.watcher.Add(m.configDir); err != nil {
		return fmt.Errorf("监控配置目录失败: %w", err)
	}

	// 启动监控协程
	go m.watchConfig()
	return nil
}

// loadAllConfigs 加载所有配置文件
func (m *Manager) loadAllConfigs() error {
	files, err := filepath.Glob(filepath.Join(m.configDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	m.metrics.Total.Set(float64(len(files)))
	// 记录新的配置
	newConfigs := make(map[string]*PipelineConfig)

	// 加载每个配置文件
	for _, file := range files {
		config, err := m.loadConfigFile(file)
		if err != nil {
			logger.Warnf("加载配置文件 %s 失败: %v", file, err)
			continue
		}
		newConfigs[config.ID] = config
	}

	// 对比新旧配置，进行更新
	m.updatePipelines(newConfigs)
	return nil
}

// loadConfigFile 加载单个配置文件
func (m *Manager) loadConfigFile(filename string) (*PipelineConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config PipelineConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// updatePipelines 更新 Pipeline
func (m *Manager) updatePipelines(newConfigs map[string]*PipelineConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止已删除的 Pipeline
	for id := range m.configs {
		if _, exists := newConfigs[id]; !exists {
			logger.Infof("停止已删除的 Pipeline: %s", id)
			if err := m.pipelineMgr.StopPipeline(id); err != nil {
				logger.Warnf("停止 Pipeline %s 失败: %v", id, err)
			}
		}
	}

	// 更新或创建 Pipeline
	for id, newConfig := range newConfigs {
		oldConfig, exists := m.configs[id]
		if !exists {
			// 创建新的 Pipeline
			if err := m.createPipeline(newConfig); err != nil {
				logger.Infof("创建 Pipeline %s 失败: %v", id, err)
				continue
			}
		} else if !m.configEquals(oldConfig, newConfig) {
			// todo  id 修改为 name
			m.metrics.Reload.With(prometheus.Labels{"name": id}).Inc() // 重载指标
			// 配置发生变化，重新创建 Pipeline
			logger.Infof("更新 Pipeline: %s", id)
			if err := m.pipelineMgr.StopPipeline(id); err != nil {
				logger.Warnf("停止旧 Pipeline %s 失败: %v", id, err)
			}
			if err := m.createPipeline(newConfig); err != nil {
				logger.Warnf("创建新 Pipeline %s 失败: %v", id, err)
				continue
			}
		}
	}

	// 更新配置缓存
	m.configs = newConfigs
}

// createPipeline 根据配置创建 Pipeline
func (m *Manager) createPipeline(config *PipelineConfig) error {
	// 创建 Receiver
	receiver, err := m.factory.CreateReceiver(config.Name, config.Receiver.Type, config.Receiver.Config)
	if err != nil {
		return fmt.Errorf("创建 Receiver 失败: %w", err)
	}

	// 创建 Processors
	var processors []component.Processor
	for _, p := range config.Processors {
		processor, err := m.factory.CreateProcessor(p.Type, p.Config)
		if err != nil {
			return fmt.Errorf("创建 Processor 失败: %w", err)
		}
		processors = append(processors, processor)
	}

	// 创建 Exporters
	var exporters []component.Exporter
	for _, e := range config.Exporters {
		exporter, err := m.factory.CreateExporter(e.Type, e.Config)
		if err != nil {
			return fmt.Errorf("创建 Exporter 失败: %w", err)
		}
		exporters = append(exporters, exporter)
	}

	// 创建 Pipeline
	pipe := pipeline.NewPipeline(
		receiver,
		component.BuildProcessorLink(processors...),
		component.BuildExporters(exporters...),
	)

	// 注册并启动 Pipeline
	if err = m.pipelineMgr.RegisterPipeline(config.ID, config.Name, pipe); err != nil {
		return fmt.Errorf("注册 Pipeline 失败: %w", err)
	}

	if err = m.pipelineMgr.StartPipeline(config.ID); err != nil {
		return fmt.Errorf("启动 Pipeline 失败: %w", err)
	}

	return nil
}

// watchConfig 监控配置文件变化
func (m *Manager) watchConfig() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				// 配置文件发生变化，重新加载所有配置
				if err := m.loadAllConfigs(); err != nil {
					log.Printf("重新加载配置失败: %v", err)
				}
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("配置监控错误: %v", err)
		}
	}
}

// configEquals 比较两个配置是否相同
func (m *Manager) configEquals(a, b *PipelineConfig) bool {
	if a == nil || b == nil {
		return a == b
	}

	// 比较基本字段
	if a.ID != b.ID || a.Name != b.Name {
		return false
	}

	// 比较 Receiver
	if a.Receiver.Type != b.Receiver.Type ||
		!compareConfig(a.Receiver.Config, b.Receiver.Config) {
		return false
	}

	// 比较 Processors
	if len(a.Processors) != len(b.Processors) {
		return false
	}
	for i := range a.Processors {
		if a.Processors[i].Type != b.Processors[i].Type ||
			!compareConfig(a.Processors[i].Config, b.Processors[i].Config) {
			return false
		}
	}

	// 比较 Exporters
	if len(a.Exporters) != len(b.Exporters) {
		return false
	}
	for i := range a.Exporters {
		if a.Exporters[i].Type != b.Exporters[i].Type ||
			!compareConfig(a.Exporters[i].Config, b.Exporters[i].Config) {
			return false
		}
	}

	return true
}

// compareConfig 比较两个配置映射是否相同
func compareConfig(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for k, va := range a {
		vb, exists := b[k]
		if !exists {
			return false
		}

		// 处理嵌套的 map
		if ma, okA := va.(map[string]interface{}); okA {
			if mb, okB := vb.(map[string]interface{}); okB {
				if !compareConfig(ma, mb) {
					return false
				}
				continue
			}
			return false
		}

		// 处理切片
		if sa, okA := va.([]interface{}); okA {
			sb, okB := vb.([]interface{})
			if !okB || len(sa) != len(sb) {
				return false
			}
			// 比较切片中的每个元素
			for i := range sa {
				if !compareValue(sa[i], sb[i]) {
					return false
				}
			}
			continue
		}

		// 比较其他类型的值
		if !compareValue(va, vb) {
			return false
		}
	}

	return true
}

// compareValue 比较两个值是否相同
func compareValue(a, b interface{}) bool {
	// 处理数字类型的特殊情况
	// YAML 解析可能会将整数解析为不同的数字类型
	switch va := a.(type) {
	case int:
		switch vb := b.(type) {
		case int:
			return va == vb
		case int64:
			return int64(va) == vb
		case float64:
			return float64(va) == vb
		}
	case int64:
		switch vb := b.(type) {
		case int:
			return va == int64(vb)
		case int64:
			return va == vb
		case float64:
			return float64(va) == vb
		}
	case float64:
		switch vb := b.(type) {
		case int:
			return va == float64(vb)
		case int64:
			return va == float64(vb)
		case float64:
			return va == vb
		}
	}

	// 其他类型直接比较
	return a == b
}

// Stop 停止配置管理器
func (m *Manager) Stop() error {
	return m.watcher.Close()
}
