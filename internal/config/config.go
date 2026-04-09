package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadAll loads all config files and returns an AppConfig struct. It also validates that the required fields are present.
func LoadAll(configPath, topicsPath, sourcesPath string) (*AppConfig, error) {
	var cfg Config
	if err := loadYAML(configPath, &cfg); err != nil {
		return nil, err
	}

	var topics TopicsFile
	if err := loadYAML(topicsPath, &topics); err != nil {
		return nil, err
	}

	var sources SourcesFile
	if err := loadYAML(sourcesPath, &sources); err != nil {
		return nil, err
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("database_url is empty (use ${DATABASE_URL} or a literal)")
	}
	return &AppConfig{Config: cfg, Topics: topics, Sources: sources}, nil
}

// loadYAML is a helper function to read a YAML file and unmarshal it into the provided struct.
func loadYAML(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, out)
}
