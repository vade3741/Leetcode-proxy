package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/vade3741/Leetcode-proxy/internal/platform"
)

func resolveConfigurationValue(environmentKey string, fallbackDefaultValue string) string {
	resolvedValue := os.Getenv(environmentKey)
	if resolvedValue == "" {
		return fallbackDefaultValue
	}
	return resolvedValue
}

func resolveDurationConfiguration(environmentKey string, fallbackSecondCount int) time.Duration {
	parsedValueString := os.Getenv(environmentKey)
	if parsedValueString == "" {
		return time.Duration(fallbackSecondCount) * time.Second
	}
	secondCount, parseError := strconv.Atoi(parsedValueString)
	if parseError != nil {
		return time.Duration(fallbackSecondCount) * time.Second
	}
	return time.Duration(secondCount) * time.Second
}

func main() {
	serverPort := resolveConfigurationValue("SERVER_PORT", "8080")
	upstreamEndpointURL := resolveConfigurationValue("UPSTREAM_GRAPHQL_ENDPOINT", "https://leetcode.com/graphql")
	clientTimeoutDuration := resolveDurationConfiguration("HTTP_CLIENT_TIMEOUT_SECONDS", 10)
	serverReadTimeout := resolveDurationConfiguration("SERVER_READ_TIMEOUT_SECONDS", 5)
	serverWriteTimeout := resolveDurationConfiguration("SERVER_WRITE_TIMEOUT_SECONDS", 15)
	serverIdleTimeout := resolveDurationConfiguration("SERVER_IDLE_TIMEOUT_SECONDS", 60)

	gatewayClient := platform.NewGatewayClient(upstreamEndpointURL, clientTimeoutDuration)
	serviceRouter := platform.NewMerchantServiceRouter(gatewayClient)

	httpDaemon := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      serviceRouter,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	systemTerminationSignal := make(chan os.Signal, 1)
	signal.Notify(systemTerminationSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("merchant proxy listening on port %s", serverPort)
		if listenError := httpDaemon.ListenAndServe(); listenError != nil && !errors.Is(listenError, http.ErrServerClosed) {
			log.Fatalf("server terminated unexpectedly: %v", listenError)
		}
	}()

	<-systemTerminationSignal
	log.Println("system shutdown signal received, cleaning up connections")

	shutdownExecutionContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if terminationError := httpDaemon.Shutdown(shutdownExecutionContext); terminationError != nil {
		log.Fatalf("graceful termination failed: %v", terminationError)
	}

	log.Println("server shutdown completed successfully")
}
