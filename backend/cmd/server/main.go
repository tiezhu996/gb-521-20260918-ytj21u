package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mine-ventilation-network-simulator/backend/internal/config"
	"mine-ventilation-network-simulator/backend/internal/router"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server_stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger := config.ConfigureLogger(cfg.LogLevel)
	db, err := config.OpenDatabase(cfg)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	engine := router.New(cfg, db, logger)
	server := &http.Server{
		Addr: ":" + cfg.Port, Handler: engine,
		ReadHeaderTimeout: cfg.ShutdownTimeout,
		IdleTimeout:       cfg.ShutdownTimeout * 8,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server_started", "port", cfg.Port, "db_driver", cfg.DBDriver)
		serverErrors <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signal := <-signals:
		logger.Info("shutdown_requested", "signal", signal.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}
