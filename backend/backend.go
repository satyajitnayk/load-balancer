package backend

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend interface {
	SetAlive(bool) // serve the backend status
	IsAlive() bool // check the backend status
	GetURL() *url.URL
	GetActiveConnections() int
	// relay the request to the corresponding URL using a reverse proxy
	Serve(http.ResponseWriter, *http.Request)
}

type backend struct {
	url   *url.URL
	alive bool
	// serves to avoid having race conditions while performing operations on the other attributes.
	mux          sync.RWMutex
	connections  int
	reverseProxy *httputil.ReverseProxy
}

func (b *backend) GetActiveConnections() int {
	// The RLock() - Read Lock allows multiple goroutines to read a shared resource simultaneously.
	// It ensures that concurrent readers don't interfere with each other and provides
	// a level of concurrency.
	b.mux.RLock()
	connections := b.connections
	b.mux.RUnlock()
	return connections
}

func (b *backend) SetAlive(alive bool) {
	// The Lock() - Write Lock provides exclusive access to a shared resource,
	// allowing only one goroutine to modify it at a time. It ensures that no
	// other goroutines can read or write while the write lock is held,
	// ensuring consistency and preventing conflicts
	b.mux.Lock()
	b.alive = alive
	b.mux.Unlock()
}

func (b *backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.alive
	defer b.mux.RUnlock()
	return alive
}

func (b *backend) GetURL() *url.URL {
	return b.url
}

// The Serve method manages connection count, proxies incoming HTTP requests,
// and guarantees proper decrementing even on errors, using mutex locks for
// atomic updates and the defer statement for cleanup.
func (b *backend) Serve(rw http.ResponseWriter, req *http.Request) {
	// anonymous function (closure) that is immediately executed
	defer func() {
		b.mux.Lock()
		b.connections--
		b.mux.Unlock()
	}()

	b.mux.Lock()
	b.connections++
	b.mux.Unlock()

	// The ServeHTTP method of a reverse proxy typically takes care of forwarding the
	// incoming request to the appropriate backend server, receiving the response,
	// and then writing that response back to the original client through
	// the provided http.ResponseWriter.
	b.reverseProxy.ServeHTTP(rw, req)
}

// returns a pointer to the created backend instance
// it's a factory function that creates and returns a new instance of a
// type implementing the Backend interface
func NewBackend(u *url.URL, rp *httputil.ReverseProxy) Backend {
	return &backend{
		url:          u,
		alive:        true,
		reverseProxy: rp,
	}
}
