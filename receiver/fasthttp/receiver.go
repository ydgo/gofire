package fasthttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"gofire/component"
	"gofire/event"
	"gofire/pkg/logger"
	"io"
	"net/http"
	"time"
)

func init() {
	component.RegisterReceiver("fasthttp", NewReceiver)
}

type Receiver struct {
	engine *fasthttp.Server
	cancel context.CancelFunc
	events chan *event.Event

	errChan chan error
}

func NewReceiver(opts component.ReceiverOpts) (component.Receiver, error) {
	if opts.ComponentType != "fasthttp" {
		return nil, errors.New("fasthttp receiver requires component type `fasthttp`")
	}
	port, ok := opts.Setting["port"].(int)
	if !ok {
		port = 8080
	}
	queueCapacity, ok := opts.Setting["queue_capacity"].(uint)
	if !ok {
		queueCapacity = 500
	}
	receiver := &Receiver{
		events:  make(chan *event.Event, queueCapacity),
		errChan: make(chan error, 1),
	}
	receiver.engine = &fasthttp.Server{
		Name:             "Go Fire",
		Handler:          receiver.HandleRequest,
		ErrorHandler:     receiver.HandlerError,
		LogAllErrors:     true,
		TCPKeepalive:     true,
		DisableKeepalive: false,
	}
	go func() {
		defer close(receiver.events)
		defer close(receiver.errChan)
		if err := receiver.engine.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
			logger.Warnf("%s/%s ListenAndServe: %s", opts.Pipeline, opts.ComponentType, err.Error())
			receiver.errChan <- err
		}
	}()
	return receiver, nil
}

func (r *Receiver) HandleRequest(ctx *fasthttp.RequestCtx) {
	defer func() {
		if err := recover(); err != nil {
			logger.Warnf("handle panic: %s", err)
		}
	}()
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	body := ctx.PostBody()
	if len(body) == 0 {
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}
	evt := event.NewEvent()
	evt.SetMessage(string(body))

	// 尝试发送数据到输出通道
	select {
	case r.events <- evt:
		ctx.SetStatusCode(http.StatusOK)
	case <-time.After(time.Second * 1):
		evt.Release() // 释放 event
		ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
	}
}

func (r *Receiver) HandlerError(_ *fasthttp.RequestCtx, err error) {
	logger.Warnf("handle error: %s", err.Error())
}

func (r *Receiver) ReadMessage(ctx context.Context) (evt *event.Event, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err = <-r.errChan:
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	case data, ok := <-r.events:
		if !ok {
			return nil, io.EOF
		}
		return data, nil

	}
}

func (r *Receiver) Shutdown() error {
	return r.engine.Shutdown() // 停止接收数据
}
