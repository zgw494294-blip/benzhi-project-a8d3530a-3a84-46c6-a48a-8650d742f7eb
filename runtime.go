package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/transport"
)

func buildHandler(dataDir string, logger *slog.Logger) (http.Handler, error) {
	store, err := repository.Open(dataDir)
	if err != nil {
		return nil, err
	}
	service := application.NewService(store, rules.NewEngine())
	return transport.NewServer(service, logger).Handler(), nil
}

func runServer(address, dataDir string, logger *slog.Logger) error {
	handler, err := buildHandler(dataDir, logger)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := transport.NewHTTPServer(address, handler)
	logger.Info("海岸带修复证据验收台已启动", "address", "http://"+listener.Addr().String(), "data", dataDir)

	stopped := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		stopped <- err
	}()

	signalContext, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	select {
	case err := <-stopped:
		return err
	case <-signalContext.Done():
		shutdownContext, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := server.Shutdown(shutdownContext); err != nil {
			return err
		}
		return <-stopped
	}
}
