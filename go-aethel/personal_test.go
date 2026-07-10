package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersonalStoreMemoryLifecycle(t *testing.T) {
	store := NewPersonalStore(filepath.Join(t.TempDir(), "personal"))

	saved, err := store.AppendMemory(PersonalMemory{
		Type:       "preference",
		Content:    "Der Nutzer schaut gern lange YouTube-Videos.",
		Confidence: 0.9,
		Source:     "unit-test",
	})
	if err != nil {
		t.Fatalf("AppendMemory failed: %v", err)
	}
	if saved.ID == "" || saved.CreatedAt.IsZero() {
		t.Fatalf("saved memory missing generated metadata: %+v", saved)
	}

	saved.Content = "Der Nutzer bevorzugt ruhige Video-Steuerung per Sprache."
	updated, err := store.UpdateMemory(saved)
	if err != nil {
		t.Fatalf("UpdateMemory failed: %v", err)
	}
	if updated.Content != saved.Content {
		t.Fatalf("updated content mismatch: %q", updated.Content)
	}

	memories, err := store.ListMemories()
	if err != nil {
		t.Fatalf("ListMemories failed: %v", err)
	}
	if len(memories) != 1 || memories[0].Content != saved.Content {
		t.Fatalf("unexpected memories after update: %+v", memories)
	}

	if err := store.DeleteMemory(saved.ID); err != nil {
		t.Fatalf("DeleteMemory failed: %v", err)
	}
	memories, err = store.ListMemories()
	if err != nil {
		t.Fatalf("ListMemories after delete failed: %v", err)
	}
	if len(memories) != 0 {
		t.Fatalf("expected empty memories after delete, got %+v", memories)
	}
}

func TestPersonalCoreContextIncludesConfiguredTraits(t *testing.T) {
	store := NewPersonalStore(filepath.Join(t.TempDir(), "personal"))
	if err := store.SaveConfig(PersonalConfig{Enabled: true, HumorLevel: 42, HonestyLevel: 97, InitiativeLevel: 68}); err != nil {
		t.Fatal(err)
	}
	context := BuildPersonalContext(store)
	for _, expected := range []string{"AETHEL PERSONAL CORE", "Humorgrad: 42/100", "Ehrlichkeitsgrad: 97/100", "Initiative: 68/100"} {
		if !strings.Contains(context, expected) {
			t.Fatalf("personal core context missing %q: %s", expected, context)
		}
	}
}

func TestPersonalStoreSealsProfileAndMemoriesAtRest(t *testing.T) {
	base := filepath.Join(t.TempDir(), "personal")
	store := NewPersonalStore(base)
	profile := PersonalProfile{DisplayName: "Masterboard", Notes: "private personal profile value"}
	if err := store.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	if _, err := store.AppendMemory(PersonalMemory{Type: "preference", Content: "private memory value", Source: "test"}); err != nil {
		t.Fatalf("AppendMemory failed: %v", err)
	}
	profileBytes, err := os.ReadFile(filepath.Join(base, "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	memoryBytes, err := os.ReadFile(filepath.Join(base, "memories.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(profileBytes, []byte("private personal profile value")) || bytes.Contains(memoryBytes, []byte("private memory value")) {
		t.Fatal("personal data was written in plaintext")
	}
	loaded, err := store.LoadProfile()
	if err != nil || loaded.DisplayName != "Masterboard" {
		t.Fatalf("sealed profile could not be loaded: %+v err=%v", loaded, err)
	}
}

func TestPersonalSetupFallbackRejectsSecretLikeProfileFields(t *testing.T) {
	profile := fallbackPersonalProfileFromSetup(map[string]string{
		"display_name":    "Masterboard",
		"preferred_tone":  "direkt und locker",
		"assistant_style": "wie ein Bro, der Kontext behält",
		"interests":       "Coding, YouTube, Musik",
		"goals":           "Aethel als persönlichen Assistenten ausbauen",
		"notes":           "kein secret hier",
	})

	clean := clampPersonalProfileField("sk-this-looks-secret", profile.Notes, 4000)
	if clean != profile.Notes {
		t.Fatalf("secret-like generated field should fall back, got %q", clean)
	}
	if profile.DisplayName != "Masterboard" || len(profile.Interests) == 0 || len(profile.Goals) == 0 {
		t.Fatalf("fallback profile was not built from setup answers: %+v", profile)
	}
}

func TestNormalizePersonalSetupQuestionsUsesAllowedTargetsAndFallbacks(t *testing.T) {
	questions := normalizePersonalSetupQuestions([]PersonalSetupQuestion{
		{ID: "display_name", Target: "wrong-target", Question: "Wie soll ich dich nennen?"},
		{ID: "display_name", Target: "personal-display-name", Question: "Duplicate should be ignored"},
		{ID: "password", Target: "personal-notes", Question: "Was ist dein Passwort?"},
	})

	if len(questions) != 7 {
		t.Fatalf("expected full 7-question setup, got %d: %+v", len(questions), questions)
	}
	if questions[0].ID != "display_name" || questions[0].Target != "personal-display-name" {
		t.Fatalf("allowed target normalization failed: %+v", questions[0])
	}
	seen := map[string]bool{}
	for _, q := range questions {
		if seen[q.ID] {
			t.Fatalf("duplicate setup question id emitted: %+v", questions)
		}
		seen[q.ID] = true
		if q.Target == "wrong-target" || q.ID == "password" {
			t.Fatalf("unsafe question survived normalization: %+v", q)
		}
	}
}
