package frontend

import (
	"net/http"

	"github.com/satyajitnayk/load-balancer/serverpool"
)

const (
	RETRY_ATTEMPTED int = 0
)

// whether a retry attempt is allowed for an HTTP request based on the presence of a
// boolean value associated with the constant RETRY_ATTEMPTED in the request's context.
func AllowRetry(r *http.Request) bool {
	if _, ok := r.Context().Value(RETRY_ATTEMPTED).(bool); ok {
		return false
	}
	return true
}

type LoadBalancer interface {
	Serve(http.ResponseWriter, *http.Request)
}

type loadBalancer struct {
	serverPool serverpool.ServerPool
}

func (lb *loadBalancer) Serve(w http.ResponseWriter, r *http.Request) {
	peer := lb.serverPool.GetNextValidPeer()
	if peer != nil {
		peer.Serve(w, r)
		return
	}
	http.Error(w, "service not available", http.StatusServiceUnavailable)
}

// factory function that creates and returns a new instance of a
// type implementing the LoadBalancer interface.
func NewLoadBalancer(serverPool serverpool.ServerPool) LoadBalancer {
	return &loadBalancer{
		serverPool: serverPool,
	}
}
