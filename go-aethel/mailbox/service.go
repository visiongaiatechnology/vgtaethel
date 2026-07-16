package mailbox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	messageMail "github.com/emersion/go-message/mail"
	"go-aethel/security"
)

const passwordSecretID = "mail-account-password"

var hostPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)

var SharedService *Service

type Config struct {
	Enabled      bool   `json:"enabled"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name,omitempty"`
	Username     string `json:"username"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPSecurity string `json:"smtp_security"` // tls or starttls
}

type Message struct {
	UID     uint32    `json:"uid"`
	From    string    `json:"from"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Preview string    `json:"preview"`
	Unread  bool      `json:"unread"`
}

type OutgoingMessage struct {
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

type Service struct {
	configPath string
	vault      *security.SecretVault
}

func NewService(configPath string, vault *security.SecretVault) *Service {
	return &Service{configPath: configPath, vault: vault}
}

func (s *Service) LoadConfig() (Config, error) {
	if s == nil || s.vault == nil {
		return Config{}, errors.New("mail service unavailable")
	}
	data, _, err := security.ReadSealedFile(s.configPath)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil {
		return Config{}, errors.New("mail configuration is invalid")
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Service) SaveConfig(cfg Config, password string) error {
	if s == nil || s.vault == nil {
		return errors.New("mail service unavailable")
	}
	if err := validateConfig(cfg); err != nil {
		return err
	}
	if password != "" {
		if len(password) < 8 || len(password) > 1024 {
			return errors.New("mail password length is invalid")
		}
		if err := s.vault.Add(security.SecretItem{
			ID: passwordSecretID, Service: "mail", Type: "password", Token: password,
			AllowedActions: []string{"imap.read", "smtp.send"},
			AllowedTargets: []string{cfg.IMAPHost, cfg.SMTPHost}, RequiresApproval: true,
			RotationHint: "Rotate when the provider reports a credential event.",
		}); err != nil {
			return err
		}
	} else if _, err := s.vault.GetToken(passwordSecretID); err != nil {
		return errors.New("mail password is required for initial setup")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return errors.New("mail configuration could not be encoded")
	}
	return security.WriteSealedFile(s.configPath, data)
}

func (s *Service) DeleteConfig() error {
	if s == nil || s.vault == nil {
		return errors.New("mail service unavailable")
	}
	if err := s.vault.Delete(passwordSecretID); err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if err := os.Remove(s.configPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Service) ListMessages(limit int) ([]Message, error) {
	cfg, password, err := s.credentials()
	if err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	dialer := &net.Dialer{Timeout: 12 * time.Second, KeepAlive: 20 * time.Second}
	client, err := imapclient.DialWithDialerTLS(dialer, net.JoinHostPort(cfg.IMAPHost, fmt.Sprint(cfg.IMAPPort)), tlsConfig(cfg.IMAPHost))
	if err != nil {
		return nil, errors.New("secure IMAP connection failed")
	}
	client.Timeout = 20 * time.Second
	defer client.Logout()
	if err := client.Login(cfg.Username, password); err != nil {
		return nil, errors.New("IMAP authentication failed")
	}
	selected, err := client.Select("INBOX", true)
	if err != nil {
		return nil, errors.New("IMAP inbox selection failed")
	}
	if selected.Messages == 0 {
		return []Message{}, nil
	}
	start := uint32(1)
	if selected.Messages > uint32(limit) {
		start = selected.Messages - uint32(limit) + 1
	}
	set := new(imap.SeqSet)
	set.AddRange(start, selected.Messages)
	section := &imap.BodySectionName{Peek: true, Partial: []int{0, 128 << 10}}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, section.FetchItem()}
	channel := make(chan *imap.Message, limit)
	fetchErr := make(chan error, 1)
	go func() { fetchErr <- client.Fetch(set, items, channel) }()
	messages := make([]Message, 0, limit)
	for item := range channel {
		if item == nil || item.Envelope == nil {
			continue
		}
		preview := ""
		if body := item.GetBody(section); body != nil {
			preview = extractPreview(body)
		}
		from := ""
		if len(item.Envelope.From) > 0 {
			from = item.Envelope.From[0].Address()
			if item.Envelope.From[0].PersonalName != "" {
				from = item.Envelope.From[0].PersonalName + " <" + from + ">"
			}
		}
		messages = append(messages, Message{
			UID: item.Uid, From: boundedText(from, 500), Subject: boundedText(item.Envelope.Subject, 1000),
			Date: item.Envelope.Date.UTC(), Preview: preview, Unread: !containsFlag(item.Flags, imap.SeenFlag),
		})
	}
	if err := <-fetchErr; err != nil {
		return nil, errors.New("IMAP message fetch failed")
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].Date.After(messages[j].Date) })
	return messages, nil
}

