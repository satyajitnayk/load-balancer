package serverpool

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"

	"github.com/satyajitnayk/load-balancer/backend"
	"github.com/satyajitnayk/load-balancer/utils"
	"github.com/stretchr/testify/assert"
)

func TestPoolCreation(t *testing.T) {
	sp, _ := NewServerPool(utils.RoundRobin)
	url, _ := url.Parse("http://localhost:3333")
	b := backend.NewBackend(url, httputil.NewSingleHostReverseProxy(url))
	sp.AddBackend(b)

	assert.Equal(t, 1, sp.GetServerPoolSize())
}

// Test ensures that the ServerPool correctly handles concurrent calls
// to GetNextValidPeer and maintains the expected round-robin behavior
// when distributing requests among the available backend servers
func TestNextIndexIteration(t *testing.T) {
	sp, _ := NewServerPool(utils.RoundRobin)
	url, _ := url.Parse("http://localhost:3333")
	b1 := backend.NewBackend(url, httputil.NewSingleHostReverseProxy(url))
	sp.AddBackend(b1)

	url, _ = url.Parse("http://localhost:3334")
	b2 := backend.NewBackend(url, httputil.NewSingleHostReverseProxy(url))
	sp.AddBackend(b2)

	url, _ = url.Parse("http://localhost:3335")
	b3 := backend.NewBackend(url, httputil.NewSingleHostReverseProxy(url))
	sp.AddBackend(b3)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			sp.GetNextValidPeer()
		}
	}()

	// goroutines to simulate concurrent requests to the GetNextValidPeer method
	go func() {
		defer wg.Done()
		for i := 0; i < 2; i++ {
			sp.GetNextValidPeer()
		}
	}()

	// waits for both goroutines to complete before proceeding.
	wg.Wait()

	// This assertion checks if the round-robin behavior is maintained
	// after the concurrent calls to GetNextValidPeer
	assert.Equal(t, b1.GetURL().String(), sp.GetNextValidPeer().GetURL().String())
}
