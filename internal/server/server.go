package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/latoulicious/Tsugi/internal/config"
	"github.com/latoulicious/Tsugi/internal/version"
)

const (
	versionPath = "/version"
	healthzPath = "/healthz"

	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func New(cfg *config.Config, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+versionPath, handleVersion(logger))
	mux.HandleFunc("GET "+healthzPath, handleHealthz)
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func handleVersion(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// marshal first so a failure can't leave a half-written 200
		body, err := json.Marshal(version.Get())
		if err != nil {
			logger.ErrorContext(r.Context(), "marshal version", "error", err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(body)
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = io.WriteString(w, `{"status":"ok","service":"tsugi"}`+"\n")
}
