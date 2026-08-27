package handlers

// STATUS: DIAMANT VGT SUPREME

import "testing"

func TestTextOnlyRequestNeverAttachesViewportScreenshot(t *testing.T) {
	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": `SHADOW_BATCH_JSON=[{"title":"Video shows conflict escalation","summary":"Browser report"}]`,
		},
	}

	if shouldAttachViewportScreenshot(ChatRequest{TextOnly: true}, "deepseek/deepseek-chat", messages) {
		t.Fatal("text-only SHADOW analysis attempted to attach a viewport screenshot")
	}
	if !shouldAttachViewportScreenshot(ChatRequest{}, "deepseek/deepseek-chat", messages) {
		t.Fatal("regression fixture does not exercise the automatic vision trigger")
	}
}
