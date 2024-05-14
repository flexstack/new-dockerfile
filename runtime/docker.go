package runtime

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Docker struct {
	Log *slog.Logger
}

func (d *Docker) Name() RuntimeName {
	return RuntimeNameDocker
}

func (d *Docker) Match(path string) bool {
	if stat, err := os.Stat(filepath.Join(path, "Dockerfile")); err == nil && !stat.IsDir() {
		d.Log.Info("Detected Docker project")
		return true
	}

	d.Log.Debug("Docker project not detected")
	return false
}

func (d *Docker) GenerateDockerfile(path string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(path, "Dockerfile"))
	if err != nil {
		return nil, fmt.Errorf("Failed to read Dockerfile")
	}

	return b, nil
}
