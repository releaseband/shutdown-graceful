package shutdown

import (
	"context"
	"sync/atomic"
	"time"
)

const checkStateTimeout = 200 * time.Millisecond

type State struct {
	notProcessed int64
}

func NewState() *State {
	return &State{}
}

func (s *State) Increment() {
	atomic.AddInt64(&s.notProcessed, +1)
}

func (s *State) get() int64 {
	return atomic.LoadInt64(&s.notProcessed)
}

func (s *State) Decrement() {
	if s.get() > 0 {
		atomic.AddInt64(&s.notProcessed, -1)
	}
}

func (s State) iFinished() bool {
	return s.get() == 0
}

func (s *State) Shutdown(ctx context.Context) int64 {
	if !s.iFinished() {
		for {
			select {
			case <-ctx.Done():
				break
			default:
				if s.iFinished() {
					break
				}
			}

			time.Sleep(checkStateTimeout)
		}
	}

	return s.get()
}
