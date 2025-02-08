package pipeline

import (
	"context"
	"errors"
	"fmt"
	"gofire/component"
	"gofire/event"
	"gofire/metrics"
	"gofire/pkg/logger"
	"io"
	"sync"
	"time"
)

// Pipeline 代表一个完整的数据处理管道
type Pipeline struct {
	name          string
	receiver      component.Receiver
	processorLink *component.ProcessorLink
	exporters     component.Exporter
	metrics       *metrics.PipelineMetrics

	// 控制和状态
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	errChan       chan error
	batchSize     int           // 批处理大小
	flushInterval time.Duration // 强制刷新间隔

	stopOnce sync.Once    // 确保 Stop 只被执行一次
	stopped  bool         // 标记是否已经停止
	stopLock sync.RWMutex // 保护 stopped 标志
}

// NewPipeline 创建新的处理管道
func NewPipeline(name string, collector *metrics.Collector, receiver component.Receiver, processorLink *component.ProcessorLink, exporters component.Exporter) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pipeline{
		name:          name,
		receiver:      receiver,
		processorLink: processorLink,
		exporters:     exporters,
		ctx:           ctx,
		cancel:        cancel,
		errChan:       make(chan error, 1),
		batchSize:     1000,            // 默认批处理大小
		flushInterval: time.Second * 5, // 默认5秒强制刷新
		stopOnce:      sync.Once{},
		stopped:       false,
		metrics:       collector.PipelineMetrics(),
	}
}

// SetBatchSize 设置批处理大小
func (p *Pipeline) SetBatchSize(size int) {
	p.batchSize = size
}

// SetFlushInterval 设置刷新间隔
func (p *Pipeline) SetFlushInterval(interval time.Duration) {
	p.flushInterval = interval
}

// Start 启动处理管道
func (p *Pipeline) Start() error {
	p.wg.Add(1)
	go p.run()
	return nil
}

// run 运行主处理循环
func (p *Pipeline) run() {
	defer p.wg.Done()
	for {
		message, err := p.receiver.ReadMessage(p.ctx)
		if err != nil {
			if !(errors.Is(err, context.Canceled) || errors.Is(err, io.EOF)) {
				p.errChan <- fmt.Errorf("读取消息错误: %w", err)
			}
			close(p.errChan)
			return
		}
		p.processEvent(message)
	}
}

// process 处理一批数据
func (p *Pipeline) processEvent(evt *event.Event) {
	processed, err := p.processorLink.Process(evt) // evt 的释放由 Process 决定，如果仍需使用请复制一份
	if err != nil {
		logger.Warnf("处理消息错误: %s", err)
		p.metrics.AddFailure(p.name, 1)
		return
	}
	p.metrics.AddProcessed(p.name, 1)

	// 导出处理后的数据
	// processed 中的 event 由 Export 负责释放
	for _, e := range processed {
		if err = p.exporters.Export(e); err != nil {
			logger.Warnf("导出消息错误: %s", err)
			p.metrics.AddFailure(p.name, 1)
			return
		}
	}
	// todo 这里的不包含 split 生成的数据
	p.metrics.AddOut(p.name, 1)
}

// Stop 停止处理管道
func (p *Pipeline) Stop() error {
	defer p.metrics.Delete(p.name)
	// 检查是否已经停止
	p.stopLock.RLock()
	if p.stopped {
		p.stopLock.RUnlock()
		return fmt.Errorf("pipeline已经停止")
	}
	p.stopLock.RUnlock()

	var shutdownErr error
	p.stopOnce.Do(func() {
		// 标记为已停止
		p.stopLock.Lock()
		p.stopped = true
		p.stopLock.Unlock()

		// 首先关闭 receiver，停止接收新数据
		if err := p.receiver.Shutdown(); err != nil {
			shutdownErr = fmt.Errorf("关闭receiver错误: %w", err)
		}

		// 等待 run 循环完成
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		// 设置超时时间
		timeout := time.After(30 * time.Second)
		select {
		case <-done:
			// 正常关闭
		case <-timeout:
			// 发送取消信号以停止处理循环
			p.cancel()
			if shutdownErr != nil {
				shutdownErr = fmt.Errorf("pipeline关闭超时且receiver关闭失败: %w", shutdownErr)
				return
			}
			shutdownErr = fmt.Errorf("pipeline关闭超时")
			return
		}

		// 最后关闭 exporters
		if err := p.exporters.Shutdown(); err != nil {
			if shutdownErr != nil {
				shutdownErr = fmt.Errorf("关闭exporters错误: %w; 之前的错误: %w", err, shutdownErr)
				return
			}
			shutdownErr = fmt.Errorf("关闭exporters错误: %w", err)
		}
	})
	return shutdownErr
}

// IsRunning 返回 pipeline 是否正在运行
func (p *Pipeline) IsRunning() bool {
	p.stopLock.RLock()
	defer p.stopLock.RUnlock()
	return !p.stopped
}

// Errors 返回错误通道
func (p *Pipeline) Errors() <-chan error {
	return p.errChan
}
