// Command api is the service's HTTP entry point: load config, boot,
// serve, and shut down gracefully on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Laaaaksh/gohighlevel-round1/internal/boot"
	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
)

const (
	shutdownTimeout = 10 * time.Second

	// http.Server.Addr is "host:port"; an empty host means every interface.
	addrPortSeparator = ":"
)

const (
	msgConfigLoadFailed      = "failed to load config"
	msgBootFailed            = "failed to boot application"
	msgServerStarting        = "server starting"
	msgServerFailed          = "server failed"
	msgShuttingDown          = "shutting down"
	msgServerShutdownFailed  = "server shutdown failed"
	msgMongoDisconnectFailed = "mongo disconnect failed"
	msgShutdownComplete      = "shutdown complete"

	logFieldPort  = "port"
	logFieldError = "error"
)

// main does nothing but choose the exit code: os.Exit skips deferred calls,
// so everything holding a resource lives in run and returns an error instead.
func main() {
	if err := run(logger.New()); err != nil {
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		log.Error(msgConfigLoadFailed, logFieldError, err)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := boot.Boot(ctx, cfg)
	if err != nil {
		log.Error(msgBootFailed, logFieldError, err)
		return err
	}

	server := &http.Server{
		Addr:    addrPortSeparator + cfg.Server.Port,
		Handler: app.Router,
	}

	serverErr := make(chan error, 1)
	go func() { serverErr <- listenAndServe(server, log, cfg.Server.Port) }()

	err = awaitStop(ctx, serverErr, log)
	// Restore default signal handling so a second Ctrl-C kills a hung shutdown.
	stop()

	shutdown(server, app, log)
	return err
}

// awaitStop blocks until a signal arrives or the server stops on its own, and
// reports what ended it. Either way the caller still runs shutdown, so the
// Mongo client is always disconnected cleanly.
func awaitStop(ctx context.Context, serverErr <-chan error, log *slog.Logger) error {
	select {
	case err := <-serverErr:
		if err != nil {
			log.Error(msgServerFailed, logFieldError, err)
		}
		return err
	case <-ctx.Done():
		log.Info(msgShuttingDown)
		return nil
	}
}

func listenAndServe(server *http.Server, log *slog.Logger, port string) error {
	log.Info(msgServerStarting, logFieldPort, port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func shutdown(server *http.Server, app *boot.App, log *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error(msgServerShutdownFailed, logFieldError, err)
	}

	if err := database.Disconnect(app.MongoClient); err != nil {
		log.Error(msgMongoDisconnectFailed, logFieldError, err)
	}

	log.Info(msgShutdownComplete)
}
