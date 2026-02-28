package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDriver string
	DBSource string
}

func LoadDBConfig() *Config {
	godotenv.Load(".env")
	source := os.Getenv("DBCONN")
	return &Config{
		DBDriver: "postgres",
		DBSource: source,
	}
}
