package api

import (
	"fmt"
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/api/middleware"
	"github.com/crytic/medusa/fuzzing/api/routes"
	"github.com/crytic/medusa/logging"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func Start(fuzzer *fuzzing.Fuzzer) {
	port := fmt.Sprint(":", fuzzer.Config().ApiConfig.Port)

	if port == "" {
		port = ":8080" // Default port
	}

	// Create a new router
	router := mux.NewRouter()

	// Attach middleware
	middleware.AttachMiddleware(router)

	// Attach routes
	routes.AttachRoutes(router, fuzzer)

	// Get the fuzzer's custom sub-logger
	logger := logging.GlobalLogger.NewSubLogger("module", "api")

	var listener net.Listener
	var err error

	for i := 0; i < 10; i++ {
		listener, err = net.Listen("tcp", port)
		if err == nil {
			break
		}

		logger.Info("Server failed to start on port ", port[1:])
		port = incrementPort(port)
	}

	// Stop further execution if the server failed to start
	if listener == nil {
		logger.Error("Failed to start server: ", err)
		return
	}

	logger.Info("Server started on port ", port[1:])

	// Create a channel to receive interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the server in a separate goroutine
	serverErrorChan := make(chan error, 1)
	go func() {
		serverErrorChan <- http.Serve(listener, router)
	}()

	// Gracefully shutdown the server if the fuzzing context is cancelled or a server error is encountered
	select {
	case <-fuzzer.Context().Done():
		logger.Info("Shutting down server due to context cancellation")
		if err := listener.Close(); err != nil {
			logger.Error("Error closing listener: ", err)
		}
		break
	case err := <-serverErrorChan:
		logger.Error("Server error: ", err)
	}
}

func incrementPort(port string) string {
	var portNum int

	_, err := fmt.Sscanf(port, ":%d", &portNum)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(":%d", portNum+1)
}
