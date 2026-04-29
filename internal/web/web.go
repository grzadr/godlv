package web

import (
	"context"
	"embed"
	"fmt"
	"html"
	"html/template"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/grzadr/godlv/internal/config"
	"github.com/grzadr/godlv/internal/runcmd"
	"github.com/grzadr/godlv/internal/setup"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
	jobsTimeout       = 15 * time.Second
)

type ServerApp struct {
	*setup.App

	cfg     *config.ArgConfig
	getTmpl func() *template.Template
	wg      *sync.WaitGroup
}

func NewServerApp(
	app *setup.App,
	cfg *config.ArgConfig,
) *ServerApp {
	return &ServerApp{
		App: app,
		cfg: cfg,
		getTmpl: sync.OnceValue(func() *template.Template {
			return template.Must(
				template.ParseFS(templateFS, "templates/*.html"),
			)
		}),
		wg: new(sync.WaitGroup),
	}
}

func cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}

func (app *ServerApp) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	app.Info("connection", "remote", r.RemoteAddr)

	tmpl := app.getTmpl()

	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		// Using the injected structured logger
		app.Error("template execution failed", "error", err)
	}
}

// We change the signature to return an [http.HandlerFunc] so we can inject the
// context.
func (app *ServerApp) handleDownload(appCtx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const maxBodySize = 1 << 20
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		videoURL := r.FormValue("url")
		if videoURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		app.wg.Add(1)

		// Launch the goroutine, passing the explicitly injected appCtx
		app.wg.Go(func() {
			app.Info("Starting background download", "url", videoURL)

			cfgCopy := *app.cfg
			cfgCopy.NonFlag = []string{videoURL}

			// runCmd now uses the appCtx passed through the closure
			err := runcmd.RunCmd(appCtx, app.App, &cfgCopy)
			if err != nil {
				app.Error(
					"Background download failed",
					"url",
					videoURL,
					"error",
					err,
				)
				return
			}

			app.Info(
				"Background download finished successfully",
				"url",
				videoURL,
			)
		})

		w.WriteHeader(http.StatusAccepted)
		safeURL := html.EscapeString(videoURL)
		response := fmt.Sprintf(
			"<p>Download triggered successfully for: <strong>%s</strong></p>",
			safeURL,
		)

		if _, err := w.Write([]byte(response)); err != nil {
			app.Error("failed to write response to client", "error", err)
		}
	}
}

func RunServer(
	ctx context.Context,
	app *setup.App,
	cfg *config.ArgConfig,
) error {
	serverApp := NewServerApp(app, cfg)
	mux := http.NewServeMux()

	mux.Handle("GET /static/", cacheMiddleware(http.FileServerFS(staticFS)))

	mux.HandleFunc("GET /{$}", serverApp.handleIndex)
	mux.HandleFunc("POST /download", serverApp.handleDownload(ctx))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	// Create a channel to catch any errors from ListenAndServe
	serverError := make(chan error, 1)

	// 1. Start the server in a goroutine
	go func() {
		serverApp.Info("Server listening", "address", server.Addr)
		// ListenAndServe always returns a non-nil error.
		serverError <- server.ListenAndServe()
	}()

	// 2. Block until either the context cancels (Ctrl+C) or the server crashes
	select {
	case err := <-serverError:
		// If the server crashed on startup (e.g., port in use)
		return fmt.Errorf("server error: %w", err)

	case <-ctx.Done():
		// Ctrl+C was pressed. We caught the signal.
		serverApp.Info(
			"Shutdown signal received, initiating graceful shutdown...",
		)

		// Create a separate timeout context for the shutdown process itself.
		// This prevents the server from hanging forever if a client holds a
		// connection open.
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer shutdownCancel()

		// 3. Explicitly shut down the server
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		serverApp.Info("Waiting for background tasks to finish...")

		// Wrap wg.Wait() in a channel so we don't hang forever if yt-dlp gets
		// stuck
		waitCh := make(chan struct{})
		go func() {
			serverApp.wg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			serverApp.Info("All background tasks finished cleanly")
		case <-time.After(jobsTimeout):
			serverApp.Error("Timed out waiting for background tasks to abort")
		}
	}

	return nil
}
