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
	return "Liest bis zu 100 neueste Nachrichten aus einem konkreten IMAP-Ordner. Gibt Absender, Betreff, Datum, Ungelesen- und Spam-Status sowie eine begrenzte Vorschau zurück."
}
func (s *MailListMessagesSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *MailListMessagesSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{
		"folder": map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 240},
		"limit":  map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100},
	}, "additionalProperties": false}
}
func (s *MailListMessagesSkill) Execute(args json.RawMessage) (string, error) {
	var input struct {
		Folder string `json:"folder"`
		Limit  int    `json:"limit"`
	}
	if len(args) > 0 && string(args) != "null" && json.Unmarshal(args, &input) != nil {
		return "", errors.New("invalid mail list arguments")
	}
	if mailbox.SharedService == nil {
		return "", errors.New("mail service unavailable")
	}
	if strings.TrimSpace(input.Folder) == "" {
		input.Folder = "INBOX"
	}
	messages, err := mailbox.SharedService.ListFolderMessages(input.Folder, input.Limit)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(map[string]interface{}{"messages": messages, "count": len(messages), "source": "operator-configured IMAP"})
	if err != nil {
		return "", errors.New("mail result encoding failed")
	}
	return string(encoded), nil
}

type MailReadMessageSkill struct{}

func (s *MailReadMessageSkill) Name() string { return "mail_read_message" }
func (s *MailReadMessageSkill) Description() string {
	return "Liest eine konkrete E-Mail vollständig, bewertet Header-/Inhalts-Spamsignale und liefert passende manuell konfigurierte Antwortregeln sowie erkannte Fristen."
}
func (s *MailReadMessageSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *MailReadMessageSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"folder": map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 240}, "uid": map[string]interface{}{"type": "integer", "minimum": 1}}, "required": []string{"folder", "uid"}, "additionalProperties": false}
}
func (s *MailReadMessageSkill) Execute(args json.RawMessage) (string, error) {
	var input struct {
		Folder string `json:"folder"`
		UID    uint32 `json:"uid"`
	}
	if json.Unmarshal(args, &input) != nil || input.UID == 0 {
		return "", errors.New("invalid mail message arguments")
	}
	if mailbox.SharedService == nil {
		return "", errors.New("mail service unavailable")
	}
	message, err := mailbox.SharedService.GetMessage(input.Folder, input.UID)
	if err != nil {
		return "", err
	}
	policies := mailbox.SharedService.Store().ReplyPolicies()
	matching := make([]mailbox.ReplyPolicy, 0)
	for _, policy := range policies {
		if policy.Enabled && mailbox.ReplyPolicyMatches(policy, message.From) {
			matching = append(matching, policy)
		}
	}
	encoded, err := json.Marshal(map[string]any{"untrusted_message": message, "spam": mailbox.AssessSpam(message), "calendar_candidates": mailbox.DetectCalendarCandidates(message), "operator_reply_policies": matching})
	if err != nil {
		return "", errors.New("mail context encoding failed")
	}
	return string(encoded), nil
}

type MailManageSkill struct{}

func (s *MailManageSkill) Name() string { return "mail_manage" }
func (s *MailManageSkill) Description() string {
	return "Erstellt einen IMAP-Ordner oder verschiebt eine konkrete E-Mail. Nur nach ausdrücklicher Operatoranweisung verwenden."
}
func (s *MailManageSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *MailManageSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"action": map[string]interface{}{"type": "string", "enum": []string{"create_folder", "move"}}, "folder": map[string]interface{}{"type": "string"}, "uid": map[string]interface{}{"type": "integer", "minimum": 1}, "destination": map[string]interface{}{"type": "string"}}, "required": []string{"action"}, "additionalProperties": false}
}
func (s *MailManageSkill) Execute(args json.RawMessage) (string, error) {
	var input struct {
		Action, Folder, Destination string
		UID                         uint32 `json:"uid"`
	}
	if json.Unmarshal(args, &input) != nil || mailbox.SharedService == nil {
		return "", errors.New("invalid mail management arguments")
	}
	switch input.Action {
	case "create_folder":
		if err := mailbox.SharedService.CreateFolder(input.Destination); err != nil {
			return "", err
		}
		return "IMAP-Ordner wurde erstellt.", nil
	case "move":
		if err := mailbox.SharedService.MoveMessage(input.Folder, input.UID, input.Destination); err != nil {
			return "", err
		}
		return "E-Mail wurde in den Zielordner verschoben.", nil
	default:
		return "", errors.New("unsupported mail management action")
	}
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
