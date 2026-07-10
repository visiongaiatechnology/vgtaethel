// STATUS: DIAMANT VGT SUPREME
package main

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagnosticsBundleExcludesSensitiveFields(t *testing.T) {
	if state == nil {
		state = &AppState{runs: NewRunEngine(t.TempDir() + "/runs.sealed")}
	} else if state.runs == nil {
		state.runs = NewRunEngine(t.TempDir() + "/runs.sealed")
	}
	request := httptest.NewRequest("GET", "/v1/diagnostics/export", nil)
	recorder := httptest.NewRecorder()
	handleDiagnosticsExport(recorder, request)
	if recorder.Code != 200 || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected diagnostics response: code=%d", recorder.Code)
	}
	reader, err := zip.NewReader(bytes.NewReader(recorder.Body.Bytes()), int64(recorder.Body.Len()))
	if err != nil || len(reader.File) != 1 {
		t.Fatalf("invalid diagnostics archive: files=%d err=%v", len(reader.File), err)
	}
	file, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	payload, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"system_prompt", "agent_messages", "objective", "tool_args", "result", "secret"} {
		if strings.Contains(strings.ToLower(string(payload)), forbidden) {
			t.Fatalf("diagnostics leaked forbidden field %q", forbidden)
		}
	}
}
