package fasthttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"gofire/component"
	"gofire/event"
	"gofire/metrics"
	"gofire/pkg/logger"
	"net/http"
	"sync"
	"time"
)

func init() {
	component.RegisterReceiver("fasthttp", NewReceiver)
}

type Receiver struct {
	mu       sync.RWMutex
	pipeline string
	metrics  *metrics.ReceiverBasicMetrics
	engine   *fasthttp.Server
	ctx      context.Context
	cancel   context.CancelFunc

	output chan *event.Event

	port int
}

func NewReceiver(pipeName string, config map[string]interface{}, collector *metrics.Collector) (component.Receiver, error) {
	port, ok := config["port"].(int)
	if !ok {
		return nil, fmt.Errorf("port must be specified")
	}
	ctx, cancel := context.WithCancel(context.Background())
	receiver := &Receiver{
		ctx:      ctx,
		cancel:   cancel,
		pipeline: pipeName,
		metrics:  collector.ReceiverBasicMetrics(),
		port:     port,
		output:   make(chan *event.Event, 100),
	}
	receiver.engine = &fasthttp.Server{
		Name:             "gofire",
		Handler:          receiver.HandleRequest,
		ErrorHandler:     receiver.HandlerError,
		LogAllErrors:     true,
		TCPKeepalive:     true,
		DisableKeepalive: false,
	}
	go func() {
		_ = receiver.engine.ListenAndServe(fmt.Sprintf(":%d", receiver.port))
	}()
	return receiver, nil
}

func (r *Receiver) HandleRequest(ctx *fasthttp.RequestCtx) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("handle panic: %s", err)
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

	r.metrics.IncTotal(r.pipeline, "fasthttp") // 指标更新
	// 尝试发送数据到输出通道
	select {
	case r.output <- evt:
		ctx.SetStatusCode(http.StatusOK)
	case <-time.After(time.Second * 1):
		evt.Release()                                 // 释放 event
		r.metrics.IncFailures(r.pipeline, "fasthttp") // 指标更新
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
	}
}

func (r *Receiver) HandlerError(ctx *fasthttp.RequestCtx, err error) {
	logger.Warnf("handle error: %s", err.Error())
	r.metrics.IncFailures(r.pipeline, "fasthttp")
}

func (r *Receiver) ReadMessage(ctx context.Context) (evt *event.Event, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-r.output:
		if !ok {
			return nil, errors.New("receiver has been shut down")
		}
		return data, nil

	}
}

func (r *Receiver) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case <-r.ctx.Done():
		return errors.New("receiver has been shut down")
	default:
		r.cancel()
	}
	err := r.engine.Shutdown()
	r.metrics.Delete(r.pipeline)
	close(r.output)
	return err
}
