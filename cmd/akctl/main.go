package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	cli "github.com/urfave/cli/v2"

	ghpkg "github.com/elliottpolk/akctl/internal/github"
	"github.com/elliottpolk/akctl/internal/kernel"
	"github.com/elliottpolk/akctl/internal/setup"
	syncp "github.com/elliottpolk/akctl/internal/sync"
	"github.com/elliottpolk/akctl/internal/ui"
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

	forceFlag = &cli.BoolFlag{
		Name:    "force",
		Aliases: []string{"f"},
		Usage:   "Skip destructive overwrite confirmation prompts",
	}

	kernelSourceFlag = &cli.StringFlag{
		Name:    "kernel.source",
		Aliases: []string{"kernel.src"},
		Usage:   "Kernel source as github.com/<owner>/<repo> (default: github.com/elliottpolk/agentic-kernel)",
	}

	githubTokenFlag = &cli.StringFlag{
		Name:    "github.token",
		Usage:   "GitHub personal access token for private repos and higher rate limits",
		EnvVars: []string{"GITHUB_TOKEN"},
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
					forceFlag,
					kernelSourceFlag,
					githubTokenFlag,
				},
				Action: func(c *cli.Context) error {
					_, done := setupLogger(c)
					defer done(time.Now())

					ctx := context.Background()

					owner, repo, err := ghpkg.ParseSource(c.String(kernelSourceFlag.Name))
					if err != nil {
						return cli.Exit(err.Error(), 1)
					}

					token := strings.TrimSpace(c.String(githubTokenFlag.Name))

					client := ghpkg.NewClient(ctx, token)

					if err := ghpkg.CheckRateLimit(ctx, client); err != nil {
						return cli.Exit(err.Error(), 1)
					}

					var reporter kernel.ProgressReporter
					if !c.Bool(forceFlag.Name) {
						reporter = ui.NewTeaProgressReporter()
					}

					k, err := kernel.Fetch(ctx, client, owner, repo, reporter)
					if err != nil {
						if ghpkg.IsNotFound(err) && token == "" {
							return cli.Exit("repo not found or may be private; use --github.token if the repo is private", 1)
						}
						if ghpkg.IsNotFound(err) {
							return cli.Exit(fmt.Sprintf("repo %s/%s not found", owner, repo), 1)
						}
						return cli.Exit(fmt.Sprintf("fetch kernel: %v", err), 1)
					}
					defer os.RemoveAll(k.CacheDir)

					if err := setup.Run(k, setup.Options{
						Force:     c.Bool(forceFlag.Name),
						TargetDir: ".",
					}); err != nil {
						return cli.Exit(err.Error(), 1)
					}

					fmt.Println(ui.SuccessStyle.Render("Agentic kernel initialized successfully."))
					return nil
				},
			},
			{
				Name:        "sync",
				Aliases:     []string{"update"},
				Description: "Sync the project/directory agentic kernel with the latest version and updates",
				Flags: []cli.Flag{
					logLevelFlag,
					logFormatFlag,
					forceFlag,
					githubTokenFlag,
				},
				Action: func(c *cli.Context) error {
					_, done := setupLogger(c)
					defer done(time.Now())

					ctx := context.Background()

					token := strings.TrimSpace(c.String(githubTokenFlag.Name))
					client := ghpkg.NewClient(ctx, token)

					if err := ghpkg.CheckRateLimit(ctx, client); err != nil {
						return cli.Exit(err.Error(), 1)
					}

					targetDir := "."

					if err := syncp.Run(ctx, client, syncp.Options{
						Force:     c.Bool(forceFlag.Name),
						TargetDir: targetDir,
					}); err != nil {
						return cli.Exit(err.Error(), 1)
					}

					return nil
				},
			},
		},
	}

	app.Run(os.Args)
}
