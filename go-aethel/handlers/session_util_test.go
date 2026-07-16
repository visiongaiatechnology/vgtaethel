package handlers

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSessionMessagesSealAndOpen(t *testing.T) {
	messages := []json.RawMessage{json.RawMessage(`{"role":"user","content":"private operator prompt"}`)}
	sealed, err := sealSessionMessages(messages)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("private operator prompt")) {
		t.Fatal("sealed session exposed plaintext")
	}
	opened, err := openSessionMessages(sealed)
	if err != nil || len(opened) != 1 || string(opened[0]) != string(messages[0]) {
		t.Fatalf("session open failed: %v %#v", err, opened)
	}
	legacy, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	opened, err = openSessionMessages(legacy)
	if err != nil || len(opened) != 1 {
		t.Fatalf("legacy session migration failed: %v", err)
	}
}
