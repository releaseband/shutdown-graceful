package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func ListenShutdownSignals(
	teardown func(ctx context.Context) error,
	errChan chan<- error,
	timeout time.Duration,
) <-chan struct{} {
	idleClose := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT)

		s := <-sigint

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := teardown(ctx); err != nil {
			errChan <- fmt.Errorf("signal_type: %s, shutdown failed: %w", s.String(), err)
		}

		close(idleClose)
	}()

	return idleClose
}
