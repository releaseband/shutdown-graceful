package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

		signalType := s.String()

		fmt.Println("shutdown signal type ", signalType)
		if err := shutoff.Shutdown(); err != nil {
			log(fmt.Errorf("signal_type: %s, shutdown failed: %w",
				signalType, err))
		}

		close(idleConnsClosed)
	}()

	return idleConnsClosed
}

type ReadinessChecker interface {
	Shutdown()
}

type Server interface {
	Shutdown(ctx context.Context) error
}

type Configs struct {
	ReadinessChecker      ReadinessChecker
	Server                Server
	BeforeShutdownTimeout time.Duration
	ShutdownTimeout       time.Duration
	ErrorHandler          func(err error)
	InformationHandler    func(mess string)
	Modules               map[string]func(ctx context.Context) error
}

func ShutdownApp(cfg Configs) func(ctx context.Context) {
	return func(ctx context.Context) {
		if cfg.ReadinessChecker != nil {
			cfg.ReadinessChecker.Shutdown()
		}

		if cfg.InformationHandler != nil {
			cfg.InformationHandler("waiting before start shutdown graceful: " + cfg.BeforeShutdownTimeout.String())
		}

		time.Sleep(cfg.BeforeShutdownTimeout)

		count := len(cfg.Modules)
		if count > 0 {
			wg := &sync.WaitGroup{}
			wg.Add(len(cfg.Modules))

			for name, module := range cfg.Modules {
				go func(wg *sync.WaitGroup, name string, shutdown func(ctx context.Context) error) {
					if err := shutdown(ctx); err != nil {
						if cfg.ErrorHandler != nil {
							cfg.ErrorHandler(fmt.Errorf("module:%s, failed graceful shutdown: %w", name, err))
						}
					}

					wg.Done()
				}(wg, name, module)
			}

			wg.Wait()

			if cfg.Server != nil {
				if err := cfg.Server.Shutdown(ctx); err != nil {
					cfg.ErrorHandler(fmt.Errorf("server: failed graceful shutdown: %w", err))
				}
			}
		}
	}
}
