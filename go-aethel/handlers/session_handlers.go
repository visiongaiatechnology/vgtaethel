package handlers

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"

	"go-aethel/security"
)

const historyFile = "./vgt_workspace/chat_history.json"

func handleChatHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		data, sealed, err := security.ReadSealedFile(historyFile)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
			return
		}
		if !sealed {
			_ = security.WriteSealedFile(historyFile, data)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	} else if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
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

		if err := security.WriteSealedFile(historyFile, bodyBytes); err != nil {
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

	root, err := openSessionsRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer root.Close()
	files, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		http.Error(w, "Session store unavailable", http.StatusInternalServerError)
		return
	}

	var list []SessionInfo
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "session_") && strings.HasSuffix(f.Name(), ".json") {
			id := strings.TrimSuffix(strings.TrimPrefix(f.Name(), "session_"), ".json")

			data, err := root.ReadFile(f.Name())
			if err == nil {
				var rawMessages []json.RawMessage
				title := "Leerer Impuls"
				if rawMessages, err = openSessionMessages(data); err == nil && len(rawMessages) > 0 {
					for _, raw := range rawMessages {
						var m map[string]interface{}
						if json.Unmarshal(raw, &m) != nil {
							continue
						}
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

				info, infoErr := root.Stat(f.Name())
				if infoErr != nil {
					continue
				}
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

	fileName, err := sessionFileName(id)
	if err != nil {
		http.Error(w, "Invalid session id", http.StatusBadRequest)
		return
	}
	root, err := openSessionsRoot()
	if err != nil {
		http.Error(w, "Session store unavailable", http.StatusInternalServerError)
		return
	}
	defer root.Close()
	data, err := root.ReadFile(fileName)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}

	messages, err := openSessionMessages(data)
	if err != nil {
		http.Error(w, "Session data cannot be opened", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(messages)
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

	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	fileName, err := sessionFileName(req.ID)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	marshaled, err := sealSessionMessages(req.Messages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	root, err := openSessionsRoot()
	if err != nil {
		http.Error(w, "Session store unavailable", http.StatusInternalServerError)
		return
	}
	defer root.Close()
	if err := root.WriteFile(fileName, marshaled, 0600); err != nil {
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

	fileName, err := sessionFileName(id)
	if err != nil {
		http.Error(w, "Invalid session id", http.StatusBadRequest)
		return
	}
	root, err := openSessionsRoot()
	if err != nil {
		http.Error(w, "Session store unavailable", http.StatusInternalServerError)
		return
	}
	defer root.Close()
	if err := root.Remove(fileName); err != nil && !os.IsNotExist(err) {
		http.Error(w, "Session could not be removed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
