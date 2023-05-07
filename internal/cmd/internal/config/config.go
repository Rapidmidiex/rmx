package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type GoogleConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type AuthConfig struct {
	Enable              bool         `json:"enable"`
	Google              GoogleConfig `json:"google"`
	CookieHashKey       string       `json:"cookieHashKey"`
	CookieEncryptionKey string       `json:"cookieEncryptionKey"`
}

type Config struct {
	Port string     `json:"port"`
	DB   DBConfig   `json:"db"`
	Auth AuthConfig `json:"auth"`
	Dev  bool       `json:"dev"`
}

const (
	configFileName = "rmx.config.json"
)

// writes the values of the config to a file
// NOTE: this will overwrite the previous generated file
func (c *Config) WriteToFile() error {
	bs, err := json.MarshalIndent(c, "", "\t")
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

	if err := json.Unmarshal(bs, c); err != nil {
		return nil, err
	}

	return c, nil
}
