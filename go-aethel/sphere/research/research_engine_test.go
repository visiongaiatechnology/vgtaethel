// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/research/research_engine_test.go

package research

import (
	"strings"
	"testing"

	"go-aethel/sphere"
)

func TestResearchEngineProjectClipAndBriefing(t *testing.T) {
	dir := t.TempDir()
	store, err := sphere.NewObjectStore(dir)
	if err != nil {
		t.Fatalf("NewObjectStore failed: %v", err)
	}

	engine := NewEngine(store)

	proj, err := engine.CreateProject("Quantencomputing Architektur", "Recherche zu Qiskit und Cirq Frameworks", []string{"quantum", "hardware"})
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	clip, err := engine.AddClip(
		proj.ID,
		"IBM Quantum Heron Prozessor",
		"https://ibm.com/quantum",
		"IBM Quantum Computing News",
		"5000 Qubits mit 5-facher Fehlerreduktion angekündigt.",
		"Neue Heron-Architektur liefert signifikante Fehlertoleranz.",
		"Aethel Browser",
	)
	if err != nil {
		t.Fatalf("AddClip failed: %v", err)
	}
	if clip.ID == "" {
		t.Fatal("expected non-empty clip ID")
	}

	briefing, err := engine.GenerateBriefing(proj.ID)
	if err != nil {
		t.Fatalf("GenerateBriefing failed: %v", err)
	}
	if !strings.Contains(briefing, "IBM Quantum Heron Prozessor") || !strings.Contains(briefing, "Wichtigste Erkenntnisse") {
		t.Fatalf("briefing content missing expected sections: %s", briefing)
	}
}
