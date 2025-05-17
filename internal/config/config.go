package config

import (
   "fmt"
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
   // Read config from file if present; file path set by caller via v.SetConfigFile()
   if err := v.ReadInConfig(); err != nil {
       // ignore missing config file; fail on other errors
       if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
           return nil, err
       }
   }

	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &config, nil
}
