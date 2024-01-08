package serverpool

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/satyajitnayk/load-balancer/backend"
	"github.com/satyajitnayk/load-balancer/utils"
	"github.com/stretchr/testify/assert"
)

// an HTTP handler that introduces a 5-second delay when processing requests.
func SleepHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(5 * time.Second)
}

var (
	h   = http.HandlerFunc(SleepHandler)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w   = httptest.NewRecorder()
)

// test the behavior of a load balancer implementing the Least Connection algorithm
func TestLeastConnectionLB(t *testing.T) {
	dummyServer1 := httptest.NewServer(h)
	defer dummyServer1.Close()
	backend1URL, err := url.Parse(dummyServer1.URL)
	if err != nil {
		t.Fatal(err)
	}

	dummyServer2 := httptest.NewServer(h)
	defer dummyServer1.Close()
	backend2URL, err := url.Parse(dummyServer2.URL)
	if err != nil {
		t.Fatal(err)
	}

	rp1 := httputil.NewSingleHostReverseProxy(backend1URL)
	backend1 := backend.NewBackend(backend1URL, rp1)

	rp2 := httputil.NewSingleHostReverseProxy(backend2URL)
	backend2 := backend.NewBackend(backend2URL, rp2)

	serverPool, err := NewServerPool(utils.LeastConnected)
	if err != nil {
		t.Fatal(err)
	}

	serverPool.AddBackend(backend1)
	serverPool.AddBackend(backend2)

	// (test, expected, received)
	assert.Equal(t, 2, serverPool.GetServerPoolSize())

	// Using sync.WaitGroup in this way helps ensure that the test waits
	// for all concurrently executed goroutines to complete before moving
	// on to the next steps or finishing the test.
	var wg sync.WaitGroup
	wg.Add(1)

	peer := serverPool.GetNextValidPeer()
	t.Log(peer.GetURL().String())

	// Retrieves a peer from the server pool and asserts its existence.
	assert.NotNil(t, peer)

	// goroutines to simulate concurrent requests to the server pool.
	go func() {
		// The deferred function call decrements the WaitGroup counter by 1,
		// indicating that the goroutine has completed.
		defer wg.Done()
		peer.Serve(w, req)
	}()

	time.Sleep(1 * time.Second)
	peer2 := serverPool.GetNextValidPeer()
	t.Log(peer2.GetURL().String())
	connPeer2 := peer2.GetActiveConnections()

	assert.NotNil(t, peer)
	assert.Equal(t, 0, connPeer2)
	assert.NotEqual(t, peer, peer2)

	// Waits until the WaitGroup counter becomes zero, meaning that all
	// goroutines that were added with wg.Add have finished.
	wg.Wait()
}
