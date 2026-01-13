package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(logger *log.Logger) *Config {
	cfg := &Config{}

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.Fatalf("[ERR]: config.yaml not found: %v", err)
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		logger.Fatalf("[ERR]: failed to parse config.yaml: %v", err)
	}

	if cfg.BotToken == "" {
		logger.Fatal("[ERR]: bot_token is empty in config.yaml")
	}
	if cfg.TogglToken == "" {
		logger.Fatal("[ERR]: toggl_token is empty in config.yaml")
	}
	if cfg.TogglWorkspaceID == 0 {
		logger.Fatal("[ERR]: toggl_workspace is empty or 0 in config.yaml")
	}

	logger.Printf("Config loaded successfully. Mappings found: %d", len(cfg.Stats.Mappings))
	return cfg
}

func LoadFromPath(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
