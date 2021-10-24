package shutdown

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var ErrTimedOut = errors.New("timed out")

type StartShutdownProcess interface {
	Shutdown() error
}

type ReadinessChecker interface {
	Shutdown()
}

type Shutdown interface {
	Shutdown(ctx context.Context) error
}

type Logger interface {
	Info(mess string)
	Error(err error)
}

type gracefulShutdownApp struct {
	Timeouts         Timeouts
	ReadinessChecker ReadinessChecker
	Logger           Logger
	Modules          []Shutdown
	Server           Shutdown
}

func NewGracefulShutdownApp(
	timeouts Timeouts, checker ReadinessChecker, server Shutdown, modules []Shutdown, logger Logger) *gracefulShutdownApp {
	return &gracefulShutdownApp{
		Timeouts:         timeouts,
		ReadinessChecker: checker,
		Logger:           logger,
		Modules:          modules,
		Server:           server,
	}
}

func turnOffModules(ctx context.Context, done chan<- struct{}, logger Logger, modules ...Shutdown) {
	wg := &sync.WaitGroup{}

	for _, module := range modules {
		wg.Add(1)

		go func(module Shutdown) {
			if err := module.Shutdown(ctx); err != nil {
				logger.Error(err)
			}

			wg.Done()
		}(module)
	}

	wg.Wait()
	done <- struct{}{}
}

func wait(ctx context.Context, ch <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ErrTimedOut
	case <-ch:
	}

	return nil
}

func shutdown(timeout time.Duration, logger Logger, modules ...Shutdown) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan struct{})
	go turnOffModules(ctx, ch, logger, modules...)

	return wait(ctx, ch)
}

func (g *gracefulShutdownApp) waitBeforeShutdown() {
	t := g.Timeouts.GetBeforeTimeout()

	g.Logger.Info("waiting for the termination of running processes" + t.String())
	time.Sleep(t)
}

func (g *gracefulShutdownApp) shutdownModules() {
	t := g.Timeouts.GetModuleTimeout()

	g.Logger.Info("timeout for shutdown modules = " + t.String())

	if err := shutdown(t, g.Logger, g.Modules...); err != nil {
		g.Logger.Error(fmt.Errorf("modules shutdown failed: %w", err))
	}
}

func (g *gracefulShutdownApp) shutdownServer() error {
	t := g.Timeouts.GetServerTimeout()

	g.Logger.Info("timeout for shutdown server = " + t.String())

	err := shutdown(t, g.Logger, g.Server)
	if err != nil {
		g.Logger.Error(fmt.Errorf("server shutdown failed: %w", err))
	}

	return err
}

func (g *gracefulShutdownApp) Shutdown() error {
	if g.ReadinessChecker != nil {
		g.ReadinessChecker.Shutdown()
	}

	g.waitBeforeShutdown()
	g.shutdownModules()

	return g.shutdownServer()
}

func (g gracefulShutdownApp) ListenShutdownSignals(shutdown StartShutdownProcess) <-chan struct{} {
	idleConnsClosed := make(chan struct{}, 1)

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		s := <-sigint

		signalType := "signal type: " + s.String()

		g.Logger.Info(signalType)

		if err := shutdown.Shutdown(); err != nil {
			g.Logger.Error(fmt.Errorf("signal type '%s': %w", signalType, err))
		}

		close(idleConnsClosed)
	}()

	return idleConnsClosed
}
