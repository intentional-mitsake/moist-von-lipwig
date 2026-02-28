package pkg

import (
	"log/slog"
	"os"
)

func CreateLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
}
