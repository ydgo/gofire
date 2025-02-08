package pipeline

import (
	"context"
	"errors"
	"fmt"
	"gofire/component"
	"gofire/event"
	"gofire/pkg/logger"
	"io"
	"sync"
	"time"
)

// Pipeline 代表一个完整的数据处理管道
type Pipeline struct {
	name          string // 根据业务命名
	receiver      component.Receiver
	processorLink *component.ProcessorLink
	exporters     component.Exporter

	// 控制和状态
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error

	stopOnce sync.Once    // 确保 Stop 只被执行一次
	stopped  bool         // 标记是否已经停止
	stopLock sync.RWMutex // 保护 stopped 标志
}

// NewPipeline 创建新的处理管道
func NewPipeline(name string, receiver component.Receiver, processorLink *component.ProcessorLink, exporters component.Exporter) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pipeline{
		name:          name,
		receiver:      receiver,
		processorLink: processorLink,
		exporters:     exporters,
		ctx:           ctx,
		cancel:        cancel,
		errChan:       make(chan error, 1),
		stopOnce:      sync.Once{},
		stopped:       false,
	}
}

// Start 启动处理管道
func (p *Pipeline) Start() {
	p.wg.Add(1)
	go p.run()
}

// run 运行主处理循环
func (p *Pipeline) run() {
	defer p.wg.Done()
	for {
		message, err := p.receiver.ReadMessage(p.ctx)
		if err != nil {
			if !(errors.Is(err, context.Canceled) || errors.Is(err, io.EOF)) {
				p.errChan <- fmt.Errorf("读取消息错误: %w", err) // 报告 pipeline 出现的错误给 pipeline manager
			}
			close(p.errChan)
			return
		}
		if err = p.processEvent(message); err != nil {
			logger.Warn(err)
		}
	}
}

// process 处理一批数据
func (p *Pipeline) processEvent(evt *event.Event) error {
	processed, err := p.processorLink.Process(evt) // evt 的释放由 Process 决定，如果仍需使用请复制一份
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}

	// 导出处理后的数据
	// processed 中的 event 由 Export 负责释放
	for _, e := range processed {
		if err = p.exporters.Export(e); err != nil {
			return fmt.Errorf("exporter: %w", err)
		}
	}
	return nil
}

// Stop 停止处理管道
func (p *Pipeline) Stop() error {
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

		// 关闭 receiver
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

// Errors 返回错误通道
func (p *Pipeline) Errors() <-chan error {
	return p.errChan
}
