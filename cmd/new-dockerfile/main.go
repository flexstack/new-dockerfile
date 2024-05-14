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
	flag.BoolVar(&noColor, "no-color", false, "Disable colorized output")
	var runtimeArg string
	flag.StringVar(&runtimeArg, "runtime", "", "Force a specific runtime")
	flag.Parse()

	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	handler := tint.NewHandler(os.Stdout, &tint.Options{
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
			if strings.ToLower(string(rt.Name())) == strings.ToLower(runtimeArg) {
				r = rt
				break
			}
		}
		if r == nil {
			runtimeNames := make([]string, len(runtimes))
			for i, rt := range runtimes {
				runtimeNames[i] = string(rt.Name())
			}
			log.Error(fmt.Sprintf(`Runtime "%s" not found. Expected one of: %s`, runtimeArg, "\n  - "+strings.Join(runtimeNames, "\n  - ")))
			os.Exit(1)
		}
	}

	if r == nil {
		r, err = df.MatchRuntime(path)
		if err != nil {
			log.Error("fatal error", "error", err.Error())
			os.Exit(1)
		}
	}

	contents, err := r.GenerateDockerfile(path)
	if err != nil {
		os.Exit(1)
	}

	if err = os.WriteFile(filepath.Join(path, "Dockerfile"), contents, 0644); err != nil {
		log.Error("fatal error", "error", err.Error())
		os.Exit(1)
	}

	// a.log.Info("Auto-generated Dockerfile for project using " + string(lang.Name()) + "\n" + *contents)
	log.Info("Auto-generated Dockerfile for project using " + string(r.Name()))
	return
}
