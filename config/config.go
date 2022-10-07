package config

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
