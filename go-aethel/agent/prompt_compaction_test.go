// STATUS: DIAMANT VGT SUPREME
package agent

import "testing"

func TestCompactMessagesPreservesSingleSystemPrompt(t *testing.T) {
	messages := []map[string]interface{}{{"role": "system", "content": "CUSTOM PERSONA"}}
	for i := 0; i < compactThreshold+4; i++ {
		messages = append(messages, map[string]interface{}{"role": "user", "content": "message"})
	}
	summary, compacted := CompactMessagesPreservingSystem(messages)
	if summary == "" || len(compacted) == 0 {
		t.Fatal("expected compacted conversation with summary")
	}
	if compacted[0]["role"] != "system" || compacted[0]["content"] != "CUSTOM PERSONA" {
		t.Fatalf("system prompt was not preserved: %+v", compacted[0])
	}
	systemCount := 0
	for _, message := range compacted {
		if message["role"] == "system" {
			systemCount++
		}
	}
	if systemCount != 1 {
		t.Fatalf("expected exactly one system prompt, got %d", systemCount)
	}
}
