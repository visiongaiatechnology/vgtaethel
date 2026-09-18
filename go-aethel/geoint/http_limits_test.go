package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"strings"
	"testing"
)

func TestReadBoundedBody(t *testing.T) {
	data, err := readBoundedBody(strings.NewReader("12345"), 5)
	if err != nil || string(data) != "12345" {
		t.Fatalf("expected exact-boundary payload, got %q, %v", data, err)
	}
	if _, err := readBoundedBody(strings.NewReader("123456"), 5); err == nil {
		t.Fatal("expected oversized upstream payload to be rejected")
	}
}

