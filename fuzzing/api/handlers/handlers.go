package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/crytic/medusa/compilation/platforms"
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/api/utils"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/logging"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type FuzzerHandler func(fuzzer *fuzzing.Fuzzer) http.HandlerFunc

func GetFileHandler(w http.ResponseWriter, r *http.Request) {
	// Get the file path from the URL query parameter "path"
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		err := json.NewEncoder(w).Encode(map[string]any{"error": "Missing file path parameter 'path'"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the file content to the response body
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetEnvHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get solc version
		v, _ := platforms.GetSystemSolcVersion()

		var env = map[string]any{
			"config":      fuzzer.Config(),
			"system":      os.Environ(),
			"solcVersion": v.String(),
		}
		err := json.NewEncoder(w).Encode(env)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func GetFuzzingInfoHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Write the test cases to the response
		err := json.NewEncoder(w).Encode(map[string]any{"metrics": fuzzer.Metrics(), "testCases": utils.MarshalTestCases(fuzzer.TestCases())})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func GetLogsHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	var logs bytes.Buffer

	// Obtain a reference to the logger used by the fuzzer
	fuzzerLogger := fuzzer.Logger()

	// Add a writer to the fuzzer logger
	fuzzerLogger.AddWriter(&logs, logging.UNSTRUCTURED, false)

	return func(w http.ResponseWriter, r *http.Request) {
		// Read the logs from the writer
		output := logs.String()

		// Encode the logs and send them as the response
		err := json.NewEncoder(w).Encode(map[string]string{"logs": output})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func GetCoverageInfoHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if fuzzer.Corpus() == nil {
			json.NewEncoder(w).Encode(map[string]any{"error": "Corpus not yet initialized"})
			return
		}

		sourceAnalysis, err := coverage.AnalyzeSourceCoverage(fuzzer.Compilations(), fuzzer.Corpus().CoverageMaps())
		if err != nil {
			json.NewEncoder(w).Encode(map[string]any{"error": "Corpus not yet initialized"})
			return
		}

		err = json.NewEncoder(w).Encode(map[string]any{"coverage": sourceAnalysis})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func GetCorpusHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		corpus := fuzzer.Corpus()

		var err error

		if corpus != nil {
			err = json.NewEncoder(w).Encode(map[string]any{"unexecutedCallSequences": corpus.UnexecutedCallSequences()})
		} else {
			response := map[string]string{"error": "Corpus not initialized"}
			err = json.NewEncoder(w).Encode(response)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func WebsocketHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	var logs bytes.Buffer

	// Obtain a reference to the logger used by the fuzzer
	fuzzerLogger := fuzzer.Logger()

	// Add a writer to the fuzzer logger
	fuzzerLogger.AddWriter(&logs, logging.UNSTRUCTURED, false)

	return func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, func(conn *websocket.Conn) {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Println(err)
					break
				}

				switch string(message) {
				case "env":
					// Get solc version
					v, _ := platforms.GetSystemSolcVersion()

					var env = map[string]any{
						"config":      fuzzer.Config(),
						"system":      os.Environ(),
						"solcVersion": v.String(),
					}

					err = conn.WriteJSON(env)
					break
				case "fuzzing":
					testCases := utils.MarshalTestCases(fuzzer.TestCases())

					err = conn.WriteJSON(map[string]any{"testCases": testCases})
					break
				case "logs":
					err = conn.WriteJSON(map[string]string{"logs": logs.String()})
					break
				case "coverage":
					if fuzzer.Corpus() == nil {
						err = conn.WriteJSON(map[string]string{"error": "Corpus not yet initialized"})
						break
					}

					sourceAnalysis, err := coverage.AnalyzeSourceCoverage(fuzzer.Compilations(), fuzzer.Corpus().CoverageMaps())
					if err != nil {
						err = conn.WriteJSON(map[string]string{"error": "Error analyzing source coverage"})
					}

					err = conn.WriteJSON(map[string]any{"coverage": sourceAnalysis})
					break
				case "corpus":
					err = conn.WriteJSON(map[string]any{"unexecutedCallSequences": fuzzer.Corpus().UnexecutedCallSequences()})
					break
				}
			}
		})
	}
}

func WebsocketGetEnvHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, func(conn *websocket.Conn) {
			wsUpdateInterval := time.Duration(fuzzer.Config().ApiConfig.WsUpdateInterval*1000) * time.Millisecond

			ticker := time.NewTicker(wsUpdateInterval)
			defer ticker.Stop()

			for range ticker.C {
				// Stop the ticker when the connection is closed
				if conn == nil {
					return
				}

				// Get solc version
				v, _ := platforms.GetSystemSolcVersion()

				var env = map[string]any{
					"config":      fuzzer.Config(),
					"system":      os.Environ(),
					"solcVersion": v.String(),
				}

				err := conn.WriteJSON(env)
				if err != nil {
					log.Println(err)
					break
				}
			}

			conn.Close()
		})
	}
}

func WebsocketGetFuzzingInfoHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, func(conn *websocket.Conn) {
			wsUpdateInterval := time.Duration(fuzzer.Config().ApiConfig.WsUpdateInterval*1000) * time.Millisecond

			ticker := time.NewTicker(wsUpdateInterval)
			defer ticker.Stop()

			for range ticker.C {
				// Stop the ticker when the connection is closed
				if conn == nil {
					return
				}

				testCases := utils.MarshalTestCases(fuzzer.TestCases())
				err := conn.WriteJSON(map[string]any{"testCases": testCases})
				if err != nil {
					log.Println(err)
					break
				}
			}

			conn.Close()
		})
	}
}

func WebsocketGetLogsHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	var logs bytes.Buffer

	// Obtain a reference to the logger used by the fuzzer
	fuzzerLogger := fuzzer.Logger()

	// Add a writer to the fuzzer logger
	fuzzerLogger.AddWriter(&logs, logging.UNSTRUCTURED, false)

	return func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, func(conn *websocket.Conn) {
			wsUpdateInterval := time.Duration(fuzzer.Config().ApiConfig.WsUpdateInterval*1000) * time.Millisecond

			ticker := time.NewTicker(wsUpdateInterval)
			defer ticker.Stop()

			for range ticker.C {
				// Stop the ticker when the connection is closed
				if conn == nil {
					return
				}

				err := conn.WriteJSON(map[string]string{"logs": logs.String()})
				if err != nil {
					log.Println(err)
					break
				}
			}

			conn.Close()
		})
	}
}

func WebsocketGetCoverageInfoHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, func(conn *websocket.Conn) {
			wsUpdateInterval := time.Duration(fuzzer.Config().ApiConfig.WsUpdateInterval*1000) * time.Millisecond

			ticker := time.NewTicker(wsUpdateInterval)
			defer ticker.Stop()

			for range ticker.C {
				// Stop the ticker when the connection is closed
				if conn == nil {
					return
				}

				sourceAnalysis, err := coverage.AnalyzeSourceCoverage(fuzzer.Compilations(), fuzzer.Corpus().CoverageMaps())
				if err != nil {
					conn.WriteJSON(map[string]string{"error": "Error analyzing source coverage"})
				}
				err = conn.WriteJSON(map[string]any{"coverage": sourceAnalysis})
				if err != nil {
					log.Println(err)
					break
				}
			}

			conn.Close()
		})
	}
}

func WebsocketGetCorpusHandler(fuzzer *fuzzing.Fuzzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, func(conn *websocket.Conn) {
			wsUpdateInterval := time.Duration(fuzzer.Config().ApiConfig.WsUpdateInterval*1000) * time.Millisecond

			ticker := time.NewTicker(wsUpdateInterval)
			defer ticker.Stop()

			for range ticker.C {
				// Stop the ticker when the connection is closed
				if conn == nil {
					return
				}

				err := conn.WriteJSON(map[string]any{"unexecutedCallSequences": fuzzer.Corpus().UnexecutedCallSequences()})
				if err != nil {
					log.Println(err)
					break
				}
			}

			conn.Close()
		})
	}
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, handler func(conn *websocket.Conn)) {
	// Upgrade the connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Close the connection
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}(conn)

	// Handle the WebSocket connection
	go handler(conn)

	// Wait for the connection to close
	<-r.Context().Done()
}
