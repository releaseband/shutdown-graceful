package shutdown

import "time"

const (
	defaultBeforeTimeout   = 3 * time.Second
	defaultShutdownTimeout = 15 * time.Second
)

type Timeouts struct {
	Before  time.Duration
	Modules time.Duration
	Server  time.Duration
}

func (t Timeouts) GetBeforeTimeout() time.Duration {
	if t.Before != 0 {
		return t.Before
	}

	return defaultBeforeTimeout
}

func (t Timeouts) GetServerTimeout() time.Duration {
	if t.Server != 0 {
		return t.Server
	}

	return defaultShutdownTimeout
}

func (t Timeouts) GetModuleTimeout() time.Duration {
	if t.Modules != 0 {
		return t.Modules
	}

	return defaultShutdownTimeout
}
