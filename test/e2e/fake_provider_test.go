package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

type fakeProvider struct {
	server   *httptest.Server
	requests []json.RawMessage
	mu       sync.Mutex
}

func startFakeProvider() *fakeProvider {
	provider := &fakeProvider{}
	provider.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, request)
			return
		}

		var body json.RawMessage
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		provider.mu.Lock()
		provider.requests = append(provider.requests, body)
		firstRequest := len(provider.requests) == 1
		provider.mu.Unlock()

		chunks := []map[string]any{}
		if firstRequest {
			chunks = append(chunks,
				map[string]any{
					"id": "fixture-completion-1",
					"choices": []any{map[string]any{
						"index": 0,
						"delta": map[string]any{
							"role": "assistant",
							"tool_calls": []any{map[string]any{
								"index":     0,
								"id":        "fixture-tool-call",
								"type":      "function",
								"name":      "bash",
								"arguments": `{"command":"printf tool-ran"}`,
								"function": map[string]any{
									"name":      "bash",
									"arguments": `{"command":"printf tool-ran"}`,
								},
							}},
						},
						"finish_reason": nil,
					}},
				},
				map[string]any{
					"id": "fixture-completion-1",
					"choices": []any{map[string]any{
						"index": 0, "delta": map[string]any{}, "finish_reason": "tool_calls",
					}},
				},
			)
		} else {
			chunks = append(chunks,
				map[string]any{
					"id": "fixture-completion-2",
					"choices": []any{map[string]any{
						"index":         0,
						"delta":         map[string]any{"role": "assistant", "content": "fixture complete"},
						"finish_reason": nil,
					}},
				},
				map[string]any{
					"id": "fixture-completion-2",
					"choices": []any{map[string]any{
						"index": 0, "delta": map[string]any{}, "finish_reason": "stop",
					}},
				},
			)
		}
		chunks = append(chunks, map[string]any{
			"id": "fixture-usage", "choices": []any{},
			"usage": map[string]any{"prompt_tokens": 8, "completion_tokens": 2},
		})

		w.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range chunks {
			chunk["object"] = "chat.completion.chunk"
			chunk["created"] = 0
			chunk["model"] = "gpt-5"
			encoded, _ := json.Marshal(chunk)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(encoded)
			_, _ = w.Write([]byte("\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	return provider
}
