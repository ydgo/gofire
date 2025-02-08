package component

import (
	"context"
	"gofire/event"
)

type Receiver interface {
	// ReadMessage from receiver
	// 阻塞地读取一条消息，当 context 取消或者 receiver 关闭时返回 nil，且 error 不为空
	// 并发安全
	ReadMessage(ctx context.Context) (*event.Event, error)
	// Shutdown receiver
	// 释放资源
	Shutdown() error
}
