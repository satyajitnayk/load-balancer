package backend

import (
	"context"
	"net"
	"net/url"

	"github.com/satyajitnayk/load-balancer/utils"
	"go.uber.org/zap"
)

// checks the reachability of a backend by attempting to establish a TCP connection
// to a specified URL. It uses a goroutine to perform the check asynchronously and
// communicates the result (whether the backend is alive or not) through a channel.
func IsBackendAlive(ctx context.Context, aliveChannel chan bool, u *url.URL) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		utils.Logger.Debug("Site unreachable", zap.Error(err))
		aliveChannel <- false
		return
	}
	// Close the connection (cleanup)
	_ = conn.Close()

	// If the connection is successfully established and closed, signal that the site is reachable
	aliveChannel <- true
}
