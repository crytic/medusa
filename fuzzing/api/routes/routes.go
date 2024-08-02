package routes

import (
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/api/handlers"
	"github.com/gorilla/mux"
)

func attachWebsocketRoutes(router *mux.Router, fuzzer *fuzzing.Fuzzer) {
	router.HandleFunc("/ws/env", handlers.WebsocketGetEnvHandler(fuzzer)).Methods("GET")
	router.HandleFunc("/ws/fuzzing", handlers.WebsocketGetFuzzingInfoHandler(fuzzer)).Methods("GET")
	router.HandleFunc("/ws/logs", handlers.WebsocketGetLogsHandler(fuzzer)).Methods("GET")
	router.HandleFunc("/ws/coverage", handlers.WebsocketGetCoverageInfoHandler(fuzzer)).Methods("GET")
	router.HandleFunc("/ws/corpus", handlers.WebsocketGetCorpusHandler(fuzzer)).Methods("GET")
	router.HandleFunc("/ws", handlers.WebsocketHandler(fuzzer)).Methods("GET")
}

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
	attachWebsocketRoutes(router, fuzzer)
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
