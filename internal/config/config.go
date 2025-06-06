package config

import (
   "fmt"
   "os"
   "gopkg.in/yaml.v2"
)

// Config holds the configuration for the application
// Config holds the configuration for the application
type Config struct {
   Debug     bool   `yaml:"debug"`
   Context   string `yaml:"context"`
   Overrides string `yaml:"overrides"`
   Output    string `yaml:"output"`
}

// LoadConfig reads configuration from a YAML file (if it exists).
// If the file is not found, returns an empty Config without error.
func LoadConfig(path string) (*Config, error) {
   data, err := os.ReadFile(path)
   if err != nil {
       if os.IsNotExist(err) {
           return &Config{}, nil
       }
       return nil, err
   }
   var cfg Config
   if err := yaml.Unmarshal(data, &cfg); err != nil {
       return nil, fmt.Errorf("failed to parse config: %w", err)
   }
   return &cfg, nil
}
