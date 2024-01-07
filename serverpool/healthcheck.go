package serverpool

import (
	"context"
	"time"

	"github.com/satyajitnayk/load-balancer/utils"
)

// periodically launches a health check on a server pool at fixed intervals
func LaunchHealthCheck(ctx context.Context, sp ServerPool) {
	// creates a new time.Ticker that will send time values on its channel at regular intervals of 20 sec
	t := time.NewTicker(time.Second * 20)
	utils.Logger.Info("Starting health check...")

	// Periodically launch health checks
	for {
		select {
		case <-t.C:
			// waits for the ticker signal (t.C)
			// Perform an action every 20 seconds
			go HealthCheck(ctx, sp)
		case <-ctx.Done():
			// Gracefully exit when the context is canceled
			utils.Logger.Info("Closing Health check")
			return
		}
	}
}
