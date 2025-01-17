package pipeline

import (
	"context"
	"fmt"
	"gofire/component"
	"sync"
	"time"
)

// Pipeline 代表一个完整的数据处理管道
type Pipeline struct {
	receiver      component.Receiver
	processorLink *component.ProcessorLink
	exporters     *component.Exporters

	// 控制和状态
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	errChan       chan error
	batchSize     int           // 批处理大小
	flushInterval time.Duration // 强制刷新间隔
}

// NewPipeline 创建新的处理管道
func NewPipeline(receiver component.Receiver, processorLink *component.ProcessorLink, exporters *component.Exporters) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pipeline{
		receiver:      receiver,
		processorLink: processorLink,
		exporters:     exporters,
		ctx:           ctx,
		cancel:        cancel,
		errChan:       make(chan error, 1),
		batchSize:     1000,            // 默认批处理大小
		flushInterval: time.Second * 5, // 默认5秒强制刷新
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

	batch := make([]map[string]interface{}, 0, p.batchSize)
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// 处理剩余的批次
			if len(batch) > 0 {
				p.processBatch(batch)
			}
			return

		case <-ticker.C:
			// 定时刷新
			if len(batch) > 0 {
				p.processBatch(batch)
				batch = make([]map[string]interface{}, 0, p.batchSize)
			}

		default:
			// 尝试读取消息
			msg, err := p.receiver.ReadMessage(p.ctx)
			if err != nil {
				p.errChan <- fmt.Errorf("读取消息错误: %w", err)
				return
			}

			// 如果返回nil且没有错误，说明已经读取完毕
			if msg == nil {
				if len(batch) > 0 {
					p.processBatch(batch)
				}
				return
			}

			// 添加到批次
			batch = append(batch, msg)
			if len(batch) >= p.batchSize {
				p.processBatch(batch)
				batch = make([]map[string]interface{}, 0, p.batchSize)
			}
		}
	}
}

// processBatch 处理一批数据
func (p *Pipeline) processBatch(batch []map[string]interface{}) {
	// 处理数据
	for _, msg := range batch {
		processed, err := p.processorLink.Process(msg)
		if err != nil {
			p.errChan <- fmt.Errorf("处理消息错误: %w", err)
			continue
		}

		// 导出处理后的数据
		if err = p.exporters.Export(processed...); err != nil {
			p.errChan <- fmt.Errorf("导出消息错误: %w", err)
			continue
		}
	}
}

// Stop 停止处理管道
func (p *Pipeline) Stop() error {
	// 发送取消信号
	p.cancel()

	// 等待处理完成
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
		return fmt.Errorf("pipeline关闭超时")
	}

	// 关闭组件
	if err := p.receiver.Shutdown(); err != nil {
		return fmt.Errorf("关闭receiver错误: %w", err)
	}

	if err := p.exporters.Shutdown(); err != nil {
		return fmt.Errorf("关闭exporters错误: %w", err)
	}

	return nil
}

// Errors 返回错误通道
func (p *Pipeline) Errors() <-chan error {
	return p.errChan
}
