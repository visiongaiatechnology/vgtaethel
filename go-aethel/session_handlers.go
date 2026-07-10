package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const historyFile = "./vgt_workspace/chat_history.json"

func handleChatHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		data, err := os.ReadFile(historyFile)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	} else if r.Method == http.MethodPost {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var testArr []interface{}
		if err := json.Unmarshal(bodyBytes, &testArr); err != nil {
			http.Error(w, "Invalid JSON array", http.StatusBadRequest)
			return
		}

		_ = os.MkdirAll(filepath.Dir(historyFile), 0700)
		if err := os.WriteFile(historyFile, bodyBytes, 0600); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type SessionInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Timestamp string `json:"timestamp"`
}

func handleChatSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionsDir := "./vgt_workspace/sessions"
	_ = os.MkdirAll(sessionsDir, 0755)

	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var list []SessionInfo
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "session_") && strings.HasSuffix(f.Name(), ".json") {
			id := strings.TrimSuffix(strings.TrimPrefix(f.Name(), "session_"), ".json")

			filePath := filepath.Join(sessionsDir, f.Name())
			data, err := os.ReadFile(filePath)
			if err == nil {
				var msgs []map[string]interface{}
				title := "Leerer Impuls"
				if err := json.Unmarshal(data, &msgs); err == nil && len(msgs) > 0 {
					for _, m := range msgs {
						if r, ok := m["role"].(string); ok && (r == "user" || r == "assistant") {
							if val, ok := m["content"].(string); ok && val != "" {
								if len(val) > 28 {
									title = val[:25] + "..."
								} else {
									title = val
								}
								break
							}
						}
					}
				}

				info, _ := f.Info()
				list = append(list, SessionInfo{
					ID:        id,
					Title:     title,
					Timestamp: info.ModTime().Format("2006-01-02 15:04"),
				})
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Timestamp > list[j].Timestamp
	})

	if list == nil {
		list = []SessionInfo{}
	}

	json.NewEncoder(w).Encode(list)
}

func handleChatSessionsLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	filePath, err := sessionFilePath(id)
	if err != nil {
		http.Error(w, "Invalid session id", http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleChatSessionsSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID       string            `json:"id"`
		Messages []json.RawMessage `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	filePath, err := sessionFilePath(req.ID)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	marshaled, err := json.MarshalIndent(req.Messages, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(filePath, marshaled, 0600); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleChatSessionsDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	filePath, err := sessionFilePath(id)
	if err != nil {
		http.Error(w, "Invalid session id", http.StatusBadRequest)
		return
	}
	_ = os.Remove(filePath)

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
