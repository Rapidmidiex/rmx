package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
)

type Config struct {
	ServerPort    string `json:"serverPort"`
	DBHost        string `json:"dbHost"`
	DBPort        string `json:"dbPort"`
	DBName        string `json:"dbName"`
	DBUser        string `json:"dbUser"`
	DBPassword    string `json:"dbPassword"`
	RedisHost     string `json:"redisHost"`
	RedisPort     string `json:"redisPort"`
	RedisPassword string `json:"redisPassword"`
}

const (
	configFileName    = "rmx.config.json"
	devConfigFileName = "rmx-dev.config.json"
)

// writes the values of the config to a file
// NOTE: this will overwrite the previous generated file
func (c *Config) WriteToFile(dev bool) error {
	var fp string
	if dev {
		fp = devConfigFileName
	} else {
		fp = configFileName
	}

	bs, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0660)
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
func ScanConfigFile(dev bool) (*Config, error) {
	// check for a config file
	var fp string
	if dev {
		fp = devConfigFileName
	} else {
		fp = configFileName
	}

	if _, err := os.Stat(fp); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	c := &Config{}
	bs, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bs, c); err != nil {
		return nil, err
	}

	return c, nil
}

// LoadConfigFromEnv creates a Config from environment variables.
func LoadConfigFromEnv(dev bool) (*Config, error) {
	serverPort := os.Getenv("PORT")

	pgURI := os.Getenv("POSTGRES_URI")

	pgParsed, err := url.Parse(pgURI)
	if err != nil && pgURI != "" {
		return nil, fmt.Errorf("invalid POSTGRES_URL env var: %q: %w", pgURI, err)
	}

	pgUser := pgParsed.User.Username()
	pgPassword, _ := pgParsed.User.Password()

	pgHost := pgParsed.Host
	pgPort := pgParsed.Port()
	pgName := pgParsed.Path

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	return &Config{
		ServerPort:    serverPort,
		DBHost:        pgHost,
		DBPort:        pgPort,
		DBName:        pgName,
		DBUser:        pgUser,
		DBPassword:    pgPassword,
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisPassword: redisPassword,
	}, nil
}
