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

type TurnOffReadinessChecker interface {
	Shutdown()
}

type Shutdown interface {
	Shutdown(ctx context.Context) error
}

type Logger interface {
	Info(mess string)
	Error(err error)
}

type Application interface {
	Shutdown() error
	ListenShutdownSignals() <-chan struct{}
}

type app struct {
	timeouts         Timeouts
	readinessChecker TurnOffReadinessChecker
	logger           Logger
	modules          []Shutdown
	server           Shutdown
}

func NewApplication(
	timeouts Timeouts, checker TurnOffReadinessChecker, server Shutdown, modules []Shutdown, logger Logger) *app {
	return &app{
		timeouts:         timeouts,
		readinessChecker: checker,
		logger:           logger,
		modules:          modules,
		server:           server,
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

func (g *app) waitBeforeShutdown() {
	t := g.timeouts.GetBeforeTimeout()

	g.logger.Info("waiting for the termination of running processes" + t.String())
	time.Sleep(t)
}

func (g *app) shutdownModules() {
	t := g.timeouts.GetModuleTimeout()

	g.logger.Info("timeout for shutdown modules = " + t.String())

	if err := shutdown(t, g.logger, g.modules...); err != nil {
		g.logger.Error(fmt.Errorf("modules shutdown failed: %w", err))
	}
}

func (g *app) shutdownServer() error {
	t := g.timeouts.GetServerTimeout()

	g.logger.Info("timeout for shutdown server = " + t.String())

	err := shutdown(t, g.logger, g.server)
	if err != nil {
		g.logger.Error(fmt.Errorf("server shutdown failed: %w", err))
	}

	return err
}

func (g *app) Shutdown() error {
	if g.readinessChecker != nil {
		g.readinessChecker.Shutdown()
	}

	g.waitBeforeShutdown()
	g.shutdownModules()

	return g.shutdownServer()
}

func (g *app) ListenShutdownSignals() <-chan struct{} {
	idleConnsClosed := make(chan struct{}, 1)

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		s := <-sigint

		signalType := "signal type: " + s.String()

		g.logger.Info(signalType)

		if err := g.Shutdown(); err != nil {
			g.logger.Error(fmt.Errorf("signal type '%s': %w", signalType, err))
		}

		close(idleConnsClosed)
	}()

	return idleConnsClosed
}
