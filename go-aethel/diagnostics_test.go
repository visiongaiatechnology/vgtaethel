// STATUS: DIAMANT VGT SUPREME
package main

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"go-aethel/agent"
	"go-aethel/handlers"
	"go-aethel/security"
)

func TestDiagnosticsBundleExcludesSensitiveFields(t *testing.T) {
	// HandleDiagnosticsExport reads handlers package state (not main.state).
	runs := agent.NewRunEngine(t.TempDir() + "/runs.sealed")
	handlers.InitState(
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
		nil, runs, nil, nil, nil, nil, nil, nil,
		func() string { return "" },
		func() string { return "" },
		func() string { return "" },
		func() string { return "" },
		func() string { return "" },
		func(string, string, string, string, string) error { return nil },
		func() []string { return nil },
		func() []security.MountGrant { return nil },
	)

	request := httptest.NewRequest("GET", "/v1/diagnostics/export", nil)
	recorder := httptest.NewRecorder()
	handlers.HandleDiagnosticsExport(recorder, request)
	if recorder.Code != 200 || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected diagnostics response: code=%d body=%s", recorder.Code, recorder.Body.String())
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
