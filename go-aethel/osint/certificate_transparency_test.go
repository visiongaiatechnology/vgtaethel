package osint

// STATUS: DIAMANT VGT SUPREME

import "testing"

func TestParseCertificateTransparencyScopesAndDeduplicatesDomain(t *testing.T) {
	payload := []byte(`[
      {"id":"ct-1","tbs_sha256":"aa","cert_sha256":"bb","dns_names":["example.org","*.api.example.org","unrelated.test"],"issuer":{"name":"Test CA"},"not_before":"2026-01-01T00:00:00Z","not_after":"2026-04-01T00:00:00Z"},
      {"id":"ct-1","dns_names":["duplicate.example.org"]},
      {"id":"ct-2","dns_names":["attacker-example.org"]}
    ]`)
	records, err := parseCertificateTransparency(payload, "example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ID != "ct-1" || len(records[0].DNSNames) != 2 || records[0].IssuerName != "Test CA" {
		t.Fatalf("unexpected scoped CT result: %+v", records)
	}
}

func TestParseCertificateTransparencyRejectsMalformedPayload(t *testing.T) {
	if _, err := parseCertificateTransparency([]byte(`{"unexpected":true}`), "example.org"); err == nil {
		t.Fatal("malformed certificate transparency payload was accepted")
	}
}
