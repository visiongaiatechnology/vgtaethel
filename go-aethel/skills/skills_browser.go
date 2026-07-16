package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-aethel/security"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	browserExecutionMu sync.Mutex
	browserStateMu     sync.RWMutex
	lastBrowserURL     = "https://www.google.com"
)

func GetLastBrowserURL() string {
	browserStateMu.RLock()
	defer browserStateMu.RUnlock()
	return lastBrowserURL
}

func setLastBrowserURL(target string) {
	browserStateMu.Lock()
	lastBrowserURL = target
	browserStateMu.Unlock()
}

type WebBrowserSkill struct{}

type BrowserArgs struct {
	Action      string `json:"action"`
	URL         string `json:"url,omitempty"`
	SearchQuery string `json:"search_query,omitempty"`
}

func (s *WebBrowserSkill) Name() string { return "web_browser" }
func (s *WebBrowserSkill) Description() string {
	return "Öffnet einen isolierten Headless-Browser für öffentliche HTTPS-Seiten, erstellt einen Screenshot und extrahiert Text. Das Tool ist statuslos und unterstützt kein Scrollen, Klicken oder Tippen."
}
func (s *WebBrowserSkill) RiskLevel() security.RiskLevel { return security.RiskCritical }

func (s *WebBrowserSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action":       map[string]interface{}{"type": "string", "enum": []string{"navigate", "search"}, "description": "Öffentliche HTTPS-URL laden oder per Google suchen. Kein Scrollen, Klicken oder Tippen."},
			"url":          map[string]interface{}{"type": "string", "description": "Öffentliche HTTPS-URL für navigate."},
			"search_query": map[string]interface{}{"type": "string", "description": "Suchbegriff für search."},
		},
		"required": []string{"action"},
	}
}

func findChromePath() string {
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func cleanHTMLText(document string) (string, string) {
	title := "Unbekannter Titel"
	titlePattern := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	if matches := titlePattern.FindStringSubmatch(document); len(matches) > 1 {
		title = strings.TrimSpace(matches[1])
	}

	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`),
		regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`),
		regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`),
	} {
		document = pattern.ReplaceAllString(document, " ")
	}
	document = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(document, " ")
	document = regexp.MustCompile(`\s+`).ReplaceAllString(document, " ")
	document = strings.TrimSpace(document)
	if len(document) > 2000 {
		document = document[:2000] + "... [Text gekürzt]"
	}
	return title, document
}

func (s *WebBrowserSkill) Execute(args json.RawMessage) (string, error) {
	browserExecutionMu.Lock()
	defer browserExecutionMu.Unlock()

	var input BrowserArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if input.Action != "navigate" && input.Action != "search" {
		return "", errors.New("nicht erlaubte Browser-Aktion")
	}

	targetURL := strings.TrimSpace(input.URL)
	if input.Action == "search" {
		query := strings.TrimSpace(input.SearchQuery)
		if query == "" {
			return "", errors.New("keine Suchanfrage übergeben")
		}
		if len([]rune(query)) > 1000 {
			return "", errors.New("Suchanfrage überschreitet die Größenbegrenzung")
		}
		targetURL = "https://www.google.com/search?q=" + url.QueryEscape(query)
	}
	if targetURL == "" {
		return "", errors.New("keine Ziel-URL oder Suchanfrage übergeben")
	}
	if len(targetURL) > 4096 {
		return "", errors.New("Browser-URL überschreitet die Größenbegrenzung")
	}
	if !strings.Contains(targetURL, "://") {
		targetURL = "https://" + targetURL
	}

	resolveCtx, resolveCancel := context.WithTimeout(context.Background(), 5*time.Second)
	destination, err := security.ResolvePublicHTTPSDestination(resolveCtx, targetURL)
	resolveCancel()
	if err != nil {
		return "", fmt.Errorf("Browser-Ziel abgelehnt: %w", err)
	}
	targetURL = destination.URL
	resolverRules := security.ChromiumHostResolverRules(destination)

	chromePath := findChromePath()
	if chromePath == "" {
		security.LogKernelActivity("BROWSER_FAILED", targetURL, "ERROR")
		return "", errors.New("kein vertrauenswürdiger Chrome- oder Edge-Installationspfad gefunden")
	}
	security.LogKernelActivity("BROWSER_START", targetURL, "PENDING")

	screenshotPath, err := filepath.Abs("./vgt_workspace/browser_screenshot.png")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(screenshotPath), 0700); err != nil {
		return "", err
	}
	if err := os.Remove(screenshotPath); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	tempProfile, err := os.MkdirTemp("", "aethel_chrome_profile_*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempProfile)

	commonArgs := []string{
		"--headless=new",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-default-apps",
		"--no-first-run",
		"--no-proxy-server",
		"--incognito",
		"--user-data-dir=" + tempProfile,
		"--host-resolver-rules=" + resolverRules,
	}

	screenshotCtx, screenshotCancel := context.WithTimeout(context.Background(), 15*time.Second)
	screenshotArgs := append(append([]string{}, commonArgs...),
		"--window-size=1280,720",
		"--screenshot="+screenshotPath,
		targetURL,
	)
	screenshotErr := exec.CommandContext(screenshotCtx, chromePath, screenshotArgs...).Run()
	screenshotCancel()
	if _, statErr := os.Stat(screenshotPath); statErr != nil {
		if screenshotErr != nil {
			return "", errors.New("Webseite konnte innerhalb des Zeitlimits nicht sicher geladen werden")
		}
		return "", errors.New("Browser-Screenshot wurde nicht erzeugt")
	}
	if err := os.Chmod(screenshotPath, 0600); err != nil {
		return "", err
	}

	htmlCtx, htmlCancel := context.WithTimeout(context.Background(), 15*time.Second)
	htmlArgs := append(append([]string{}, commonArgs...), "--dump-html", targetURL)
	cmdHTML := exec.CommandContext(htmlCtx, chromePath, htmlArgs...)
	outHTML := newBoundedBuffer(4 * 1024 * 1024)
	cmdHTML.Stdout = &outHTML
	htmlErr := cmdHTML.Run()
	htmlCancel()
	if htmlErr != nil && outHTML.Len() == 0 {
		return "", errors.New("Webseiteninhalt konnte innerhalb des Zeitlimits nicht extrahiert werden")
	}

	title, content := cleanHTMLText(outHTML.String())
	setLastBrowserURL(targetURL)
	security.LogKernelActivity("BROWSER", targetURL, "SUCCESS")
	return fmt.Sprintf(
		"Aktion: %s\nURL: %s\nTitel: %s\nUNTRUSTED_WEB_CONTENT_BEGIN\n%s\nUNTRUSTED_WEB_CONTENT_END",
		input.Action,
		targetURL,
		title,
		content,
	), nil
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	remaining int
}

func newBoundedBuffer(limit int) boundedBuffer {
	return boundedBuffer{remaining: limit}
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	originalLength := len(data)
	if b.remaining <= 0 {
		return originalLength, nil
	}
	if len(data) > b.remaining {
		data = data[:b.remaining]
	}
	_, _ = b.buffer.Write(data)
	b.remaining -= len(data)
	return originalLength, nil
}

func (b *boundedBuffer) Len() int       { return b.buffer.Len() }
func (b *boundedBuffer) String() string { return b.buffer.String() }
