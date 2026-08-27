package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"go-aethel/mailbox"
	"go-aethel/personal"
)

func HandleMailFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		folders, err := service.ListFolders()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"folders": folders})
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if decodeStrictJSON(w, r, &req, 2<<10) != nil {
			return
		}
		if err := service.CreateFolder(req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleMailMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	messages, err := service.ListFolderMessages(r.URL.Query().Get("folder"), limit)
	if err != nil {
		http.Error(w, "mailbox synchronization failed", http.StatusBadGateway)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"messages": messages})
}

func HandleMailMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	uid64, err := strconv.ParseUint(r.URL.Query().Get("uid"), 10, 32)
	if err != nil || uid64 == 0 {
		http.Error(w, "invalid message uid", http.StatusBadRequest)
		return
	}
	message, err := service.GetMessage(r.URL.Query().Get("folder"), uint32(uid64))
	if err != nil {
		http.Error(w, "mail message could not be retrieved", http.StatusBadGateway)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"message": message, "calendar_candidates": mailbox.DetectCalendarCandidates(message)})
}

func HandleMailAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Action      string `json:"action"`
		Folder      string `json:"folder"`
		Destination string `json:"destination"`
		UID         uint32 `json:"uid"`
	}
	if decodeStrictJSON(w, r, &req, 4<<10) != nil {
		return
	}
	if req.Action != "move" {
		http.Error(w, "unsupported mail action", http.StatusBadRequest)
		return
	}
	if err := service.MoveMessage(req.Folder, req.UID, req.Destination); err != nil {
		http.Error(w, "mail action rejected", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "moved"})
}

func HandleMailPolicies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	if service.StoreError() != nil {
		http.Error(w, "encrypted mail archive integrity check failed", http.StatusConflict)
		return
	}
	store := service.Store()
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]any{"policies": store.ReplyPolicies()})
	case http.MethodPut:
		var req struct {
			Policies []mailbox.ReplyPolicy `json:"policies"`
		}
		if decodeStrictJSON(w, r, &req, 128<<10) != nil {
			return
		}
		if len(req.Policies) > 100 {
			http.Error(w, "reply policy limit exceeded", http.StatusBadRequest)
			return
		}
		for index := range req.Policies {
			policy := &req.Policies[index]
			policy.Name = personal.ClampPersonalText(policy.Name, 120)
			policy.RecipientPattern = personal.ClampPersonalText(strings.ToLower(policy.RecipientPattern), 320)
			policy.Instructions = personal.ClampPersonalText(policy.Instructions, 8000)
			policy.SystemPrompt = personal.ClampPersonalText(policy.SystemPrompt, 12000)
			policy.Category = personal.ClampPersonalText(policy.Category, 80)
			policy.ManualApproval = true
			policy.UpdatedAt = time.Now().UTC()
			if policy.ID == "" {
				policy.ID = secureMailID("policy_")
				if policy.ID == "" {
					http.Error(w, "reply policy identifier could not be generated", http.StatusInternalServerError)
					return
				}
			}
			if policy.Name == "" || policy.RecipientPattern == "" || policy.Instructions == "" || !validRecipientPattern(policy.RecipientPattern) {
				http.Error(w, "reply policy is invalid", http.StatusBadRequest)
				return
			}
		}
		if err := store.SaveReplyPolicies(req.Policies); err != nil {
			http.Error(w, "reply policies could not be sealed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleMailCalendar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	if service.StoreError() != nil {
		http.Error(w, "encrypted mail archive integrity check failed", http.StatusConflict)
		return
	}
	var req struct {
		Folder string    `json:"folder"`
		Title  string    `json:"title"`
		UID    uint32    `json:"uid"`
		DueAt  time.Time `json:"due_at"`
	}
	if decodeStrictJSON(w, r, &req, 8<<10) != nil {
		return
	}
	if req.UID == 0 || req.DueAt.IsZero() || req.DueAt.Before(time.Now().Add(-24*time.Hour)) {
		http.Error(w, "calendar event is invalid", http.StatusBadRequest)
		return
	}
	message, err := service.GetMessage(req.Folder, req.UID)
	if err != nil {
		http.Error(w, "linked mail is unavailable", http.StatusBadRequest)
		return
	}
	title := personal.ClampPersonalText(req.Title, 180)
	if title == "" {
		title = "E-Mail-Termin: " + personal.ClampPersonalText(message.Subject, 140)
	}
	eventID := secureMailID("mailcal_")
	if eventID == "" {
		http.Error(w, "calendar identifier could not be generated", http.StatusInternalServerError)
		return
	}
	event := mailbox.MailCalendarEvent{ID: eventID, MessageKey: mailbox.MessageReference(req.Folder, req.UID), Title: title, DueAt: req.DueAt.UTC(), CreatedAt: time.Now().UTC()}
	if err := service.Store().AddCalendarEvent(event); err != nil {
		http.Error(w, "calendar event could not be sealed", http.StatusInternalServerError)
		return
	}
	if operations != nil {
		deliverAt := req.DueAt.Add(-24 * time.Hour).UTC()
		if deliverAt.Before(time.Now().UTC()) {
			deliverAt = time.Now().UTC()
		}
		_, _ = operations.Enqueue(personal.OperationNotice{Kind: "mail_deadline", Priority: "high", Title: title, Body: "Frist aus E-Mail: " + message.Subject + "\nAbsender: " + message.From, Source: "mailbox", RequireAck: true, DeliverAt: deliverAt, Metadata: map[string]string{"mail_ref": event.MessageKey, "folder": req.Folder, "uid": strconv.FormatUint(uint64(req.UID), 10)}})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "created", "event": event})
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, target any, max int64) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, max))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid mail workspace request", http.StatusBadRequest)
		return err
	}
	return nil
}
func secureMailID(prefix string) string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}
	return prefix + hex.EncodeToString(value[:])
}
func validRecipientPattern(value string) bool {
	if strings.HasPrefix(value, "*@") {
		_, err := mail.ParseAddress("x" + value[1:])
		return err == nil
	}
	_, err := mail.ParseAddress(value)
	return err == nil
}