func (s *Service) Send(message OutgoingMessage) error {
	cfg, password, err := s.credentials()
	if err != nil {
		return err
	}
	to, err := validateAddressList(message.To, 20)
	if err != nil || len(to) == 0 {
		return errors.New("recipient list is invalid")
	}
	cc, err := validateAddressList(message.CC, 20)
	if err != nil {
		return errors.New("CC list is invalid")
	}
	if hasHeaderBreak(message.Subject) {
		return errors.New("mail subject contains a forbidden header break")
	}
	message.Subject = boundedText(message.Subject, 500)
	message.Body = boundedText(message.Body, 200000)
	if message.Subject == "" || message.Body == "" {
		return errors.New("mail subject and body are required")
	}
	payload, err := buildMessage(cfg, to, cc, message.Subject, message.Body)
	if err != nil {
		return errors.New("mail message could not be encoded")
	}
	address := net.JoinHostPort(cfg.SMTPHost, fmt.Sprint(cfg.SMTPPort))
	client, err := openSMTP(address, cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Auth(smtp.PlainAuth("", cfg.Username, password, cfg.SMTPHost)); err != nil {
		return errors.New("SMTP authentication failed")
	}
	if err := client.Mail(cfg.Email); err != nil {
		return errors.New("SMTP sender rejected")
	}
	for _, recipient := range append(append([]string(nil), to...), cc...) {
		if err := client.Rcpt(recipient); err != nil {
			return errors.New("SMTP recipient rejected")
		}
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("SMTP DATA command rejected")
	}
	if _, err := writer.Write(payload); err != nil {
		writer.Close()
		return errors.New("SMTP message transfer failed")
	}
	if err := writer.Close(); err != nil {
		return errors.New("SMTP message finalization failed")
	}
	if err := client.Quit(); err != nil {
		return errors.New("SMTP server did not confirm completion")
	}
	return nil
}

func (s *Service) TestConnection() error {
	_, err := s.ListMessages(1)
	return err
}

func (s *Service) credentials() (Config, string, error) {
	cfg, err := s.LoadConfig()
	if err != nil || !cfg.Enabled {
		return Config{}, "", errors.New("mail account is not configured and enabled")
	}
	password, err := s.vault.GetToken(passwordSecretID)
	if err != nil || password == "" {
		return Config{}, "", errors.New("mail credential is unavailable")
	}
	return cfg, password, nil
}

func validateConfig(cfg Config) error {
	if !cfg.Enabled {
		return errors.New("mail configuration must be enabled")
	}
	address, err := mail.ParseAddress(strings.TrimSpace(cfg.Email))
	if err != nil || address.Address != strings.TrimSpace(cfg.Email) {
		return errors.New("mail address is invalid")
	}
	if len(cfg.DisplayName) > 120 || hasHeaderBreak(cfg.DisplayName) || len(cfg.Username) < 1 || len(cfg.Username) > 320 || hasHeaderBreak(cfg.Username) {
		return errors.New("mail identity is invalid")
	}
	if !validHost(cfg.IMAPHost) || !validHost(cfg.SMTPHost) {
		return errors.New("mail server host is invalid")
	}
	if cfg.IMAPPort < 1 || cfg.IMAPPort > 65535 || cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return errors.New("mail server port is invalid")
	}
	if cfg.SMTPSecurity != "tls" && cfg.SMTPSecurity != "starttls" {
		return errors.New("SMTP security must be tls or starttls")
	}
	return nil
}

