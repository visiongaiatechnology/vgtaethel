package osint

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

const maxArticleResponseBytes = 4 << 20
const maxArticleTextRunes = 120000

type ReadableArticle struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	URL   string `json:"url"`
}

func FetchReadableArticle(ctx context.Context, rawURL string) (ReadableArticle, error) {
	if err := validatePublicCollectorURL(rawURL); err != nil {
		return ReadableArticle{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ReadableArticle{}, errors.New("article request invalid")
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-READER/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := newSafeCollectorHTTPClient().Do(req)
	if err != nil {
		return ReadableArticle{}, errors.New("article source unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ReadableArticle{}, errors.New("article source rejected the reader request")
	}
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
		return ReadableArticle{}, errors.New("article source did not return HTML")
	}
	limited := io.LimitReader(resp.Body, maxArticleResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil || len(body) > maxArticleResponseBytes {
		return ReadableArticle{}, errors.New("article response exceeds the reader limit")
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return ReadableArticle{}, errors.New("article document could not be parsed")
	}
	title, text := extractReadableHTML(doc)
	if len([]rune(text)) < 80 {
		return ReadableArticle{}, errors.New("article source exposes no readable article text")
	}
	return ReadableArticle{Title: title, Text: text, URL: resp.Request.URL.String()}, nil
}

func extractReadableHTML(root *html.Node) (string, string) {
	var title string
	var sections []string
	var walk func(*html.Node, bool)
	walk = func(node *html.Node, blocked bool) {
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			if tag == "script" || tag == "style" || tag == "noscript" || tag == "svg" || tag == "nav" || tag == "footer" || tag == "form" {
				blocked = true
			}
			if !blocked && (tag == "title" || tag == "h1" || tag == "h2" || tag == "p" || tag == "li" || tag == "blockquote") {
				value := normalizeArticleText(nodeText(node))
				if tag == "title" && title == "" {
					title = value
				} else if len([]rune(value)) >= 25 {
					sections = append(sections, value)
				}
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, blocked)
		}
	}
	walk(root, false)
	text := strings.Join(sections, "\n\n")
	runes := []rune(text)
	if len(runes) > maxArticleTextRunes {
		text = string(runes[:maxArticleTextRunes])
	}
	return title, text
}

func nodeText(node *html.Node) string {
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			out.WriteString(current.Data)
			out.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return out.String()
}

func normalizeArticleText(value string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.Join(strings.Fields(value), " ")))
}
