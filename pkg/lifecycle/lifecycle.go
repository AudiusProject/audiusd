package lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AudiusProject/audiusd/pkg/common"
	"golang.org/x/sync/errgroup"
)

// Lifecycle formally manages various long-running goroutines
// for smooth cleanup when restarting major components (e.g. mediorum).
// This allows us to wait for all registered goroutines on a service to
// gracefully shut down before restarting the service.
type Lifecycle struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     *common.Logger
	children   []*Lifecycle
	isShutDown atomic.Bool
}

func NewLifecycle(ctx context.Context, name string, logger *common.Logger) *Lifecycle {
	new_ctx, cancel := context.WithCancel(ctx)
	return &Lifecycle{
		ctx:      new_ctx,
		cancel:   cancel,
		logger:   logger.Child(name),
		children: []*Lifecycle{},
	}
}

func NewFromLifecycle(lc *Lifecycle, name string) *Lifecycle {
	if lc.isShutDown.Load() {
		panic("attempting to derive new lifecycle from already shut down lifecycle")
	}
	newLc := NewLifecycle(lc.ctx, name, lc.logger)
	lc.children = append(lc.children, newLc)
	return newLc
}

func (l *Lifecycle) AddManagedRoutine(name string, f func(context.Context)) {
	if l.isShutDown.Load() {
		panic("attempting to add managed routine to already shut down lifecycle")
	}
	l.logger.Infof("Starting managed goroutine '%s'", name)
	l.wg.Add(1)
	defer l.wg.Done()
	go func() {
		defer l.logger.Infof("Shutdown managed goroutine '%s'", name)
		f(l.ctx)
	}()
}

func (l *Lifecycle) ShutdownWithTimeout(timeout time.Duration) error {
	l.cancel()
	l.isShutDown.Store(true)
	done := make(chan error, 1)

	eg := errgroup.Group{}

	for _, child := range l.children {
		eg.Go(func() error {
			return child.ShutdownWithTimeout(timeout)
		})
	}
	eg.Go(func() error {
		l.wg.Wait()
		return nil
	})

	go func() {
		done <- eg.Wait()
	}()

	l.logger.Info("Lifecycle shutdown signaled. Waiting for managed goroutines to finish...")
	select {
	case err := <-done:
		l.logger.Info("Lifecycle shutdown complete")
		return err
	case <-time.After(timeout):
		l.logger.Info("Lifecycle shutdown timed out")
		return errors.New("lifecycle shutdown timed out")
	}
}
