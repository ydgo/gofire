package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gofire/component"
	"gofire/config"
	_ "gofire/exporter"
	"gofire/metrics"
	"gofire/pipeline"
	"gofire/pkg/logger"
	_ "gofire/processor"
	_ "gofire/receiver"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	configDir string
	debug     bool
)

func init() {
	flag.StringVar(&configDir, "config.path", "/etc/gofire/conf.d", "config directory")
	flag.BoolVar(&debug, "debug", false, "debug mode")
}

func main() {
	flag.Parse()
	if debug {
		logger.SetLevel(logger.DebugLevel)
	}

	prometheus.MustRegister(metrics.Default())
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	factory := component.DefaultFactory()
	logger.Debugf("Support receivers: %v", factory.GetReceiverTypes())
	logger.Debugf("Support processors: %v", factory.GetProcessorTypes())
	logger.Debugf("Support exporters: %v", factory.GetExporterTypes())

	pipelineMgr := pipeline.NewPipelineManager(metrics.Default())
	logger.Debug("Pipeline manager created")
	configMgr, err := config.NewConfigManager(configDir, pipelineMgr, factory)
	if err != nil {
		logger.Fatalf("Create config manager failed: %v", err)
	}
	logger.Debug("Config manager created")
	// 启动配置管理器
	if err = configMgr.Start(); err != nil {
		logger.Fatalf("Start config manager failed: %v", err)
	}
	logger.Debug("Config manager started")
	defer configMgr.Stop()

	// 启动 metrics 服务
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Infof("Metrics api listening on :9090")
		if err = http.ListenAndServe(":9090", nil); err != nil {
			logger.Warnf("Start metrics server failed: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Info("Shutting down gracefully")
	if err = pipelineMgr.Shutdown(); err != nil {
		logger.Warnf("Shutdown pipeline manager failed: %v", err)
	}
	logger.Info("Shutdown")
}
