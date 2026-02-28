package config

import "os"

type Config struct {
	DBDriver string
	DBSource string
}

func LoadDBConfig() *Config {
	source := os.Getenv("DBCONN")
	return &Config{
		DBDriver: "postgres",
		DBSource: source,
	}
}
