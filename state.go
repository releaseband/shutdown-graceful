package shutdown

import (
	"context"
	"fmt"
	"time"
)

type State struct {
	timeout      time.Duration
	notProcessed uint64
}

func NewState(timeout time.Duration) *State {
	return &State{
		timeout: timeout,
	}
}

func (s *State) Increment() {
	s.notProcessed++
}

func (s *State) Decrement() {
	if s.notProcessed > 0 {
		s.notProcessed--
	}
}

func (s State) iFinished() bool {
	return s.notProcessed == 0
}

func (s *State) Shutdown(ctx context.Context) error {
	if !s.iFinished() {
		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("shutdown context timeout done: %w", ctx.Err())
			default:
				if s.iFinished() {
					return nil
				}
			}

			time.Sleep(s.timeout)
		}
	}

	return nil
}
