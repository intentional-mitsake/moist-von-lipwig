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
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	source := os.Getenv("DBCONN")
	println(source)
	return &Config{
		DBDriver: "postgres",
		DBSource: source,
	}
}
