package shutdown

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type Shutoff interface {
	Shutdown() error
}

type errorLogging = func(err error)

func ListenShutdownSignals(shutoff Shutoff, log errorLogging) <-chan struct{} {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		s := <-sigint

		if err := shutoff.Shutdown(); err != nil {
			log(fmt.Errorf("signal_type: %s, shutdown failed: %w",
				s.String(), err))
		}

		close(idleConnsClosed)
	}()

	return idleConnsClosed
}