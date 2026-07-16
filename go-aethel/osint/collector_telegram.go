package osint

import (
	"context"
	"errors"
	"fmt"
	"go-aethel/intelligence"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
)

const CollectorTypeTelegram = "telegram"

var telegramChannelPattern = regexp.MustCompile(`^[A-Za-z0-9_]{5,32}$`)

type TelegramCollector struct {
	cfg    OSINTCollectorConfig
	client *http.Client
}

func NewTelegramCollector(cfg OSINTCollectorConfig) *TelegramCollector {
	return &TelegramCollector{cfg: cfg, client: newSafeCollectorHTTPClient()}
}

func (c *TelegramCollector) Name() string                     { return c.cfg.Name }
func (c *TelegramCollector) Domain() intelligence.OSINTDomain { return c.cfg.Domain }

func normalizeTelegramPublicURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("Telegram source must be an uncredentialed HTTPS public preview URL")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return "", errors.New("Telegram source port is not permitted")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "t.me" && host != "www.t.me" {
		return "", errors.New("Telegram source host must be t.me")
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 2 || parts[0] != "s" {
		return "", errors.New("Telegram source must use https://t.me/s/channel")
	}
	channel, err := url.PathUnescape(parts[1])
	if err != nil || !telegramChannelPattern.MatchString(channel) {
		return "", errors.New("Telegram channel name is invalid")
	}
	return "https://t.me/s/" + channel, nil
}

func (c *TelegramCollector) Collect(ctx context.Context) ([]intelligence.OSINTEvent, error) {
	canonical, err := normalizeTelegramPublicURL(c.cfg.URL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonical, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-OSINT/1.0 (Telegram public preview collector)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Telegram public preview: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Telegram public preview returned status %d", resp.StatusCode)
	}
	if finalURL, finalErr := normalizeTelegramPublicURL(resp.Request.URL.String()); finalErr != nil || finalURL == "" {
		return nil, errors.New("Telegram preview redirected outside the permitted public channel path")
	}
	const maxTelegramBytes = 3 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramBytes+1))
	if err != nil || len(body) > maxTelegramBytes {
		return nil, errors.New("Telegram public preview exceeds the response boundary")
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.New("Telegram public preview HTML is invalid")
	}
	return c.parseDocument(doc), nil
}

func (c *TelegramCollector) parseDocument(doc *html.Node) []intelligence.OSINTEvent {
	events := make([]intelligence.OSINTEvent, 0, 20)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if len(events) >= 20 {
			return
		}
		if node.Type == html.ElementNode && htmlClassContains(node, "tgme_widget_message") {
			if event, ok := c.eventFromMessageNode(node); ok {
				events = append(events, event)
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return events
}

func (c *TelegramCollector) eventFromMessageNode(node *html.Node) (intelligence.OSINTEvent, bool) {
	postID := htmlAttribute(node, "data-post")
	textNode := findHTMLNodeByClass(node, "tgme_widget_message_text")
	timeNode := findHTMLElement(node, "time")
	if postID == "" || textNode == nil || timeNode == nil {
		return intelligence.OSINTEvent{}, false
	}
	observed, err := time.Parse(time.RFC3339, htmlAttribute(timeNode, "datetime"))
	if err != nil || observed.IsZero() || observed.After(time.Now().UTC().Add(10*time.Minute)) {
		return intelligence.OSINTEvent{}, false
	}
	message := normalizeTelegramText(nodeText(textNode))
	if message == "" {
		return intelligence.OSINTEvent{}, false
	}
	linkNode := findHTMLNodeByClass(node, "tgme_widget_message_date")
	messageURL := "https://t.me/" + strings.TrimPrefix(postID, "@")
	if linkNode != nil {
		if candidate := htmlAttribute(linkNode, "href"); strings.HasPrefix(candidate, "https://t.me/") {
			messageURL = candidate
		}
	}
	title := message
	if runes := []rune(title); len(runes) > 180 {
		title = string(runes[:180]) + "…"
	}
	event := intelligence.OSINTEvent{
		ID: hashID("telegram:" + postID), Title: title, Summary: message,
		Source: c.cfg.Name, SourceURL: c.cfg.URL, Domain: c.cfg.Domain,
		Timestamp: observed.UTC(), Confidence: 0.65, Status: "raw", URL: messageURL,
	}
	enrichEventGeo(&event)
	return event, true
}

func htmlClassContains(node *html.Node, target string) bool {
	for _, className := range strings.Fields(htmlAttribute(node, "class")) {
		if className == target {
			return true
		}
	}
	return false
}

func htmlAttribute(node *html.Node, name string) string {
	if node == nil {
		return ""
	}
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return strings.TrimSpace(attribute.Val)
		}
	}
	return ""
}

func findHTMLNodeByClass(root *html.Node, className string) *html.Node {
	if root == nil {
		return nil
	}
	if root.Type == html.ElementNode && htmlClassContains(root, className) {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLNodeByClass(child, className); found != nil {
			return found
		}
	}
	return nil
}

func findHTMLElement(root *html.Node, element string) *html.Node {
	if root == nil {
		return nil
	}
	if root.Type == html.ElementNode && root.Data == element {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLElement(child, element); found != nil {
			return found
		}
	}
	return nil
}

func normalizeTelegramText(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 4000 {
		return string(runes[:4000])
	}
	return string(runes)
}
