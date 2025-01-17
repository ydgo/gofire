package receiver

import (
	"bufio"
	"context"
	"fmt"
	"gofire/component"
	"os"
	"sync"
	"time"
)

func init() {
	component.RegisterReceiver("file", NewFileReceiver)
}

// FileReceiver 实现了从文件读取数据的接收器
type FileReceiver struct {
	filePath string
	file     *os.File
	scanner  *bufio.Scanner
	mutex    sync.Mutex
	done     chan struct{}
}

// NewFileReceiver 创建新的文件接收器
func NewFileReceiver(config map[string]interface{}) (component.Receiver, error) {
	filePath, ok := config["filePath"].(string)
	if !ok {
		return nil, fmt.Errorf("must specify filePath")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}

	return &FileReceiver{
		filePath: filePath,
		file:     file,
		scanner:  bufio.NewScanner(file),
		done:     make(chan struct{}),
	}, nil
}

// ReadMessage 实现 Receiver 接口，按行读取文件内容
func (f *FileReceiver) ReadMessage(ctx context.Context) (map[string]interface{}, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-f.done:
		return nil, fmt.Errorf("receiver已关闭")
	default:
	}

	// 读取下一行
	if f.scanner.Scan() {
		line := f.scanner.Text()
		// 将行内容包装成 map
		return map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"source":    f.filePath,
			"message":   line,
		}, nil
	}

	if err := f.scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 文件读取完毕
	return nil, nil
}

// Shutdown 实现 Receiver 接口，关闭文件
func (f *FileReceiver) Shutdown() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	select {
	case <-f.done:
		return nil
	default:
		close(f.done)
	}

	if err := f.file.Close(); err != nil {
		return fmt.Errorf("关闭文件失败: %w", err)
	}
	return nil
}
