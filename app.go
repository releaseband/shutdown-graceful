package shutdown

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ReadinessChecker interface {
	Shutdown()
}

type Server interface {
	Shutdown(ctx context.Context) error
}

type GracefulShutdownApp struct {
	ReadinessChecker      ReadinessChecker
	Server                Server
	BeforeShutdownTimeout time.Duration
	ShutdownTimeout       time.Duration
	ErrorHandler          func(err error)
	InformationHandler    func(mess string)
	Modules               map[string]func(ctx context.Context) error
}

func (c GracefulShutdownApp) Shutdown() error {
	if c.ReadinessChecker != nil {
		c.ReadinessChecker.Shutdown()
	}

	if c.InformationHandler != nil {
		c.InformationHandler("waiting before start shutdown graceful: " + c.BeforeShutdownTimeout.String())
	}

	time.Sleep(c.BeforeShutdownTimeout)

	ctx := context.Background()

	if c.ShutdownTimeout > 0 {
		var cancel func()

		ctx, cancel = context.WithTimeout(ctx, c.ShutdownTimeout)

		defer cancel()
	}

	count := len(c.Modules)
	if count > 0 {
		wg := &sync.WaitGroup{}
		wg.Add(len(c.Modules))

		for name, module := range c.Modules {
			go func(wg *sync.WaitGroup, name string, shutdown func(ctx context.Context) error) {
				if err := shutdown(ctx); err != nil {
					if c.ErrorHandler != nil {
						c.ErrorHandler(fmt.Errorf("module:%s, failed graceful shutdown: %w", name, err))
					}
				}

				wg.Done()
			}(wg, name, module)
		}

		wg.Wait()

		if c.Server != nil {
			if err := c.Server.Shutdown(ctx); err != nil {
				c.ErrorHandler(fmt.Errorf("server: failed graceful shutdown: %w", err))
			}
		}
	}

	return nil
}
