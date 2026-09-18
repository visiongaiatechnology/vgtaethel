// STATUS: DIAMANT VGT SUPREME
package handlers

import (
	"bytes"
	"encoding/json"
	"go-aethel/security"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeWorkspaceRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := SetCodeWorkspaceRoot(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ClearCodeWorkspaceRoot)
	request := httptest.NewRequest(http.MethodGet, "/v1/code/workspace/file?path=..%2Foutside.txt", nil)
	response := httptest.NewRecorder()
	HandleCodeWorkspaceFile(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected traversal rejection, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCodeWorkspaceDoesNotSelectInternalRuntimeData(t *testing.T) {
	ClearCodeWorkspaceRoot()
	request := httptest.NewRequest(http.MethodGet, "/v1/code/workspace/tree", nil)
	response := httptest.NewRecorder()
	HandleCodeWorkspaceTree(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected explicit project selection requirement, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCodeWorkspaceWriteReadAndSnapshot(t *testing.T) {
	root := t.TempDir()
	if err := SetCodeWorkspaceRoot(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ClearCodeWorkspaceRoot)
	previousSecurityState := securityStateForCodeTest(root)
	defer previousSecurityState()
	relative := filepath.ToSlash(filepath.Join("handler-test-"+strings.ReplaceAll(t.Name(), "/", "-"), "main.go"))

	body, err := json.Marshal(map[string]string{"path": relative, "content": "package main\n"})
	if err != nil {
		t.Fatal(err)
	}
	writeRequest := httptest.NewRequest(http.MethodPut, "/v1/code/workspace/file", bytes.NewReader(body))
	writeRequest.Header.Set("Content-Type", "application/json")
	writeResponse := httptest.NewRecorder()
	HandleCodeWorkspaceFile(writeResponse, writeRequest)
	if writeResponse.Code != http.StatusOK {
		t.Fatalf("write failed with %d: %s", writeResponse.Code, writeResponse.Body.String())
	}
	var writePayload codeFilePayload
	if err := json.Unmarshal(writeResponse.Body.Bytes(), &writePayload); err != nil {
		t.Fatal(err)
	}
	if writePayload.SnapshotID == "" {
		t.Fatal("expected a sealed recovery snapshot")
	}

	readRequest := httptest.NewRequest(http.MethodGet, "/v1/code/workspace/file?path="+relative, nil)
	readResponse := httptest.NewRecorder()
	HandleCodeWorkspaceFile(readResponse, readRequest)
	if readResponse.Code != http.StatusOK {
		t.Fatalf("read failed with %d: %s", readResponse.Code, readResponse.Body.String())
	}
	var readPayload codeFilePayload
	if err := json.Unmarshal(readResponse.Body.Bytes(), &readPayload); err != nil {
		t.Fatal(err)
	}
	if readPayload.Content != "package main\n" || readPayload.Path != relative {
		t.Fatalf("unexpected read payload: %#v", readPayload)
	}
}

func securityStateForCodeTest(root string) func() {
	canonicalRoot, err := security.CanonicalDir(root)
	if err != nil {
		panic(err)
	}
	security.InitState(func(path string, access security.MountAccess) bool {
		canonical, err := security.CanonicalTarget(path)
		return err == nil && (access == security.MountRead || access == security.MountWrite) && security.IsPathInside(canonicalRoot, canonical)
	})
	return func() { security.InitState(nil) }
}
