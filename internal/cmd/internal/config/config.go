package config

import (
	"errors"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type GoogleConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

type AuthConfig struct {
	Enable    bool         `yaml:"enable"`
	Google    GoogleConfig `yaml:"google"`
	CookieKey string       `yaml:"cookie_key"`
}

type Config struct {
	Port string     `yaml:"port"`
	DB   DBConfig   `yaml:"db"`
	Auth AuthConfig `yaml:"auth"`
	Dev  bool       `yaml:"dev"`
}

const (
	configFileName = "rmx.config.yaml"
)

// writes the values of the config to a file
// NOTE: this will overwrite the previous generated file
func (c *Config) WriteToFile() error {
	bs, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(configFileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	if _, err := f.Write(bs); err != nil {
		return err
	}

	return nil
}

// checks for a config file and if one is available the value is returned
func ScanConfigFile() (*Config, error) {
	// check for a config file
	if _, err := os.Stat(configFileName); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	c := &Config{}
	bs, err := os.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(bs, c); err != nil {
		return nil, err
	}

	return c, nil
}
