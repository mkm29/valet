package config

import (
   "fmt"
   "os"
   "gopkg.in/yaml.v2"
   "github.com/mkm29/valet/internal/telemetry"
)

// Config holds the configuration for the application
type Config struct {
   Debug     bool                `yaml:"debug"`
   Context   string             `yaml:"context"`
   Overrides string             `yaml:"overrides"`
   Output    string             `yaml:"output"`
   Telemetry *telemetry.Config  `yaml:"telemetry"`
}

// LoadConfig reads configuration from a YAML file (if it exists).
// If the file is not found, returns an empty Config without error.
func LoadConfig(path string) (*Config, error) {
   cfg := &Config{
       Telemetry: telemetry.DefaultConfig(),
   }
   
   data, err := os.ReadFile(path)
   if err != nil {
       if os.IsNotExist(err) {
           return cfg, nil
       }
       return nil, err
   }
   
   if err := yaml.Unmarshal(data, cfg); err != nil {
       return nil, fmt.Errorf("failed to parse config: %w", err)
   }
   
   // Ensure telemetry config is not nil
   if cfg.Telemetry == nil {
       cfg.Telemetry = telemetry.DefaultConfig()
   }
   
   return cfg, nil
}
