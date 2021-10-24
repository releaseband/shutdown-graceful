package shutdown

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testModule struct {
	expErr error
	sleep  time.Duration
}

func newTestModule(sleep time.Duration, expErr error) *testModule {
	return &testModule{
		expErr: expErr,
		sleep:  sleep,
	}
}

func (s *testModule) Shutdown(_ context.Context) error {
	if s.sleep != 0 {
		time.Sleep(s.sleep)
	}

	return s.expErr
}

type testLogger struct {
	info   []string
	errors []error
}

func newLogger() *testLogger {
	return &testLogger{}
}

func (l *testLogger) Info(mess string) {
	l.info = append(l.info, mess)
}

func (l *testLogger) Error(err error) {
	l.errors = append(l.errors, err)
}

func expErr(t *testing.T, exp, got error) {
	if !errors.Is(got, exp) {
		t.Fatal("exp=", exp, "; got=", got)
	}
}

func cutAndCheckError(t *testing.T, logger *testLogger, exp error) {
	if len(logger.errors) == 0 {
		t.Fatal("logger.errors is nil")
	}

	gotErr := logger.errors[0]
	logger.errors = logger.errors[1:]

	expErr(t, exp, gotErr)
}

func expIsNil(t *testing.T, logger *testLogger, err error) {
	expErr(t, nil, err)

	if len(logger.errors) != 0 {
		t.Fatal("logger.errors is not nil")
	}
}

type testHelper struct {
	errShutdownFailed error
	defaultTimeout    time.Duration
	logger            *testLogger
}

func newTestHelper() *testHelper {
	return &testHelper{
		errShutdownFailed: errors.New("shutdown failed"),
		defaultTimeout:    1 * time.Second,
		logger:            newLogger(),
	}
}

func (tp testHelper) makeTimeouts() Timeouts {
	return Timeouts{
		Before:  200 * time.Nanosecond,
		Modules: tp.defaultTimeout,
		Server:  tp.defaultTimeout,
	}
}

func (tp testHelper) makeTimeoutModule() *testModule {
	return newTestModule(tp.defaultTimeout*2, nil)
}

func (tp testHelper) makeShutdownFailedTestModule() *testModule {
	return newTestModule(0, tp.errShutdownFailed)
}

func (tp testHelper) makeSuccessModule() *testModule {
	return newTestModule(0, nil)
}

func (tp *testHelper) makeGSA(server Shutdown, modules ...Shutdown) *gracefulShutdownApp {
	mList := make([]Shutdown, len(modules))
	copy(mList, modules)

	return NewGracefulShutdownApp(tp.makeTimeouts(), nil, server, mList, tp.logger)
}

func TestGracefulShutdownApp_Shutdown(t *testing.T) {
	t.Parallel()

	th := newTestHelper()

	t.Run("success", func(t *testing.T) {
		gsa := th.makeGSA(th.makeSuccessModule(), th.makeSuccessModule(), th.makeSuccessModule())
		expIsNil(t, th.logger, gsa.Shutdown())
	})
	t.Run("server shutdown failed", func(t *testing.T) {
		gsa := th.makeGSA(th.makeShutdownFailedTestModule(), th.makeSuccessModule(), th.makeSuccessModule())
		err := gsa.Shutdown()
		cutAndCheckError(t, th.logger, th.errShutdownFailed)
		expIsNil(t, th.logger, err)
	})
	t.Run("module shutdown timed out", func(t *testing.T) {
		gsa := th.makeGSA(th.makeSuccessModule(), th.makeTimeoutModule(), th.makeTimeoutModule(), th.makeSuccessModule())
		err := gsa.Shutdown()
		cutAndCheckError(t, th.logger, ErrTimedOut)
		expIsNil(t, th.logger, err)
	})
	t.Run("server and modules timed out", func(t *testing.T) {
		gsa := th.makeGSA(th.makeTimeoutModule(), th.makeTimeoutModule(), th.makeTimeoutModule(), th.makeTimeoutModule())
		err := gsa.Shutdown()
		expErr(t, ErrTimedOut, err)
		cutAndCheckError(t, th.logger, ErrTimedOut)
		cutAndCheckError(t, th.logger, ErrTimedOut)
		expIsNil(t, th.logger, nil)
	})
}
