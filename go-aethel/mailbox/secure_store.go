package mailbox

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"go-aethel/security"
)

const mailSignatureContext = "VGT-AETHEL-MAIL-STORE-v1"

type StoredMessage struct {
	AccountID   string            `json:"account_id"`
	Folder      string            `json:"folder"`
	UID         uint32            `json:"uid"`
	MessageID   string            `json:"message_id"`
	From        string            `json:"from"`
	To          []string          `json:"to,omitempty"`
	CC          []string          `json:"cc,omitempty"`
	Subject     string            `json:"subject"`
	Date        time.Time         `json:"date"`
	TextBody    string            `json:"text_body"`
	HTMLBody    string            `json:"html_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Unread      bool              `json:"unread"`
	SpamScore   int               `json:"spam_score"`
	SpamClass   string            `json:"spam_class"`
	SpamReasons []string          `json:"spam_reasons,omitempty"`
	SyncedAt    time.Time         `json:"synced_at"`
}

type ReplyPolicy struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	RecipientPattern string    `json:"recipient_pattern"`
	Instructions     string    `json:"instructions"`
	SystemPrompt     string    `json:"system_prompt,omitempty"`
	Category         string    `json:"category"`
	Enabled          bool      `json:"enabled"`
	ManualApproval   bool      `json:"manual_approval"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MailCalendarEvent struct {
	ID         string    `json:"id"`
	MessageKey string    `json:"message_key"`
	Title      string    `json:"title"`
	DueAt      time.Time `json:"due_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type secureMailData struct {
	Version        int                      `json:"version"`
	Messages       map[string]StoredMessage `json:"messages"`
	ReplyPolicies  []ReplyPolicy            `json:"reply_policies"`
	CalendarEvents []MailCalendarEvent      `json:"calendar_events"`
}

type signedEnvelope struct {
	Version   int    `json:"version"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type SecureMailStore struct {
	mu       sync.RWMutex
	dataPath string
	keyPath  string
	data     secureMailData
}

func NewSecureMailStore(dataPath, keyPath string) *SecureMailStore {
	return &SecureMailStore{dataPath: dataPath, keyPath: keyPath, data: secureMailData{Version: 1, Messages: map[string]StoredMessage{}}}
}

func (s *SecureMailStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	plaintext, _, err := security.ReadSealedFile(s.dataPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, public, err := s.loadOrCreateKey(false)
	if err != nil {
		return err
	}
	var envelope signedEnvelope
	if json.Unmarshal(plaintext, &envelope) != nil || envelope.Version != 1 || envelope.Algorithm != "ML-DSA-65" {
		return errors.New("mail store envelope is invalid")
	}
	payload, payloadErr := base64.StdEncoding.DecodeString(envelope.Payload)
	signature, signatureErr := base64.StdEncoding.DecodeString(envelope.Signature)
	encodedPublic, publicErr := base64.StdEncoding.DecodeString(envelope.PublicKey)
	if payloadErr != nil || signatureErr != nil || publicErr != nil || subtle.ConstantTimeCompare(encodedPublic, public.Bytes()) != 1 {
		return errors.New("mail store identity verification failed")
	}
	if !mldsa65.Verify(public, payload, []byte(mailSignatureContext), signature) {
		return errors.New("mail store ML-DSA signature verification failed")
	}
	var decoded secureMailData
	if json.Unmarshal(payload, &decoded) != nil || decoded.Version != 1 {
		return errors.New("mail store payload is invalid")
	}
	if decoded.Messages == nil {
		decoded.Messages = map[string]StoredMessage{}
	}
	s.data = decoded
	return nil
}

func (s *SecureMailStore) UpsertMessages(messages []StoredMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Messages == nil {
		s.data.Messages = map[string]StoredMessage{}
	}
	for _, message := range messages {
		message.SyncedAt = time.Now().UTC()
		s.data.Messages[messageKey(message.Folder, message.UID)] = message
	}
	return s.saveLocked()
}

func (s *SecureMailStore) Message(folder string, uid uint32) (StoredMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	message, ok := s.data.Messages[messageKey(folder, uid)]
	return message, ok
}

func (s *SecureMailStore) SaveReplyPolicies(policies []ReplyPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.ReplyPolicies = append([]ReplyPolicy(nil), policies...)
	return s.saveLocked()
}

func (s *SecureMailStore) ReplyPolicies() []ReplyPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]ReplyPolicy(nil), s.data.ReplyPolicies...)
}

func ReplyPolicyMatches(policy ReplyPolicy, sender string) bool {
	parsed, err := mail.ParseAddress(strings.TrimSpace(sender))
	if err != nil {
		return false
	}
	address := strings.ToLower(parsed.Address)
	pattern := strings.ToLower(strings.TrimSpace(policy.RecipientPattern))
	if strings.HasPrefix(pattern, "*@") {
		return strings.HasSuffix(address, pattern[1:])
	}
	return subtle.ConstantTimeCompare([]byte(address), []byte(pattern)) == 1
}

func (s *SecureMailStore) AddCalendarEvent(event MailCalendarEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.CalendarEvents = append(s.data.CalendarEvents, event)
	return s.saveLocked()
}

func (s *SecureMailStore) saveLocked() error {
	private, public, err := s.loadOrCreateKey(true)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	signature := make([]byte, mldsa65.SignatureSize)
	if err := mldsa65.SignTo(private, payload, []byte(mailSignatureContext), true, signature); err != nil {
		return errors.New("mail store ML-DSA signing failed")
	}
	envelope := signedEnvelope{
		Version: 1, Algorithm: "ML-DSA-65",
		PublicKey: base64.StdEncoding.EncodeToString(public.Bytes()),
		Payload:   base64.StdEncoding.EncodeToString(payload),
		Signature: base64.StdEncoding.EncodeToString(signature),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return security.WriteSealedFile(s.dataPath, encoded)
}

func (s *SecureMailStore) loadOrCreateKey(create bool) (*mldsa65.PrivateKey, *mldsa65.PublicKey, error) {
	encoded, _, err := security.ReadSealedFile(s.keyPath)
	if err == nil {
		seedBytes, decodeErr := base64.StdEncoding.DecodeString(string(encoded))
		if decodeErr != nil || len(seedBytes) != mldsa65.SeedSize {
			return nil, nil, errors.New("mail signing key is invalid")
		}
		var seed [mldsa65.SeedSize]byte
		copy(seed[:], seedBytes)
		public, private := mldsa65.NewKeyFromSeed(&seed)
		return private, public, nil
	}
	if !errors.Is(err, os.ErrNotExist) || !create {
		return nil, nil, errors.New("mail signing key is unavailable")
	}
	if _, statErr := os.Stat(s.dataPath); statErr == nil {
		return nil, nil, errors.New("mail signing key missing for existing store")
	}
	var seed [mldsa65.SeedSize]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(s.keyPath), 0700); err != nil {
		return nil, nil, err
	}
	if err := security.WriteSealedFile(s.keyPath, []byte(base64.StdEncoding.EncodeToString(seed[:]))); err != nil {
		return nil, nil, err
	}
	public, private := mldsa65.NewKeyFromSeed(&seed)
	return private, public, nil
}

func messageKey(folder string, uid uint32) string { return folder + ":" + fmt.Sprint(uid) }
