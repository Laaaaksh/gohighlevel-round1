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

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error(msgConfigLoadFailed, logFieldError, err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := boot.Boot(ctx, cfg)
	if err != nil {
		log.Error(msgBootFailed, logFieldError, err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    addrPortSeparator + cfg.Server.Port,
		Handler: app.Router,
	}

	go runServer(server, log, cfg.Server.Port)

	<-ctx.Done()
	stop()
	log.Info(msgShuttingDown)

	shutdown(server, app, log)
}

func runServer(server *http.Server, log *slog.Logger, port string) {
	log.Info(msgServerStarting, logFieldPort, port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error(msgServerFailed, logFieldError, err)
		os.Exit(1)
	}
}

func shutdown(server *http.Server, app *boot.App, log *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error(msgServerShutdownFailed, logFieldError, err)
	}

	if err := app.MongoClient.Disconnect(shutdownCtx); err != nil {
		log.Error(msgMongoDisconnectFailed, logFieldError, err)
	}

	log.Info(msgShutdownComplete)
}
