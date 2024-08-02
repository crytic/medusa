package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/crytic/medusa/compilation/platforms"
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/api/handlers"
	"github.com/crytic/medusa/fuzzing/api/middleware"
	"github.com/crytic/medusa/fuzzing/api/routes"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGetFileHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	router := initializeRouter(fuzzer)

	defer fuzzer.Stop()

	pathToTestFile := "testdata/test_contract.sol"

	req, err := http.NewRequest("GET", fmt.Sprintf("/file?path=%s", url.QueryEscape(pathToTestFile)), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Read the file content
	data, err := os.ReadFile(pathToTestFile)
	if err != nil {
		t.Fatal(err)
	}

	// Compare file content with the response body
	if bytes.Compare(rr.Body.Bytes(), data) != 0 {
		t.Errorf("handler returned wrong body: got %v want %v", rr.Body.String(), string(data))
	}
}

func TestEnvHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	router := initializeRouter(fuzzer)

	defer fuzzer.Stop()

	req, err := http.NewRequest("GET", "/env", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Read the response body
	var body map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &body)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := body["config"]; !ok {
		t.Fatalf("handler did not return config information: got %v", body)
	}
	if systemInterfaces, ok := body["system"].([]interface{}); ok {
		systemInfo := make([]string, len(systemInterfaces))
		for i, v := range systemInterfaces {
			systemInfo[i] = v.(string)
		}

		if !reflect.DeepEqual(systemInfo, os.Environ()) {
			t.Fatalf("handler returned wrong system information: got %v want %v", systemInfo, os.Environ())
		}
	} else {
		t.Fatalf("handler did not return system information: got %v", body)
	}

	if solcVersion, ok := body["solcVersion"]; ok {
		v, _ := platforms.GetSystemSolcVersion()
		if solcVersion != v.String() {
			t.Fatalf("handler returned wrong solc version information: got %v want %v", solcVersion, v.String())
		}
	} else {
		t.Fatalf("handler did not return solc version: got %v", body)
	}
}

func TestFuzzingHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	router := initializeRouter(fuzzer)

	defer fuzzer.Stop()

	req, err := http.NewRequest("GET", "/fuzzing", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Read the response body
	var body map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &body)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := body["metrics"]; !ok {
		t.Fatalf("handler did not return fuzzer metrics: got %v", body)
	}
	if _, ok := body["testCases"]; !ok {
		t.Fatalf("handler did not return fuzzer metrics: got %v", body)
	}
}

func TestLogsHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	router := initializeRouter(fuzzer)

	defer fuzzer.Stop()

	req, err := http.NewRequest("GET", "/logs", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Read the response body
	var body map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &body)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := body["logs"]; !ok {
		t.Fatalf("handler did not return logs: got %v", body)
	}
}

func TestCoverageHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	router := initializeRouter(fuzzer)

	defer fuzzer.Stop()

	req, err := http.NewRequest("GET", "/coverage", nil)
	if err != nil {
		t.Fatal(err)
	}

	if fuzzer.Corpus() == nil {
		// wait until corpus is initialized
		for fuzzer.Corpus() == nil {
			time.Sleep(time.Millisecond * 100)
		}
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var body map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &body)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := body["coverage"]; !ok {
		t.Fatalf("handler did not return coverage: got %v", body)
	}
}

func TestCorpusHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	router := initializeRouter(fuzzer)
	defer fuzzer.Stop()

	req, err := http.NewRequest("GET", "/corpus", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	if fuzzer.Corpus() == nil {
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check that we got the expected 'corpus not initialized' message
		respBody := strings.TrimSpace(rr.Body.String())
		expected := `{"error":"Corpus not initialized"}`
		if respBody != expected {
			t.Fatalf("handler returned unexpected body: got %v want %v", respBody, expected)
		}
	}

	for fuzzer.Corpus() == nil {
		time.Sleep(3 * time.Second)
	}

	rr = httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the body is a map containing only a "unexecutedCallSequences" field
	var body map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &body)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := body["unexecutedCallSequences"]; !ok {
		t.Fatalf("handler did not return unexecuted call sequences: got %v", body)
	}
}

func TestWebsocketHandler(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	go Start(fuzzer)

	defer fuzzer.Stop()

	tcs := []struct {
		name           string
		message        string
		expectedFields []string
	}{
		{
			name:           "WebsocketEnvHandler",
			message:        "env",
			expectedFields: []string{"config", "system", "solcVersion"},
		},
		{
			name:           "WebsocketFuzzingHandler",
			message:        "fuzzing",
			expectedFields: []string{"testCases"},
		},
		{
			name:           "WebsocketLogsHandler",
			message:        "logs",
			expectedFields: []string{"logs"},
		},
		{
			name:           "WebsocketCorpusHandler",
			message:        "corpus",
			expectedFields: []string{"unexecutedCallSequences"},
		},
		{
			name:           "WebsocketCoverageHandler",
			message:        "coverage",
			expectedFields: []string{"coverage"},
		},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, ws := newWSServer(t, handlers.WebsocketHandler(fuzzer))
			defer s.Close()
			defer ws.Close()

			sendMessage(t, ws, tt.message)

			reply := receiveWSMessage[map[string]any](t, ws)

			// Check that we get every expected field
			for _, field := range tt.expectedFields {
				if _, ok := reply[field]; !ok {
					t.Errorf("handler did not return %s: got %v", field, reply)
				}
			}
		})
	}
}

