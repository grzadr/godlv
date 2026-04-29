package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"time"

	"github.com/grzadr/godlv/internal/config"
	"github.com/grzadr/godlv/internal/setup"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

const readHeaderTimeout = 5 * time.Second

type ServerApp struct {
	setup.App
}

func NewServerApp(app *setup.App) *ServerApp {
	return &ServerApp{
		App: *app,
	}
}

func (app *ServerApp) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		// Using the injected structured logger
		app.Error("template execution failed", "error", err)
	}
}

func (app *ServerApp) handleDownload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)

	_, err := w.Write([]byte("<p>Download triggered successfully.</p>"))
	if err != nil {
		// Structured logging allows easy key-value querying later
		app.Error("failed to write response to client", "error", err)
	}
}

func RunServer(
	ctx context.Context,
	app *setup.App,
	cfg *config.ArgConfig,
) error {
	serverApp := NewServerApp(app)
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	mux.HandleFunc("GET /", serverApp.handleIndex)
	mux.HandleFunc("POST /download", serverApp.handleDownload)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	serverApp.Info("Server listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}
	return nil
}
