package main

import (
	"log/slog"

	"github.com/grzadr/godlv/internal/app"
)

func main() {
	app := app.NewApp(slog.LevelInfo)
	app.Info("hello world")
}
