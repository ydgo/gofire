package config

import (
	"log"
	"os"
	"path"
)

// Provider 实现多个配置源的目的是你要这样部署，他要那样部署
// 每个人的项目在 main 文件中使用不同的方法去获取配置，而不是给一个参数去选择

type Provider interface {
	// Get todo 这里获取不到配置可能是因为错误，也可能是因为真的没有配置数据，需要添加容忍度（忽略、最大次数、立即）
	// Get all config
	Get() []Pipeline
}

type FileProvider struct {
	path  string
	isDir bool
	name  string
}

func NewFileProvider(path string) (Provider, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &FileProvider{
		path:  path,
		isDir: info.IsDir(),
		name:  info.Name(),
	}, nil
}

func (p *FileProvider) Get() []Pipeline {
	if !p.isDir {
		value, err := os.ReadFile(p.path)
		if err != nil {
			log.Printf("read %s: %s", p.path, err)
			return nil
		}
		return []Pipeline{{Name: p.name, Value: value}}
	}
	files, err := os.ReadDir(p.path)
	if err != nil {
		log.Printf("reading directory %s: %s", p.path, err)
		return nil
	}
	data := make([]Pipeline, 0)
	for _, file := range files {
		configPath := path.Join(p.path, file.Name())
		value, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("read %s: %s", p.path, err)
			return nil
		}
		data = append(data, Pipeline{
			Name:  file.Name(),
			Value: value,
		})
	}
	return data

}
