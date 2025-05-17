package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config holds the configuration for the application
type Config struct {
	Debug     bool   `mapstructure:"debug"`
	Context   string `mapstructure:"context"`
	Overrides string `mapstructure:"overrides"`
	Output    string `mapstructure:"output"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(v *viper.Viper) (*Config, error) {
	v.AddConfigPath(".schemagen")
	v.SetConfigName("config")
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
		return nil, err
	}

	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &config, nil
}
