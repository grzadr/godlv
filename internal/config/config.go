package config

import (
	"flag"
	"log/slog"
)

type ArgConfig struct {
	LogLevel  slog.Level
	OutputDir string
	NonFlag   []string
}

func NewArgConfig(args []string) (*ArgConfig, error) {
	fs := flag.NewFlagSet("Default", flag.ContinueOnError)
	conf := &ArgConfig{}

	const (
		outputDirDefault = "./"
		outputDirUsage   = "location of downloaded files"
	)

	debugMode := fs.Bool(
		"debug",
		false,
		"enable debug mode (logs, asserts, ...)",
	)

	fs.TextVar(
		&conf.LogLevel,
		"log-level",
		slog.LevelInfo,
		"log level (debug, info, warn, error)",
	)

	fs.StringVar(&conf.OutputDir, "output", outputDirDefault, outputDirUsage)
	fs.StringVar(&conf.OutputDir, "o", outputDirDefault, outputDirUsage)

	if parseErr := fs.Parse(args); parseErr != nil {
		return conf, parseErr
	}

	if *debugMode {
		conf.LogLevel = slog.LevelDebug
	}

	conf.NonFlag = fs.Args()

	return conf, nil
}
