package mailbox

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-aethel/security"
)

func TestSecureMailStoreEncryptsSignsAndVerifies(t *testing.T) {
	directory := t.TempDir()
	dataPath := filepath.Join(directory, "mail_store.enc")
	keyPath := filepath.Join(directory, "mail_key.enc")
	store := NewSecureMailStore(dataPath, keyPath)
	message := StoredMessage{Folder: "INBOX", UID: 42, From: "sender@example.com", Subject: "Confidential invoice", TextBody: "Payment due 30.09.2026", Date: time.Now().UTC()}
	if err := store.UpsertMessages([]StoredMessage{message}); err != nil {
		t.Fatalf("store: %v", err)
	}
	raw, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), message.Subject) || strings.Contains(string(raw), message.TextBody) {
		t.Fatal("mail plaintext leaked into encrypted store")
	}
	reloaded := NewSecureMailStore(dataPath, keyPath)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("verified reload: %v", err)
	}
	if loaded, ok := reloaded.Message("INBOX", 42); !ok || loaded.Subject != message.Subject {
		t.Fatalf("message round trip failed: %+v", loaded)
	}
}

func TestSecureMailStoreRejectsMLDSATampering(t *testing.T) {
	directory := t.TempDir()
	dataPath := filepath.Join(directory, "mail_store.enc")
	keyPath := filepath.Join(directory, "mail_key.enc")
	store := NewSecureMailStore(dataPath, keyPath)
	if err := store.UpsertMessages([]StoredMessage{{Folder: "INBOX", UID: 1, Subject: "Original", TextBody: "Body"}}); err != nil {
		t.Fatal(err)
	}
	plaintext, _, err := security.ReadSealedFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	var envelope signedEnvelope
	if json.Unmarshal(plaintext, &envelope) != nil {
		t.Fatal("decode envelope")
	}
	payload, _ := base64.StdEncoding.DecodeString(envelope.Payload)
	var data secureMailData
	if json.Unmarshal(payload, &data) != nil {
		t.Fatal("decode payload")
	}
	item := data.Messages[messageKey("INBOX", 1)]
	item.Subject = "Tampered"
	data.Messages[messageKey("INBOX", 1)] = item
	forged, _ := json.Marshal(data)
	envelope.Payload = base64.StdEncoding.EncodeToString(forged)
	modified, _ := json.Marshal(envelope)
	if err := security.WriteSealedFile(dataPath, modified); err != nil {
		t.Fatal(err)
	}
	if err := NewSecureMailStore(dataPath, keyPath).Load(); err == nil {
		t.Fatal("tampered ML-DSA envelope accepted")
	}
}

func TestSpamAndDeadlineAssessment(t *testing.T) {
	message := StoredMessage{From: "Billing <billing@example.com>", Subject: "Zahlungsaufforderung dringend", TextBody: "Bitte zahlbar bis 30.09.2026. https://example.com/pay", Headers: map[string]string{"Authentication-Results": "spf=fail dkim=fail dmarc=fail"}}
	assessment := AssessSpam(message)
	if assessment.Class != "spam" || assessment.Score < 70 {
		t.Fatalf("spam assessment too weak: %+v", assessment)
	}
	candidates := DetectCalendarCandidates(message)
	if len(candidates) != 1 || candidates[0].DueAt.Day() != 30 || candidates[0].DueAt.Month() != time.September {
		t.Fatalf("deadline detection failed: %+v", candidates)
	}
}

func TestReplyPolicyMatching(t *testing.T) {
	if !ReplyPolicyMatches(ReplyPolicy{RecipientPattern: "*@kanzlei.de"}, "Legal <kontakt@kanzlei.de>") {
		t.Fatal("domain policy did not match")
	}
	if ReplyPolicyMatches(ReplyPolicy{RecipientPattern: "boss@example.com"}, "attacker@example.com") {
		t.Fatal("unrelated sender matched exact policy")
	}
}
