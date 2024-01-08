package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/satyajitnayk/load-balancer/backend"
	"github.com/satyajitnayk/load-balancer/frontend"
	"github.com/satyajitnayk/load-balancer/serverpool"
	"github.com/satyajitnayk/load-balancer/utils"
	"go.uber.org/zap"
)

func main() {
	logger := utils.InitLogger()
	//1. flush any buffered log entries and ensure that they are written
	// to the underlying log destination (file, console, etc.).
	//2. This is useful for making sure that all log entries are persisted
	// even if the program terminates unexpectedly
	defer logger.Sync()

	config, err := utils.GetLBConfig()
	if err != nil {
		utils.Logger.Fatal(err.Error())
	}
	// sets up a context (ctx) that will be canceled when the program receives
	// either an interrupt signal (os.Interrupt) or a termination signal (syscall.SIGTERM).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverPool, err := serverpool.NewServerPool(utils.GetLBStrategy(config.Strategy))
	if err != nil {
		utils.Logger.Fatal(err.Error())
	}

	loadBalancer := frontend.NewLoadBalancer(serverPool)

	for _, u := range config.Backends {
		endpoint, err := url.Parse(u)
		if err != nil {
			logger.Fatal(err.Error(), zap.String("URL", u))

		}

		rp := httputil.NewSingleHostReverseProxy(endpoint)
		backendServer := backend.NewBackend(endpoint, rp)

		// logs errors, updates the backend server's availability status, handles
		// retries based on the configured retry policy (frontend.AllowRetry), and logs
		// information about the retry attempt before initiating the retry through the load balancer
		rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("error handling the request",
				zap.String("host", endpoint.Host),
				zap.Error(err),
			)
			backendServer.SetAlive(false)

			if !frontend.AllowRetry(r) {
				utils.Logger.Info(
					"Max retry attempts reached, terminating",
					zap.String("address", r.RemoteAddr),
					zap.String("path", r.URL.Path),
				)

				http.Error(w, "Service not available", http.StatusServiceUnavailable)
				return
			}

			logger.Info(
				"Attempting retry",
				zap.String("address", r.RemoteAddr),
				zap.String("URL", r.URL.Path),
				zap.Bool("retry", true),
			)

			// creates a new context derived from the original context (r.Context()), but with
			// an additional key-value pair. The key is frontend.RETRY_ATTEMPTED and the value
			// is true, indicating that a retry attempt has been made.
			loadBalancer.Serve(
				w,
				r.WithContext(
					context.WithValue(r.Context(), frontend.RETRY_ATTEMPTED, true),
				),
			)
		}
		serverPool.AddBackend(backendServer)
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: http.HandlerFunc(loadBalancer.Serve),
	}

	go serverpool.LaunchHealthCheck(ctx, serverPool)

	// launches a goroutine to gracefully shut down an HTTP server upon receiving
	// a cancellation signal from a context, using a new context with a timeout for the shutdown.
	go func() {
		// Waits for the cancellation signal from the provided context (ctx).
		// The <-ctx.Done() line blocks until the context is canceled.
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() // Call cancel to avoid a context leak

		// The Shutdown method initiates a graceful shutdown of the server.
		// It allows existing connections to finish processing before the
		// server stops accepting new requests.
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}
	}()

	logger.Info(
		"Load Balancer started",
		zap.Int("port", config.Port),
	)

	// http.ErrServerClosed, it typically means that the server was intentionally shut down
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal("ListenAndServe() error", zap.Error(err))
	}
}
