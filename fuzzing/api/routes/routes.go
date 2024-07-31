package routes

import "github.com/crytic/medusa/fuzzing"

func AttachRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	// Register routes
	// TODO

	// Route for serving files
	router.HandleFunc("/file", handlers.GetFileHandler).Methods("GET")

	// Catch-all 404 handler
	router.HandleFunc("/", handlers.NotFoundHandler)
}
