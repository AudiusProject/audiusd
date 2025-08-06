package service

import (
	"context"

	"go.uber.org/zap"
)

type State int

const (
	StateNew State = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateFailed
)

// any type implements Message
type Message interface{}

type Service interface {
	// Initialize is called before Start and should initialize dependencies,
	// validate configuration, and prepare resources.
	Initialize(ctx context.Context) error

	// Start begins normal operation of the Service. Should return only after
	// the service is fully running.
	Start(ctx context.Context) error

	// AwaitStart returns a channel that will be closed when the service is running.
	// Allows dependent services to block until ready.
	AwaitStart() <-chan struct{}

	// Restart stops the service and starts it again. Implementations should
	// internally call Stop(), PreStart(), and Start() in sequence.
	Restart(ctx context.Context) error

	// Stop gracefully halts the service and releases resources.
	Stop(ctx context.Context) error

	// PostStop is called after Stop to perform any cleanup that should happen
	// once the service is fully stopped.
	PostStop(ctx context.Context) error

	// State returns the current lifecycle state of the service.
	State() State

	// IsRunning returns true if the service is in StateRunning.
	IsRunning() bool

	// LastError returns the last error encountered during lifecycle transitions.
	LastError() error

	// Dependencies returns a list of other services that must be started first.
	Dependencies() []Service

	// HealthCheck performs a quick check to determine if the service is
	// responsive and healthy. Should not block for long.
	HealthCheck(ctx context.Context) error

	// Fire-and-forget: send a message to the service.
	Tell(msg Message)

	// Request–response: send a message and wait for a reply or timeout.
	Ask(ctx context.Context, msg Message) (Message, error)

	// Sets the logger of the service, should be callable from any state.
	SetLogger(logger *zap.Logger)
}
