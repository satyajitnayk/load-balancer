package serverpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/satyajitnayk/load-balancer/backend"
	"github.com/satyajitnayk/load-balancer/utils"
	"go.uber.org/zap"
)

// group a set of backend servers.
type ServerPool interface {
	GetBackends() []backend.Backend
	GetNextValidPeer() backend.Backend
	AddBackend(backend.Backend)
	GetServerPoolSize() int
}

type roundRobinServerPool struct {
	backends []backend.Backend
	mux      sync.RWMutex
	current  int
}

func (s *roundRobinServerPool) GetServerPoolSize() int {
	return len(s.backends)
}

// Rotate method increments the current count and returns the next server on the line
func (s *roundRobinServerPool) Rotate() backend.Backend {
	s.mux.Lock()
	s.current = (s.current + 1) % s.GetServerPoolSize()
	s.mux.Unlock()
	return s.backends[s.current]
}

// validate if the server is alive and able to receive requests,
// if that’s not the case, continues iteration until one is found.
func (s *roundRobinServerPool) GetNextValidPeer() backend.Backend {
	for i := 0; i < s.GetServerPoolSize(); i++ {
		nextPeer := s.Rotate()
		if nextPeer.IsAlive() {
			return nextPeer
		}
	}
	return nil
}

func (s *roundRobinServerPool) GetBackends() []backend.Backend {
	return s.backends
}

func (s *roundRobinServerPool) AddBackend(b backend.Backend) {
	s.backends = append(s.backends, b)
}

// Behind the scenes, we need a way to figure out
// if a certain server is responding or not. To determine if a backend is alive,
// a health check routine has been implemented, which checks continuously on
// every backend in the server pool
func HealthCheck(ctx context.Context, s ServerPool) {
	aliveChannel := make(chan bool, 1)

	// Iterate over the backends in the server pool
	for _, b := range s.GetBackends() {
		b := b

		// Defining a context with timeout
		// It limit the execution time of the health check for each backend.
		// This ensures that the health check does not block indefinitely.
		requesContext, stop := context.WithTimeout(ctx, 10*time.Second)

		// ensures that the context (requestContext) is canceled and resources
		// associated with it are cleaned up when the function exits, regardless of how it exits.
		defer stop()

		status := "up"

		// Perform a health check on the backend concurrently
		// The context is passed to the backend.IsBackendAlive function, allowing for proper cancellation propagation.
		go backend.IsBackendAlive(requesContext, aliveChannel, b.GetURL())

		// Wait for the health check result
		// The select statement is used to wait for either the overall context (ctx.Done())
		// to be canceled or the result of the health check (alive) to be received on the aliveChannel.
		select {
		case <-ctx.Done():
			// If the overall context is canceled, log and exit
			utils.Logger.Info("Gracefully shutting down health check")
			return
		case alive := <-aliveChannel:
			// Update the backend's alive status based on the health check result
			b.SetAlive(alive)
			if !alive {
				status = "down"
			}
		}

		// Log the result of the health check
		utils.Logger.Debug(
			"URL status",
			zap.String("URl", b.GetURL().String()),
			zap.String("status", status),
		)
	}
}

func NewServerPool(strategy utils.LBStrategy) (ServerPool, error) {
	switch strategy {
	case utils.RoundRobin:
		return &roundRobinServerPool{
			backends: make([]backend.Backend, 0),
			current:  0,
		}, nil

	case utils.LeastConnected:
		return &lcServerPool{
			backends: make([]backend.Backend, 0),
		}, nil
	default:
		return nil, fmt.Errorf("Invalid strategy")
	}
}
