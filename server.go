package rpclens

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func ListenAndServeUntilSignal(server *http.Server) error {
	// from server.Shutdown documentation
	shutdownDone := make(chan error)
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)

		sig := <-sigint

		slog.Warn(fmt.Sprintf("Received %s, shutting down", sig.String()))
		// We received an interrupt signal, shut down.
		err := server.Shutdown(context.Background())
		shutdownDone <- err
	}()

	// TODO: Deal with a socket passed in on FD 3
	// TODO: Add TLS support if necessary
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Log(context.Background(), slog.LevelError+4, "FATAL: "+err.Error())
		return err
	}

	err = <-shutdownDone
	if err != nil {
		slog.Error("Error shutting down: " + err.Error())
	}
	return err
}
