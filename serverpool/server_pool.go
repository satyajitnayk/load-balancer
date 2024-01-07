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
	for _, b := range s.GetBackends() {
		b := b

		// Defining a context with timeout
		requesContext, stop := context.WithTimeout(ctx, 10*time.Second)
		defer stop()

		status := "up"
		go backend.IsBackendAlive(requesContext, aliveChannel, b.GetURL())

		select {
		case <-ctx.Done():
			utils.Logger.Info("Gracefully shutting down health check")
			return
		case alive := <-aliveChannel:
			b.SetAlive(alive)
			if !alive {
				status = "down"
			}
		}

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
