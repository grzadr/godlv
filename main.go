package main

import (
	"log/slog"
	"os"
)

type App struct {
	logLevel *slog.LevelVar
	logger   *slog.Logger
}

func NewApp(level slog.Level) *App {
	lvl := new(slog.LevelVar)
	lvl.Set(level)

	return new(App{
		logLevel: lvl,
		logger: slog.New(
			slog.NewTextHandler(
				os.Stdout,
				new(slog.HandlerOptions{Level: lvl}),
			),
		),
	})
}

func (a *App) Debug(msg string, args ...any) {
	a.logger.Debug(msg, args...)
}

func (a *App) Info(msg string, args ...any) {
	a.logger.Info(msg, args...)
}

func main() {
	app := NewApp(slog.LevelInfo)
	app.Info("hello world")
}
