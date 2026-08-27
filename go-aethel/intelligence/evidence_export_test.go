package intelligence

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"sort"
	"testing"
)

func TestDocumentImportVaultAndSignedCaseExport(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "canonical.json"), nil)
	raw := []byte("Original operator-provided incident report with immutable bytes.\n")
	imported, err := store.ImportDocument(raw, "text", "operator-import", "incident.txt", "general")
	if err != nil {
		t.Fatal(err)
	}
	if imported.SnapshotID != bytesSHA256(raw) || imported.CaptureScope != "original" {
		t.Fatalf("document import did not retain original acquisition provenance: %+v", imported)
	}
	storedRaw, snapshot, err := store.vault.Read(imported.SnapshotID)
	if err != nil || !bytes.Equal(storedRaw, raw) || snapshot.RawSHA256 != imported.RawSHA256 {
		t.Fatalf("immutable vault round-trip failed: snapshot=%+v err=%v", snapshot, err)
	}

	caseRecord, err := store.CreateCase("Forensic export", "Verify signed evidence package")
	if err != nil {
		t.Fatal(err)
	}
	snapshotState := store.GetSnapshot()
	if len(snapshotState.Evidence) != 1 {
		t.Fatalf("expected one imported evidence reference, got %d", len(snapshotState.Evidence))
	}
	worldEvidence := snapshotState.Evidence[0]
	sealed, err := store.SealCaseEvidence(caseRecord.ID, worldEvidence.ID, worldEvidence.SourceID, worldEvidence.URL, worldEvidence.Excerpt, worldEvidence.SHA256, "")
	if err != nil {
		t.Fatal(err)
	}
	if sealed.SnapshotID != imported.SnapshotID || sealed.CaptureScope != "original" {
		t.Fatalf("case promotion lost original snapshot provenance: %+v", sealed)
	}
	store.UpsertSource(Source{ID: "operator-import", Name: "Operator import", SourceType: "local", TrustTier: 1})
	claim, err := store.AddClaim(Claim{CaseID: caseRecord.ID, Subject: "incident report", Predicate: "documents", Object: "incident", Statement: "The imported report documents the incident.", AssertingSourceID: "operator-import", SourceNature: "primary", PassageIDs: []string{"passage-" + imported.ObservationIDs[0]}, SupportingEvidenceIDs: []string{sealed.ID}, Confidence: 85, CalibrationBasis: "Direct primary document"})
	if err != nil {
		t.Fatal(err)
	}
	claim, err = store.ReviewClaim(claim.ID, "verified", "analyst-a", "Primary source and sealed evidence reviewed")
	if err != nil || claim.ReviewedBy != "analyst-a" || claim.Status != "verified" {
		t.Fatalf("claim review failed: %+v %v", claim, err)
	}

	archiveBytes, manifest, err := store.ExportCaseEvidence(caseRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyEvidenceExportManifest(manifest); err != nil {
		t.Fatalf("signed manifest failed verification: %v", err)
	}
	if len(manifest.Evidence) != 1 || manifest.Evidence[0].CaptureScope != "original" || manifest.Evidence[0].ArchivePath == "" {
		t.Fatalf("export manifest lost forensic scope: %+v", manifest)
	}

	reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		stream, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		payload, readErr := io.ReadAll(stream)
		_ = stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[file.Name] = payload
	}
	if !bytes.Equal(files[manifest.Evidence[0].ArchivePath], raw) {
		t.Fatal("signed archive does not contain the original raw snapshot")
	}
	if bytesSHA256(files["report.md"]) != manifest.ReportSHA256 || bytesSHA256(files["passages.json"]) != manifest.PassagesSHA256 {
		t.Fatal("report or passage catalog is not bound into the signed manifest")
	}
	if !bytes.Contains(files["report.md"], []byte(claim.ID)) || !bytes.Contains(files["report.md"], []byte(sealed.ID)) || !bytes.Contains(files["report.md"], []byte("Contradicting evidence: none")) {
		t.Fatalf("claim-level report citations are incomplete: %s", files["report.md"])
	}
	var archivedManifest EvidenceExportManifest
	if err := json.Unmarshal(files["manifest.json"], &archivedManifest); err != nil {
		t.Fatal(err)
	}
	if err := VerifyEvidenceExportManifest(archivedManifest); err != nil {
		t.Fatalf("archived manifest signature failed verification: %v", err)
	}
	if _, err := VerifyEvidenceExportArchive(archiveBytes); err != nil {
		t.Fatalf("complete signed archive verification failed: %v", err)
	}
	files["report.md"] = []byte("tampered report")
	var tamperedArchive bytes.Buffer
	tamperedWriter := zip.NewWriter(&tamperedArchive)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := writeEvidenceArchiveFile(tamperedWriter, name, files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := tamperedWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyEvidenceExportArchive(tamperedArchive.Bytes()); err == nil {
		t.Fatal("archive verifier accepted a report modified after signing")
	}

	tampered := archivedManifest
	tampered.CaseSHA256 = bytesSHA256([]byte("tampered"))
	if VerifyEvidenceExportManifest(tampered) == nil {
		t.Fatal("tampered evidence manifest was accepted")
	}
}
