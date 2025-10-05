package processing

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigLoader holds the loaded ingestion configurations
type ConfigLoader struct {
	configs map[string]IngestionConfig
}

// NewConfigLoader recursively scans a directory for YAML files, loads them, validates them
// and returns a ConfigLoader instance.
func NewConfigLoader(configPath string) (*ConfigLoader, error) {
	configs := make(map[string]IngestionConfig)

	err := filepath.WalkDir(configPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || (filepath.Ext(d.Name()) != ".yaml" && filepath.Ext(d.Name()) != ".yml") {
			return nil
		}

		slog.Info("Loading ingestion config", "file", path)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		var config IngestionConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse YAML for %s: %w", path, err)
		}

		if err := config.Validate(); err != nil {
			return fmt.Errorf("validation failed for %s: %w", path, err)
		}

		if _, exists := configs[config.ReportType]; exists {
			return fmt.Errorf("duplicate reportType '%s' found in %s", config.ReportType, path)
		}

		configs[config.ReportType] = config
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking config directory %s: %w", configPath, err)
	}

	if len(configs) == 0 {
		slog.Warn("No ingestion configs were loaded.", "path", configPath)
	}

	return &ConfigLoader{configs: configs}, nil
}

// GetConfig retrieves a validated configuration by its report type.
func (l *ConfigLoader) GetConfig(reportType string) (IngestionConfig, bool) {
	config, ok := l.configs[reportType]
	return config, ok
}
