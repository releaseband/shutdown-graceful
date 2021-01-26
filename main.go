package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func ListenShutdownSignals(
	teardown func(ctx context.Context),
	timeout time.Duration,
) <-chan struct{} {
	idleClose := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT)

		<-sigint
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		teardown(ctx)

		close(idleClose)
	}()

	return idleClose
}
