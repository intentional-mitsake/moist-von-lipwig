package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDriver string
	DBSource string
}

type Delivery struct {
	PostID      string
	Delivery    time.Time
	IsDelivered bool
}

func LoadDBConfig() *Config {
	godotenv.Load(".env")
	source := os.Getenv("DBCONN")
	return &Config{
		DBDriver: "postgres",
		DBSource: source,
	}
}
