package serverpool

import (
	"sync"

	"github.com/satyajitnayk/load-balancer/backend"
)

type lcServerPool struct {
	backends []backend.Backend
	mux      sync.RWMutex
}

func (s *lcServerPool) GetNextValidPeer() backend.Backend {
	var leastConnectedPeer backend.Backend

	// Check which one has the least number of active connections
	for _, b := range s.backends {
		if !b.IsAlive() {
			continue
		}

		if leastConnectedPeer == nil || b.GetActiveConnections() < leastConnectedPeer.GetActiveConnections() {
			leastConnectedPeer = b
		}
	}
	return leastConnectedPeer
}

func (s *lcServerPool) AddBackend(b backend.Backend) {
	s.backends = append(s.backends, b)
}

func (s *lcServerPool) GetServerPoolSize() int {
	return len(s.backends)
}

func (s *lcServerPool) GetBackends() []backend.Backend {
	return s.backends

}
