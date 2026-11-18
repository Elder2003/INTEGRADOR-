// Command test-ai provides deterministic local responses for system tests.
// It is deliberately small and is not part of the production runtime.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const embeddingDimensions = 768

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/embeddings", embeddings)
	mux.HandleFunc("/v1/chat/completions", chatCompletions)
	mux.HandleFunc("/api/generate", generate)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("test AI listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func embeddings(w http.ResponseWriter, _ *http.Request) {
	vector := make([]float32, embeddingDimensions)
	vector[0] = 1
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data": []map[string]any{{"object": "embedding", "index": 0, "embedding": vector}},
	})
}

func chatCompletions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"id": "test-completion",
		"object": "chat.completion",
		"created": time.Now().Unix(),
		"model": "nextlab-test-model",
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]string{"role": "assistant", "content": "Respuesta determinista de pruebas."},
			"finish_reason": "stop",
		}},
		"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	})
}

func generate(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"model": "nextlab-test-model", "response": "ok", "done": true})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
