package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen    string              `yaml:"listen"`
	Upstreams map[string]Upstream `yaml:"upstreams"`
	Routes    []Route             `yaml:"routes"`
}

type Upstream struct {
	URL string `yaml:"url"`
}

type Route struct {
	Path     string `yaml:"path"`
	Upstream string `yaml:"upstream"`
}

func LoadConfig(path string) (*Config, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