func validHost(host string) bool {
	host = strings.TrimSpace(host)
	return len(host) <= 253 && hostPattern.MatchString(host) && !strings.Contains(host, "..")
}

func tlsConfig(host string) *tls.Config {
	return &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
}

func openSMTP(address string, cfg Config) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: 12 * time.Second, KeepAlive: 20 * time.Second}
	if cfg.SMTPSecurity == "tls" {
		connection, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig(cfg.SMTPHost))
		if err != nil {
			return nil, errors.New("secure SMTP connection failed")
		}
		_ = connection.SetDeadline(time.Now().Add(30 * time.Second))
		client, err := smtp.NewClient(connection, cfg.SMTPHost)
		if err != nil {
			connection.Close()
			return nil, errors.New("SMTP session initialization failed")
		}
		return client, nil
	}
	connection, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, errors.New("SMTP connection failed")
	}
	_ = connection.SetDeadline(time.Now().Add(30 * time.Second))
	client, err := smtp.NewClient(connection, cfg.SMTPHost)
	if err != nil {
		connection.Close()
		return nil, errors.New("SMTP session initialization failed")
	}
	if ok, _ := client.Extension("STARTTLS"); !ok {
		client.Close()
		return nil, errors.New("SMTP server does not offer STARTTLS")
	}
	if err := client.StartTLS(tlsConfig(cfg.SMTPHost)); err != nil {
		client.Close()
		return nil, errors.New("SMTP STARTTLS negotiation failed")
	}
	return client, nil
}

func buildMessage(cfg Config, to, cc []string, subject, body string) ([]byte, error) {
	var output bytes.Buffer
	header := messageMail.Header{}
	from := &mail.Address{Name: cfg.DisplayName, Address: cfg.Email}
	header.SetAddressList("From", []*mail.Address{from})
	header.SetAddressList("To", parsedAddresses(to))
	if len(cc) > 0 {
		header.SetAddressList("Cc", parsedAddresses(cc))
	}
	header.SetSubject(subject)
	header.SetDate(time.Now().UTC())
	header.GenerateMessageID()
	writer, err := messageMail.CreateWriter(&output, header)
	if err != nil {
		return nil, err
	}
	inline := messageMail.InlineHeader{}
	inline.Set("Content-Type", "text/plain; charset=utf-8")
	part, err := writer.CreateSingleInline(inline)
	if err != nil {
		writer.Close()
		return nil, err
	}
	if _, err := io.WriteString(part, body); err != nil {
		part.Close()
		writer.Close()
		return nil, err
	}
	if err := part.Close(); err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func extractPreview(reader io.Reader) string {
	message, err := messageMail.CreateReader(reader)
	if err != nil {
		return ""
	}
	defer message.Close()
	for {
		part, err := message.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			break
		}
		inline, ok := part.Header.(*messageMail.InlineHeader)
		if !ok {
			continue
		}
		contentType, _, _ := inline.ContentType()
		if contentType != "text/plain" {
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(part.Body, 8192))
		return boundedText(string(data), 1200)
	}
	return ""
}

func validateAddressList(values []string, max int) ([]string, error) {
	if len(values) > max {
		return nil, errors.New("recipient limit exceeded")
	}
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if hasHeaderBreak(value) {
			return nil, errors.New("invalid recipient")
		}
		parsed, err := mail.ParseAddress(strings.TrimSpace(value))
		if err != nil || parsed.Address == "" {
			return nil, errors.New("invalid recipient")
		}
		canonical := strings.ToLower(parsed.Address)
		if !seen[canonical] {
			seen[canonical] = true
			result = append(result, parsed.Address)
		}
	}
	return result, nil
}

func parsedAddresses(values []string) []*mail.Address {
	result := make([]*mail.Address, 0, len(values))
	for _, value := range values {
		result = append(result, &mail.Address{Address: value})
	}
	return result
}

func containsFlag(flags []string, target string) bool {
	for _, flag := range flags {
		if flag == target {
			return true
		}
	}
	return false
}

func hasHeaderBreak(value string) bool { return strings.ContainsAny(value, "\r\n") }

func boundedText(value string, limit int) string {
	value = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, value))
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}
