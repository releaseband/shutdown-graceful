package graceful

import (
	"net"
	"sync"
	"sync/atomic"
)

// GracefulListener defines a listener that we can gracefully stop
type GracefulListener struct {
	// inner listener
	ln net.Listener

	wg sync.WaitGroup

	metrics
}

var _ net.Listener = (*GracefulListener)(nil)

// NewGracefulListener wraps the given listener into 'graceful shutdown' listener.
func NewGracefulListener(ln net.Listener) *GracefulListener {
	return &GracefulListener{ln: ln}
}

// Accept creates a conn
func (ln *GracefulListener) Accept() (net.Conn, error) {
	c, err := ln.ln.Accept()
	if err != nil {
		return nil, err
	}

	ln.wg.Add(1)
	ln.recordAccept()

	return &gracefulConn{
		Conn:     c,
		doneFunc: func() { ln.wg.Done(); ln.recordClose() },
	}, nil
}

// Addr returns the listen address
func (ln *GracefulListener) Addr() net.Addr {
	return ln.ln.Addr()
}

// Close closes the listener. Any blocked Accept operations will be unblocked
// and return errors.
//
// Unlike other listeners
func (ln *GracefulListener) Close() error {
	if err := ln.ln.Close(); err != nil {
		return err
	}

	ln.wg.Wait() // Wait for all connections to close

	return nil
}

func (ln *GracefulListener) Metrics() (opened, closed uint64) { return ln.get() }

type gracefulConn struct {
	net.Conn
	doneFunc func()
}

func (c *gracefulConn) Close() error {
	defer c.doneFunc()

	return c.Conn.Close()
}

type metrics struct{ opened, closed uint64 }

func (m *metrics) recordAccept()         { atomic.AddUint64(&m.opened, 1) }
func (m *metrics) recordClose()          { atomic.AddUint64(&m.opened, 1) }
func (m *metrics) get() (uint64, uint64) { return m.opened, m.closed }
