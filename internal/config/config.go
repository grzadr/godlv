package config

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

//go:generate stringer -type=FormatAlias -linecomment
type FormatAlias int

const (
	Mp3 FormatAlias = iota + 1 // mp3
	Aac                        // aac
	Mp4                        // mp4
	Mkv                        // mkv
)

func (f *FormatAlias) UnmarshalText(text []byte) error {
	switch strings.ToLower((string(text))) {
	case "mp3":
		*f = Mp3
	case "aac":
		*f = Aac
	case "mp4":
		*f = Mp4
	case "mkv":
		*f = Mkv
	default:
		return fmt.Errorf("invalid format alias: %q", text)
	}
	return nil
}

func (f FormatAlias) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

type ArgConfig struct {
	LogLevel   slog.Level
	OutputDir  string
	TempDir    string
	NonFlag    []string
	Overwrite  bool
	NoContinue bool
	Format     FormatAlias
	Port       int
}

func NewArgConfig(args []string) (*ArgConfig, error) {
	fs := flag.NewFlagSet("Default", flag.ContinueOnError)
	conf := &ArgConfig{}

	const (
		outputDirDefault  = "./"
		outputDirUsage    = "location of downloaded files, default: ./"
		outputTempDefault = "./temp"
		outputTempUsage   = "location of temporary files, default: ./temp"
		noContinueUsage   = "Do not resume partially downloaded fragments. If the file is not fragmented, restart download of the entire file"
		overwriteUsage    = "force files to be "
		formatUsage       = "[mp3|aac|mp4|mkv] default: mkv"
		portDefault       = 8080
		portUsage         = "port use for web server, default: 8080"
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
	fs.StringVar(&conf.TempDir, "temp", outputTempDefault, outputTempUsage)
	fs.StringVar(&conf.TempDir, "t", outputTempDefault, outputTempUsage)
	fs.BoolVar(&conf.NoContinue, "no-continue", false, noContinueUsage)
	fs.BoolVar(&conf.Overwrite, "overwrite", false, overwriteUsage)
	fs.TextVar(&conf.Format, "format", Mkv, formatUsage)
	fs.TextVar(&conf.Format, "f", Mkv, formatUsage)
	fs.IntVar(&conf.Port, "port", portDefault, portUsage)
	fs.IntVar(&conf.Port, "p", portDefault, portUsage)

	if parseErr := fs.Parse(args); parseErr != nil {
		return conf, parseErr
	}

	if *debugMode {
		conf.LogLevel = slog.LevelDebug
	}

	conf.NonFlag = fs.Args()

	return conf, nil
}

type ArgFlags []string

func NewArgFlags(cfg *ArgConfig) (ArgFlags, error) {
	args := []string{
		"--no-progress",
		"--paths", cfg.OutputDir,
		"--paths", "temp:" + cfg.TempDir,
		"-t", cfg.Format.String(),
	}

	if cfg.Overwrite {
		args = append(args, "--force-overwrites")
	} else if cfg.NoContinue {
		args = append(args, "--no-continue")
	}

	return args, nil
}
