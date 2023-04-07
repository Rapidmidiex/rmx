package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

type Config struct {
	ServerPort         string `json:"serverPort"`
	DBURL              string `json:"dbUrl"`
	DBHost             string `json:"dbHost"`
	DBPort             string `json:"dbPort"`
	DBName             string `json:"dbName"`
	DBUser             string `json:"dbUser"`
	DBPassword         string `json:"dbPassword"`
	RedisHost          string `json:"redisHost"`
	RedisPort          string `json:"redisPort"`
	RedisPassword      string `json:"redisPassword"`
	GoogleClientID     string `json:"googleClientId"`
	GoogleClientSecret string `json:"googleClientSecret"`
	Dev                bool   `json:"dev"`
}

const (
	configFileName = "rmx.config.json"
)

// writes the values of the config to a file
// NOTE: this will overwrite the previous generated file
func (c *Config) WriteToFile() error {
	bs, err := json.MarshalIndent(c, "", "    ")
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
