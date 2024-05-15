package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerfile "github.com/flexstack/new-dockerfile"
	"github.com/flexstack/new-dockerfile/runtime"
	"github.com/lmittmann/tint"
	flag "github.com/spf13/pflag"
)

func main() {
	var path string
	flag.StringVar(&path, "path", ".", "Path to the project directory")
	var noColor bool
	flag.BoolVar(&noColor, "no-color", os.Getenv("NO_COLOR") == "true" || os.Getenv("ANSI_COLORS_DISABLED") == "true", "Disable colorized output")
	var runtimeArg string
	flag.StringVar(&runtimeArg, "runtime", "", "Force a specific runtime")
	var quiet bool
	flag.BoolVar(&quiet, "quiet", false, "Disable all log output except errors")
	var write bool
	flag.BoolVar(&write, "write", false, "Write the Dockerfile to disk at ./Dockerfile")
	flag.Parse()

	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	} else if quiet {
		level = slog.LevelError
	}

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
		NoColor:    noColor,
	})

	log := slog.New(handler)
	df := dockerfile.New(log)

	var (
		r   runtime.Runtime
		err error
	)

	if runtimeArg != "" {
		runtimes := df.ListRuntimes()

		for _, rt := range runtimes {
			if strings.EqualFold(string(rt.Name()), runtimeArg) {
				r = rt
				break
			}
		}
		if r == nil {
			runtimeNames := make([]string, len(runtimes))
			for i, rt := range runtimes {
				runtimeNames[i] = strings.ToLower(string(rt.Name()))
			}

			if runtimeArg == "list" {
				fmt.Println("Available runtimes:")
				fmt.Println("  - " + strings.Join(runtimeNames, "\n  - "))
				os.Exit(0)
			}

			log.Error(fmt.Sprintf(`Runtime "%s" not found. Expected one of: %s`, runtimeArg, "\n  - "+strings.Join(runtimeNames, "\n  - ")))
			os.Exit(1)
		}
	}

	if r == nil {
		r, err = df.MatchRuntime(path)
		if err != nil {
			log.Error("Fatal error: " + err.Error())
			os.Exit(1)
		}
	}

	contents, err := r.GenerateDockerfile(path)
	if err != nil {
		os.Exit(1)
	}

	if !write {
		fmt.Println(string(contents))
		return
	}

	output := filepath.Join(path, "Dockerfile")
	if err = os.WriteFile(output, contents, 0644); err != nil {
		log.Error("Fatal error: " + err.Error())
		os.Exit(1)
	}

	log.Info(fmt.Sprintf("Auto-generated Dockerfile for project using %s: %s", string(r.Name()), output))
}
