package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-aethel/mailbox"
	"go-aethel/security"
	"strings"
)

type MailListMessagesSkill struct{}

func (s *MailListMessagesSkill) Name() string { return "mail_list_messages" }
func (s *MailListMessagesSkill) Description() string {
	return "Liest bis zu 20 neueste Nachrichten aus dem vom Operator konfigurierten IMAP-Posteingang. Gibt Absender, Betreff, Datum, Ungelesen-Status und eine begrenzte Textvorschau zurück."
}
func (s *MailListMessagesSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *MailListMessagesSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{
		"limit": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 20},
	}, "additionalProperties": false}
}
func (s *MailListMessagesSkill) Execute(args json.RawMessage) (string, error) {
	var input struct {
		Limit int `json:"limit"`
	}
	if len(args) > 0 && string(args) != "null" && json.Unmarshal(args, &input) != nil {
		return "", errors.New("invalid mail list arguments")
	}
	if mailbox.SharedService == nil {
		return "", errors.New("mail service unavailable")
	}
	messages, err := mailbox.SharedService.ListMessages(input.Limit)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(map[string]interface{}{"messages": messages, "count": len(messages), "source": "operator-configured IMAP"})
	if err != nil {
		return "", errors.New("mail result encoding failed")
	}
	return string(encoded), nil
}

type MailSendMessageSkill struct{}

func (s *MailSendMessageSkill) Name() string { return "mail_send_message" }
func (s *MailSendMessageSkill) Description() string {
	return "Sendet genau eine E-Mail über das vom Operator konfigurierte TLS/STARTTLS-SMTP-Konto. Erfordert eine konkrete Empfängerliste, Betreff, vollständigen Text und eine einmalige Operatorfreigabe."
}
func (s *MailSendMessageSkill) RiskLevel() security.RiskLevel { return security.RiskCritical }
func (s *MailSendMessageSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{
		"to":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "minItems": 1, "maxItems": 20},
		"cc":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "maxItems": 20},
		"subject": map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 500},
		"body":    map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 200000},
	}, "required": []string{"to", "subject", "body"}, "additionalProperties": false}
}
func (s *MailSendMessageSkill) Execute(args json.RawMessage) (string, error) {
	var input mailbox.OutgoingMessage
	if json.Unmarshal(args, &input) != nil || len(input.To) == 0 || strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.Body) == "" {
		return "", errors.New("invalid outgoing mail arguments")
	}
	if mailbox.SharedService == nil {
		return "", errors.New("mail service unavailable")
	}
	if err := mailbox.SharedService.Send(input); err != nil {
		return "", err
	}
	return fmt.Sprintf("E-Mail wurde vom SMTP-Server für %d Empfänger bestätigt.", len(input.To)+len(input.CC)), nil
}
