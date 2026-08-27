package mailbox

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	messageMail "github.com/emersion/go-message/mail"
)

type SpamAssessment struct {
	Score   int      `json:"score"`
	Class   string   `json:"class"`
	Reasons []string `json:"reasons"`
}

type CalendarCandidate struct {
	Title      string    `json:"title"`
	DueAt      time.Time `json:"due_at"`
	Confidence float64   `json:"confidence"`
	Reason     string    `json:"reason"`
}

var (
	suspiciousSubjectPattern = regexp.MustCompile(`(?i)(dringend|urgent|konto gesperrt|account suspended|gewinn|lotterie|crypto.{0,12}(bonus|airdrop)|passwort.{0,12}(bestätigen|verify)|zahlung.{0,12}(sofort|heute))`)
	paymentPattern           = regexp.MustCompile(`(?i)(zahlungsaufforderung|rechnung|invoice|mahnung|fällig|faellig|zahlbar bis|payment due|deadline|frist)`)
	dateGermanPattern        = regexp.MustCompile(`\b(0?[1-9]|[12][0-9]|3[01])\.(0?[1-9]|1[0-2])\.(20[0-9]{2})\b`)
	dateISOPattern           = regexp.MustCompile(`\b(20[0-9]{2})-(0[1-9]|1[0-2])-([0-2][0-9]|3[01])\b`)
	urlPattern               = regexp.MustCompile(`(?i)https?://[^\s<>()]+`)
)

func (s *Service) GetMessage(folder string, uid uint32) (StoredMessage, error) {
	if s.storeErr != nil {
		return StoredMessage{}, errors.New("encrypted mail archive integrity check failed")
	}
	if !validMailboxName(folder) || uid == 0 {
		return StoredMessage{}, errors.New("mail message request is invalid")
	}
	cfg, password, err := s.credentials()
	if err != nil {
		return StoredMessage{}, err
	}
	client, err := dialIMAP(cfg, password)
	if err != nil {
		return StoredMessage{}, err
	}
	defer client.Logout()
	if _, err := client.Select(folder, true); err != nil {
		return StoredMessage{}, errors.New("IMAP folder selection failed")
	}
	set := new(imap.SeqSet)
	set.AddNum(uid)
	section := &imap.BodySectionName{Peek: true}
	channel := make(chan *imap.Message, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.UidFetch(set, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, section.FetchItem()}, channel)
	}()
	var item *imap.Message
	for candidate := range channel {
		if candidate != nil {
			item = candidate
		}
	}
	if fetchErr := <-errCh; fetchErr != nil || item == nil {
		return StoredMessage{}, errors.New("IMAP message fetch failed")
	}
	body := item.GetBody(section)
	if body == nil {
		return StoredMessage{}, errors.New("IMAP message body unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(body, 5<<20))
	if err != nil {
		return StoredMessage{}, errors.New("mail message could not be read")
	}
	message, err := parseStoredMessage(folder, item.Uid, item.Flags, raw)
	if err != nil {
		return StoredMessage{}, err
	}
	assessment := AssessSpam(message)
	message.AccountID = strings.ToLower(cfg.Email)
	message.SpamScore, message.SpamClass, message.SpamReasons = assessment.Score, assessment.Class, assessment.Reasons
	if err := s.store.UpsertMessages([]StoredMessage{message}); err != nil {
		return StoredMessage{}, errors.New("encrypted mail archive write failed")
	}
	return message, nil
}

func parseStoredMessage(folder string, uid uint32, flags []string, raw []byte) (StoredMessage, error) {
	reader, err := messageMail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return StoredMessage{}, errors.New("mail MIME structure is invalid")
	}
	defer reader.Close()
	from, _ := reader.Header.AddressList("From")
	to, _ := reader.Header.AddressList("To")
	cc, _ := reader.Header.AddressList("Cc")
	subject, _ := reader.Header.Subject()
	date, _ := reader.Header.Date()
	messageID, _ := reader.Header.MessageID()
	message := StoredMessage{
		Folder: folder, UID: uid, MessageID: boundedText(messageID, 500),
		From: boundedText(addressesText(from), 500), To: addressStrings(to), CC: addressStrings(cc),
		Subject: boundedText(subject, 1000), Date: date.UTC(), Unread: !containsFlag(flags, imap.SeenFlag),
		Headers: map[string]string{},
	}
	for _, key := range []string{"Authentication-Results", "Received-SPF", "DKIM-Signature", "Return-Path", "Reply-To", "List-Unsubscribe", "X-Spam-Status", "Content-Type"} {
		if value := boundedText(reader.Header.Get(key), 4000); value != "" {
			message.Headers[key] = value
		}
	}
	for {
		part, partErr := reader.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}
		if partErr != nil {
			return StoredMessage{}, errors.New("mail MIME part could not be decoded")
		}
		inline, ok := part.Header.(*messageMail.InlineHeader)
		if !ok {
			continue
		}
		contentType, _, _ := inline.ContentType()
		data, readErr := io.ReadAll(io.LimitReader(part.Body, 2<<20))
		if readErr != nil {
			continue
		}
		switch contentType {
		case "text/plain":
			if message.TextBody == "" {
				message.TextBody = boundedText(string(data), 200000)
			}
		case "text/html":
			if message.HTMLBody == "" {
				message.HTMLBody = boundedText(string(data), 500000)
			}
		}
	}
	if message.TextBody == "" {
		message.TextBody = "Kein Textteil verfügbar."
	}
	return message, nil
}

