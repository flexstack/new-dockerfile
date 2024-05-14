package runtime_test

import (
	"log/slog"
)

type noopWriter struct{}

func (w *noopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var logger = slog.New(slog.NewJSONHandler(&noopWriter{}, nil))
