package provider

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// Ollama probes must never block the model registry or health endpoints.
// Windows + "localhost" can stall on IPv6/half-open sockets; use a hard
// context deadline, 127.0.0.1, and a short TTL cache.

const ollamaProbeTimeout = 250 * time.Millisecond
const ollamaCacheTTL = 5 * time.Second

var (
	ollamaCacheMu     sync.Mutex
	ollamaCacheAt     time.Time
	ollamaCacheModels []map[string]interface{}
	ollamaCacheOK     bool
)

func GetLocalOllamaModels() []map[string]interface{} {
	ollamaCacheMu.Lock()
	if ollamaCacheOK && time.Since(ollamaCacheAt) < ollamaCacheTTL {
		out := append([]map[string]interface{}(nil), ollamaCacheModels...)
		ollamaCacheMu.Unlock()
		return out
	}
	ollamaCacheMu.Unlock()

	models := probeLocalOllamaModels()

	ollamaCacheMu.Lock()
	ollamaCacheModels = append([]map[string]interface{}(nil), models...)
	ollamaCacheAt = time.Now()
	ollamaCacheOK = true
	ollamaCacheMu.Unlock()

	return models
}

func probeLocalOllamaModels() []map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), ollamaProbeTimeout)
	defer cancel()

	// Force IPv4 loopback — avoids Windows localhost/::1 stalls.
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: ollamaProbeTimeout}
			return d.DialContext(ctx, "tcp4", "127.0.0.1:11434")
		},
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   ollamaProbeTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:11434/api/tags", nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	list := make([]map[string]interface{}, 0, len(result.Models))
	for _, m := range result.Models {
		if m.Name == "" {
			continue
		}
		list = append(list, map[string]interface{}{
			"id":             "ollama/" + m.Name,
			"name":           m.Name,
			"provider":       "Ollama",
			"tier":           "Local",
			"cost_input_1m":  0.0,
			"cost_output_1m": 0.0,
			"supports_tools": false,
			"supports_vision": false,
		})
	}
	return list
}
