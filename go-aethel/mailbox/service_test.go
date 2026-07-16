package mailbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-aethel/security"
)

func testConfig() Config {
	return Config{
		Enabled: true, Email: "operator@example.com", DisplayName: "Operator", Username: "operator@example.com",
		IMAPHost: "imap.example.com", IMAPPort: 993,
		SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPSecurity: "starttls",
	}
}

func TestMailConfigAndPasswordAreEncryptedAtRest(t *testing.T) {
	directory := t.TempDir()
	vaultPath := filepath.Join(directory, "vault.enc")
	vault, err := security.NewSecretVault(vaultPath, filepath.Join(directory, "vault.key"))
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(directory, "mail.enc")
	service := NewService(configPath, vault)
	password := "  exact app password  "
	if err := service.SaveConfig(testConfig(), password); err != nil {
		t.Fatal(err)
	}
	loaded, err := service.LoadConfig()
	if err != nil || loaded.Email != "operator@example.com" {
		t.Fatalf("mail config round trip failed: %+v %v", loaded, err)
	}
	secret, err := vault.GetToken(passwordSecretID)
	if err != nil || secret != password {
		t.Fatal("mail password was not preserved exactly")
	}
	for _, path := range []string{configPath, vaultPath} {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if strings.Contains(string(data), "operator@example.com") || strings.Contains(string(data), "exact app password") {
			t.Fatalf("sensitive mail data stored in plaintext at %s", path)
		}
	}
}

func TestMailValidationBlocksHeaderInjectionAndInsecureModes(t *testing.T) {
	config := testConfig()
	config.SMTPSecurity = "plain"
	if err := validateConfig(config); err == nil {
		t.Fatal("plaintext SMTP mode accepted")
	}
	if _, err := validateAddressList([]string{"victim@example.com\r\nBcc: attacker@example.com"}, 20); err == nil {
		t.Fatal("recipient header injection accepted")
	}
	config = testConfig()
	if _, err := buildMessage(config, []string{"victim@example.com"}, nil, "safe subject", "body"); err != nil {
		t.Fatalf("valid message encoding failed: %v", err)
	}
}