func TestWebsocketHandlers(t *testing.T) {
	fuzzer, err := initializeFuzzer()
	if err != nil {
		t.Fatal(err)
	}
	go Start(fuzzer)
	defer fuzzer.Stop()

	tcs := []struct {
		name           string
		handlerFunc    http.HandlerFunc
		expectedFields []string
	}{
		{
			name:           "WebsocketEnvHandler",
			handlerFunc:    handlers.WebsocketGetEnvHandler(fuzzer),
			expectedFields: []string{"config", "system", "solcVersion"},
		},
		{
			name:           "WebsocketFuzzingHandler",
			handlerFunc:    handlers.WebsocketGetFuzzingInfoHandler(fuzzer),
			expectedFields: []string{"testCases"},
		},
		{
			name:           "WebsocketLogsHandler",
			handlerFunc:    handlers.WebsocketGetLogsHandler(fuzzer),
			expectedFields: []string{"logs"},
		},
		{
			name:           "WebsocketCorpusHandler",
			handlerFunc:    handlers.WebsocketGetCorpusHandler(fuzzer),
			expectedFields: []string{"unexecutedCallSequences"},
		},
		{
			name:           "WebsocketCoverageHandler",
			handlerFunc:    handlers.WebsocketGetCoverageInfoHandler(fuzzer),
			expectedFields: []string{"coverage"},
		},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, ws := newWSServer(t, tt.handlerFunc)
			defer s.Close()
			defer ws.Close()

			reply := receiveWSMessage[map[string]any](t, ws)

			// Check that we get every expected field
			for _, field := range tt.expectedFields {
				if _, ok := reply[field]; !ok {
					t.Errorf("handler did not return %s: got %v", field, reply)
				}
			}
		})
	}
}

func TestNotFoundHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/not-found", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.NotFoundHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func initializeFuzzer() (*fuzzing.Fuzzer, error) {
	// Obtain default projectConfig
	projectConfig, err := config.GetDefaultProjectConfig("crytic-compile")
	if err != nil {
		return nil, err
	}

	// Update compilation target
	err = projectConfig.Compilation.SetTarget("testdata")
	if err != nil {
		return nil, err
	}
	projectConfig.Fuzzing.TargetContracts = []string{"TestContract"}
	projectConfig.ApiConfig.Enabled = true
	projectConfig.ApiConfig.WsUpdateInterval = 0.1

	fuzzer, err := fuzzing.NewFuzzer(*projectConfig)
	if err != nil {
		return nil, err
	}

	// Start the fuzzer
	go fuzzer.Start()

	return fuzzer, nil
}

func initializeRouter(fuzzer *fuzzing.Fuzzer) *mux.Router {
	// Create a new router
	router := mux.NewRouter()

	// Attach middleware
	middleware.AttachMiddleware(router)

	// Attach routes
	routes.AttachRoutes(router, fuzzer)

	return router
}

func testWebSocketHandler(t *testing.T, handlerFunc http.HandlerFunc, expectedFields []string, fuzzer *fuzzing.Fuzzer) {
	s, ws := newWSServer(t, handlerFunc)
	defer s.Close()
	defer ws.Close()

	reply := receiveWSMessage[map[string]any](t, ws)

	// Check that we get every expected field
	for _, field := range expectedFields {
		if _, ok := reply[field]; !ok {
			t.Errorf("handler did not return %s: got %v", field, reply)
		}
	}
}

func newWSServer(t *testing.T, h http.Handler) (*httptest.Server, *websocket.Conn) {
	t.Helper()

	s := httptest.NewServer(h)
	urlStr := httpToWs(t, s.URL)

	ws, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		t.Fatal(err)
	}

	return s, ws
}

func sendMessage(t *testing.T, ws *websocket.Conn, msg string) {
	t.Helper()

	if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		t.Fatalf("%v", err)
	}
}

func receiveWSMessage[T any](t *testing.T, ws *websocket.Conn) T {
	t.Helper()

	var reply T
	err := ws.ReadJSON(&reply)
	if err != nil {
		t.Fatalf("%v", err)
	}

	return reply
}

func httpToWs(t *testing.T, urlString string) string {
	t.Helper()

	u, err := url.Parse(urlString)
	if err != nil {
		t.Fatal(err)
	}

	u.Scheme = "ws"
	return u.String()
}
