package shutdown

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type StartShutdownProcess interface {
	Shutdown() error
}

func ListenShutdownSignals(shutdown StartShutdownProcess, errorHandler func(err error)) <-chan struct{} {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		s := <-sigint

		signalType := s.String()

		fmt.Println("shutdown signal type ", signalType)
		if err := shutdown.Shutdown(); err != nil {
			errorHandler(fmt.Errorf("signal_type: %s, shutdown failed: %w",
				signalType, err))
		}

		close(idleConnsClosed)
	}()

	return idleConnsClosed
}
