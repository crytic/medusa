package handlers

import (
	"encoding/json"
	"github.com/crytic/medusa/fuzzing"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
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

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
