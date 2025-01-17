package main

import (
	"gofire/component"
	"gofire/config"
	"gofire/pipeline"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	factory := component.DefaultFactory()
	pipelineMgr := pipeline.NewPipelineManager()
	configMgr, err := config.NewConfigManager("./etc", pipelineMgr, factory)
	if err != nil {
		log.Fatal(err)
	}

	// 启动配置管理器
	if err = configMgr.Start(); err != nil {
		log.Fatal(err)
	}
	defer configMgr.Stop()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	log.Println("正在关闭...")
	if err = pipelineMgr.Shutdown(); err != nil {
		log.Printf("关闭时发生错误: %v", err)
	}
}
