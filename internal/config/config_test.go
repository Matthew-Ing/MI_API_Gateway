package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("../../configs/gateway.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if config.Listen != ":8088" {
		t.Fatalf("Expected listen to be 8088, got %s", config.Listen)
	}
	if len(config.Upstreams) != 2 {
		t.Fatalf("Expected 2 upstreams, got %d", len(config.Upstreams))
	}
	if len(config.Routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d", len(config.Routes))
	}
}
