package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// --- 5. SKILL: WEB BROWSER ---

type WebBrowserSkill struct{}

type BrowserArgs struct {
	Action      string `json:"action"` // "navigate" or "search"
	URL         string `json:"url,omitempty"`
	SearchQuery string `json:"search_query,omitempty"`
}

func (s *WebBrowserSkill) Name() string { return "web_browser" }
func (s *WebBrowserSkill) Description() string {
	return "Öffnet einen lokalen Browser um Webseiten zu laden, Screenshots zu machen und Inhalte zu analysieren."
}
func (s *WebBrowserSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *WebBrowserSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action":       map[string]interface{}{"type": "string", "enum": []string{"navigate", "search"}, "description": "Aktion: navigieren oder suchen"},
			"url":          map[string]interface{}{"type": "string", "description": "Die URL bei 'navigate' (z.B. 'https://wikipedia.org')"},
			"search_query": map[string]interface{}{"type": "string", "description": "Suchbegriff bei 'search'"},
		},
		"required": []string{"action"},
	}
}

func findChromePath() string {
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`chrome`,
		`msedge`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		if !strings.Contains(p, `\`) {
			if _, err := exec.LookPath(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func cleanHTMLText(html string) (string, string) {
	// 1. Title extraction
	title := "Unbekannter Titel"
	reTitle := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	if matches := reTitle.FindStringSubmatch(html); len(matches) > 1 {
		title = strings.TrimSpace(matches[1])
	}

	// 2. Strip scripts and styles
	reScript := regexp.MustCompile(`(?s)<script.*?>.*?</script>`)
	html = reScript.ReplaceAllString(html, " ")
	reStyle := regexp.MustCompile(`(?s)<style.*?>.*?</style>`)
	html = reStyle.ReplaceAllString(html, " ")
	reNav := regexp.MustCompile(`(?s)<nav.*?>.*?</nav>`)
	html = reNav.ReplaceAllString(html, " ")
	reFooter := regexp.MustCompile(`(?s)<footer.*?>.*?</footer>`)
	html = reFooter.ReplaceAllString(html, " ")

	// 3. Strip tags
	reTags := regexp.MustCompile(`<[^>]*>`)
	text := reTags.ReplaceAllString(html, " ")

	// 4. Condense spaces
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Limit to ~2000 characters
	if len(text) > 2000 {
		text = text[:2000] + "... [Text gekürzt]"
	}

	return title, text
}

func (s *WebBrowserSkill) Execute(args json.RawMessage) (string, error) {
	var input BrowserArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	targetURL := input.URL
	if input.Action == "search" {
		query := strings.TrimSpace(input.SearchQuery)
		if query == "" {
			return "", errors.New("keine Suchanfrage uebergeben")
		}
		targetURL = "https://www.google.com/search?q=" + url.QueryEscape(query)
	}

	if targetURL == "" {
		return "", errors.New("keine Ziel-URL oder Suchanfrage übergeben")
	}

	// Verify standard URL schemes
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}
	parsedURL, err := url.Parse(targetURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", errors.New("ungueltige oder nicht erlaubte Browser-URL")
	}
	targetURL = parsedURL.String()

	chromePath := findChromePath()
	if chromePath == "" {
		LogKernelActivity("BROWSER_FAILED", targetURL, "ERROR")
		return "", errors.New("kein installierter Browser (Google Chrome oder Microsoft Edge) auf dem Windows-Host gefunden")
	}
	LogKernelActivity("BROWSER_START", targetURL, "PENDING")

	screenshotPath, err := filepath.Abs("./vgt_workspace/browser_screenshot.png")
	if err != nil {
		return "", err
	}
	_ = os.MkdirAll(filepath.Dir(screenshotPath), 0700)
	_ = os.Remove(screenshotPath) // Delete old screenshot

	tempProfile, err := os.MkdirTemp("", "aethel_chrome_profile_*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempProfile)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Capture Screenshot
	cmdScreenshot := exec.CommandContext(ctx, chromePath,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--incognito",
		"--user-data-dir="+tempProfile,
		"--window-size=1280,720",
		"--screenshot="+screenshotPath,
		targetURL,
	)
	_ = cmdScreenshot.Run() // Capture screenshot (ignore error, we check file next)

	// Verify screenshot exists
	if _, err := os.Stat(screenshotPath); err != nil {
		return "", fmt.Errorf("Fehler beim Laden der Webseite (Screenshot konnte nicht generiert werden / Timeout): %v", err)
	}

	// Step 2: Dump HTML
	cmdHTML := exec.CommandContext(ctx, chromePath,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--incognito",
		"--user-data-dir="+tempProfile,
		"--dump-html",
		targetURL,
	)
	var outHTML bytes.Buffer
	cmdHTML.Stdout = &outHTML
	_ = cmdHTML.Run()

	title, content := cleanHTMLText(outHTML.String())

	LogKernelActivity("BROWSER", targetURL, "SUCCESS")
	result := fmt.Sprintf("Aktion: %s\nURL: %s\nTitel: %s\nExtrahierter Text-Inhalt:\n%s", input.Action, targetURL, title, content)
	return result, nil
}