func AssessSpam(message StoredMessage) SpamAssessment {
	score := 0
	reasons := make([]string, 0, 8)
	auth := strings.ToLower(message.Headers["Authentication-Results"] + " " + message.Headers["Received-SPF"])
	if strings.Contains(auth, "spf=fail") || strings.Contains(auth, "spf fail") {
		score += 25
		reasons = append(reasons, "SPF-Prüfung fehlgeschlagen")
	}
	if strings.Contains(auth, "dkim=fail") {
		score += 25
		reasons = append(reasons, "DKIM-Prüfung fehlgeschlagen")
	}
	if strings.Contains(auth, "dmarc=fail") {
		score += 30
		reasons = append(reasons, "DMARC-Prüfung fehlgeschlagen")
	}
	if strings.Contains(strings.ToLower(message.Headers["X-Spam-Status"]), "yes") {
		score += 45
		reasons = append(reasons, "Mailserver markiert die Nachricht als Spam")
	}
	if suspiciousSubjectPattern.MatchString(message.Subject) {
		score += 18
		reasons = append(reasons, "Betreff enthält typisches Druck- oder Ködermuster")
	}
	links := urlPattern.FindAllString(message.TextBody+" "+message.HTMLBody, 12)
	if len(links) >= 5 {
		score += 12
		reasons = append(reasons, "Ungewöhnlich viele externe Links")
	}
	if len(message.TextBody) < 40 && len(links) > 0 {
		score += 10
		reasons = append(reasons, "Sehr wenig Text bei externem Link")
	}
	fromDomain := addressDomain(message.From)
	returnDomain := addressDomain(message.Headers["Return-Path"])
	if fromDomain != "" && returnDomain != "" && fromDomain != returnDomain {
		score += 12
		reasons = append(reasons, "Absender- und Return-Path-Domain unterscheiden sich")
	}
	if score > 100 {
		score = 100
	}
	class := "clean"
	if score >= 70 {
		class = "spam"
	} else if score >= 35 {
		class = "suspicious"
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "Keine auffälligen Header- oder Inhaltsmerkmale erkannt")
	}
	return SpamAssessment{Score: score, Class: class, Reasons: reasons}
}

func DetectCalendarCandidates(message StoredMessage) []CalendarCandidate {
	text := message.Subject + "\n" + message.TextBody
	if !paymentPattern.MatchString(text) {
		return []CalendarCandidate{}
	}
	results := make([]CalendarCandidate, 0, 2)
	for _, match := range dateGermanPattern.FindAllStringSubmatch(text, 3) {
		day, _ := strconv.Atoi(match[1])
		month, _ := strconv.Atoi(match[2])
		year, _ := strconv.Atoi(match[3])
		if due := validCalendarDate(year, month, day); !due.IsZero() {
			results = append(results, CalendarCandidate{Title: calendarTitle(message), DueAt: due, Confidence: .82, Reason: "Zahlungs-/Fristkontext mit deutschem Datum"})
		}
	}
	for _, match := range dateISOPattern.FindAllStringSubmatch(text, 3) {
		year, _ := strconv.Atoi(match[1])
		month, _ := strconv.Atoi(match[2])
		day, _ := strconv.Atoi(match[3])
		if due := validCalendarDate(year, month, day); !due.IsZero() {
			results = append(results, CalendarCandidate{Title: calendarTitle(message), DueAt: due, Confidence: .82, Reason: "Zahlungs-/Fristkontext mit ISO-Datum"})
		}
	}
	return results
}

func validCalendarDate(year, month, day int) time.Time {
	value := time.Date(year, time.Month(month), day, 9, 0, 0, 0, time.Local)
	if value.Year() != year || int(value.Month()) != month || value.Day() != day {
		return time.Time{}
	}
	return value
}

func calendarTitle(message StoredMessage) string {
	return boundedText("E-Mail-Frist: "+message.Subject, 180)
}
func addressesText(values []*mail.Address) string {
	if len(values) == 0 {
		return ""
	}
	return values[0].String()
}
func addressStrings(values []*mail.Address) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Address)
	}
	return result
}
func addressDomain(value string) string {
	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.ToLower(parsed.Address), "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func (s *Service) Store() *SecureMailStore { return s.store }
func (s *Service) StoreError() error       { return s.storeErr }
func MessageReference(folder string, uid uint32) string {
	return fmt.Sprintf("mail://%s/%d", folder, uid)
}
