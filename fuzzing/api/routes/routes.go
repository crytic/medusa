package routes

import (
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/api/handlers"
	"github.com/gorilla/mux"
)

func attachEnvRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	router.HandleFunc("/env", handlers.GetEnvHandler(fuzzer)).Methods("GET")
}

func attachFuzzingRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	router.HandleFunc("/fuzzing", handlers.GetFuzzingInfoHandler(fuzzer)).Methods("GET")
}

func attachLogsRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	router.HandleFunc("/logs", handlers.GetLogsHandler(fuzzer)).Methods("GET")
}

func attachCoverageRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	router.HandleFunc("/coverage", handlers.GetCoverageInfoHandler(fuzzer)).Methods("GET")
}

func attachCorpusRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	router.HandleFunc("/corpus", handlers.GetCorpusHandler(fuzzer)).Methods("GET")
}

func AttachRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	// Register routes
	attachEnvRoutes(router, fuzzer)
	attachFuzzingRoutes(router, fuzzer)
	attachLogsRoutes(router, fuzzer)
	attachCoverageRoutes(router, fuzzer)
	attachCorpusRoutes(router, fuzzer)

	// Route for serving files
	router.HandleFunc("/file", handlers.GetFileHandler).Methods("GET")

	// Catch-all 404 handler
	router.HandleFunc("/", handlers.NotFoundHandler)
}
