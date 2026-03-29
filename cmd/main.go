package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	cli "github.com/urfave/cli/v2"
)

var (
	version  string
	compiled string = fmt.Sprint(time.Now().Unix())
	githash  string

	logLevelFlag = &cli.StringFlag{
		Name:    "log.level",
		Aliases: []string{"ll"},
		Usage:   "Set the logging level (debug, info, warn, error)",
		Value:   "info",
	}

	logFormatFlag = &cli.StringFlag{
		Name:    "log.format",
		Aliases: []string{"lf"},
		Usage:   "Set the logging format (text, json)",
		Value:   "text",
	}
)

// setupLogger is a helper function to consistently setup the logger for the
// application. It will return the logger and a function intended to be used to
// log the duration of the application. In addition, this function can terminate
// the application early if incorrect values are supplied for the log level, log
// format, or log output. This is intentional as it should only be run at the start
// of the commands.
func setupLogger(ctx *cli.Context) (*slog.Logger, func(time.Time)) {
	// setup sane defaults for the logger options
	opts := slog.HandlerOptions{AddSource: false}
	if lvl := ctx.String(logLevelFlag.Name); len(lvl) > 0 {
		switch strings.ToLower(lvl) {
		case "trace":
			opts.Level = slog.LevelDebug // use debug level for trace
			opts.AddSource = true        // add source to trace level
		case "debug":
			opts.Level = slog.LevelDebug
		case "info":
			opts.Level = slog.LevelInfo
		case "warn":
			opts.Level = slog.LevelWarn
		case "error":
			opts.Level = slog.LevelError
		default:
			slog.Error("unsupported log level", "log.level.requested", lvl)
			os.Exit(1)
		}
	}

	// setup the writer for the logger, defaulting to stdout
	w := os.Stdout

	// setup the format handler for the logger, defaulting to JSON
	var h slog.Handler = slog.NewJSONHandler(w, &opts)
	if lf := ctx.String(logFormatFlag.Name); len(lf) > 0 {
		switch strings.ToLower(lf) {
		case "json":
			h = slog.NewJSONHandler(w, &opts)
		case "text":
			h = slog.NewTextHandler(w, &opts)
		default:
			slog.Error("unsupported log format", "log.format.requested", lf)
			os.Exit(1)
		}
	}

	// create the new logger prior to return so that it can be used in the "done" function
	logger := slog.New(h)

	// return the logger and the "done" function to be used to log the duration
	// of the application runtime.
	return logger, func(s time.Time) { logger.Debug("done", "duration", fmt.Sprintf("%dms", time.Since(s).Milliseconds())) }
}

func main() {
	ct, err := strconv.ParseInt(compiled, 0, 0)
	if err != nil {
		panic(err)
	}

	crdate := "2026"
	if now := time.Now().Format("2006"); now != crdate {
		crdate = fmt.Sprintf("%s-%s", crdate, now)
	}

	app := cli.App{
		Name:        "akctl",
		Description: "Agentic Kernel CLI tool for managing and updating the agentic kernel",
		Copyright:   fmt.Sprintf("© %s The Karoshi Workshop", crdate),
		Version:     fmt.Sprintf("%s | compiled %s | commit %s", version, time.Unix(ct, -1).Format(time.RFC3339), githash),
		Compiled:    time.Unix(ct, -1),
		Commands: []*cli.Command{
			{
				Name:        "init",
				Aliases:     []string{"setup"},
				Description: "Initialize the project/directory with the agentic kernel",
				Flags: []cli.Flag{
					logLevelFlag,
					logFormatFlag,
				},
				Action: func(c *cli.Context) error {
					logger, done := setupLogger(c)
					defer done(time.Now())

					logger.Info("stubbed command executed", "command", "init")

					return cli.Exit("not implemented yet", 1)
				},
			},
		},
	}

	app.Run(os.Args)
}
