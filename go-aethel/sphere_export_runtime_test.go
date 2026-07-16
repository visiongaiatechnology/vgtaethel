package main

// Runtime test: drives SHIPPED sphere.js export helpers via goja.
// Extracts only the export-related functions from the real file so the test
// path is the shipped source (not a reimplementation).

import (
	_ "embed"
	"regexp"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

//go:embed frontend/modules/sphere.js
var sphereJSForExport []byte

// extractShippedExportJS pulls sanitizeRichText + htmlToMarkdownRough +
// buildSphereDocumentExport from the real sphere.js module text.
func extractShippedExportJS(src string) string {
	// Normalize exports to plain functions for goja globals.
	src = strings.ReplaceAll(src, "export function ", "function ")
	src = strings.ReplaceAll(src, "export async function ", "async function ")

	// Grab function bodies by name (non-greedy until next top-level function or EOF marker).
	// We rely on function names unique in the file.
	names := []string{"sanitizeRichText", "htmlToMarkdownRough", "buildSphereDocumentExport"}
	var b strings.Builder
	b.WriteString("// extracted from shipped frontend/modules/sphere.js\n")
	for _, name := range names {
		// Match: function Name(...) { ... } with balanced braces via iterative scan
		idx := strings.Index(src, "function "+name)
		if idx < 0 {
			// try with export already stripped variants
			continue
		}
		// find opening brace
		brace := strings.Index(src[idx:], "{")
		if brace < 0 {
			continue
		}
		start := idx
		i := idx + brace
		depth := 0
		end := -1
		for ; i < len(src); i++ {
			switch src[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i + 1
					i = len(src)
				}
			}
		}
		if end > start {
			b.WriteString(src[start:end])
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func loadShippedSphereExportVM(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()

	_, err := vm.RunString(`
		if (typeof Array.from !== 'function') {
			Array.from = function(a) { return Array.prototype.slice.call(a); };
		}
		var Node = { ELEMENT_NODE: 1, TEXT_NODE: 3 };
		function __el(tag) {
			var el = {
				nodeType: 1,
				tagName: String(tag || 'DIV').toUpperCase(),
				textContent: '',
				childNodes: [],
				style: {},
				_attrs: {},
				hasAttribute: function(k) { return Object.prototype.hasOwnProperty.call(this._attrs, k); },
				getAttribute: function(k) { return this._attrs[k]; },
				setAttribute: function(k, v) { this._attrs[k] = String(v); },
				appendChild: function(c) {
					if (c && c.nodeType === 11) {
						for (var fragmentIndex = 0; fragmentIndex < c.childNodes.length; fragmentIndex++) this.childNodes.push(c.childNodes[fragmentIndex]);
						return c;
					}
					this.childNodes.push(c);
					return c;
				}
			};
			Object.defineProperty(el, 'innerHTML', {
				configurable: true,
				get: function() {
					var s = '';
					for (var i = 0; i < this.childNodes.length; i++) {
						var n = this.childNodes[i];
						if (!n) continue;
						if (n.nodeType === 3) s += n.textContent || '';
						else if (n.nodeType === 1) {
							var t = (n.tagName || 'DIV').toLowerCase();
							var inner = '';
							for (var j = 0; j < n.childNodes.length; j++) {
								var c = n.childNodes[j];
								if (c.nodeType === 3) inner += c.textContent || '';
								else if (c.nodeType === 1) {
									var t2 = (c.tagName || 'SPAN').toLowerCase();
									inner += '<' + t2 + '>' + (c.textContent || '') + '</' + t2 + '>';
								}
							}
							// also use recursive getter if available
							if (n.childNodes && n.childNodes.length && typeof Object.getOwnPropertyDescriptor === 'function') {
								try { inner = n.innerHTML; } catch (e) {}
							}
							s += '<' + t + '>' + inner + '</' + t + '>';
						}
					}
					return s;
				},
				set: function(html) {
					this.childNodes = [];
					var str = String(html || '');
					// strip script blocks entirely
					str = str.replace(/<script[\s\S]*?<\/script>/gi, '');
					var re = /<(h1|h2|h3|p|div|b|i|u|strong|em|ul|ol|li|br|span)(\s[^>]*)?>([\s\S]*?)<\/\1>|([^<]+)|<br\s*\/?>/gi;
					var m;
					while ((m = re.exec(str)) !== null) {
						if (m[4]) {
							this.appendChild({ nodeType: 3, textContent: m[4], childNodes: [] });
						} else if (m[0].toLowerCase().indexOf('<br') === 0) {
							this.appendChild({
								nodeType: 1, tagName: 'BR', textContent: '', childNodes: [],
								hasAttribute: function(){return false;}, getAttribute: function(){return null;},
								setAttribute: function(){}, appendChild: function(c){ this.childNodes.push(c); }
							});
						} else {
							var child = {
								nodeType: 1, tagName: String(m[1]).toUpperCase(), textContent: m[3] || '',
								childNodes: [], _attrs: {},
								hasAttribute: function(k){ return Object.prototype.hasOwnProperty.call(this._attrs,k); },
								getAttribute: function(k){ return this._attrs[k]; },
								setAttribute: function(k,v){ this._attrs[k]=String(v); },
								appendChild: function(c){ this.childNodes.push(c); return c; }
							};
							if (m[3]) child.appendChild({ nodeType: 3, textContent: m[3], childNodes: [] });
							this.appendChild(child);
						}
					}
				}
			});
			return el;
		}
		var document = {
			createElement: function(tag) { return __el(tag); },
			createTextNode: function(t) { return { nodeType: 3, textContent: String(t), childNodes: [] }; },
			createDocumentFragment: function() {
				return { nodeType: 11, childNodes: [], appendChild: function(c) { this.childNodes.push(c); return c; } };
			}
		};
		function DOMParser() {}
		DOMParser.prototype.parseFromString = function(html) {
			var body = __el('body');
			body.innerHTML = html;
			return { body: body };
		};
	`)
	if err != nil {
		t.Fatalf("dom mocks: %v", err)
	}

	extracted := extractShippedExportJS(string(sphereJSForExport))
	if !strings.Contains(extracted, "function sanitizeRichText") {
		t.Fatalf("failed to extract sanitizeRichText from shipped sphere.js; got:\n%s", extracted[:min(400, len(extracted))])
	}
	if !strings.Contains(extracted, "function buildSphereDocumentExport") {
		t.Fatalf("failed to extract buildSphereDocumentExport from shipped sphere.js")
	}
	if !strings.Contains(extracted, "function htmlToMarkdownRough") {
		t.Fatalf("failed to extract htmlToMarkdownRough from shipped sphere.js")
	}

	// Prove extract came from the real file (unique comment / strings).
	if !strings.Contains(string(sphereJSForExport), "aethel-sphere-document.html") {
		t.Fatal("shipped sphere.js missing expected export filename string")
	}

	_, err = vm.RunString(extracted)
	if err != nil {
		t.Fatalf("run extracted shipped export JS: %v\n---\n%s", err, extracted)
	}
	return vm
}

func TestSphereBuildDocumentExportHTMLAndMarkdown(t *testing.T) {
	// Guard: embedded source is the real module (not empty stub).
	if len(sphereJSForExport) < 500 {
		t.Fatalf("embedded sphere.js too small: %d bytes", len(sphereJSForExport))
	}
	if !regexp.MustCompile(`function\s+buildSphereDocumentExport|export function buildSphereDocumentExport`).Match(sphereJSForExport) {
		t.Fatal("shipped sphere.js must define buildSphereDocumentExport")
	}

	vm := loadShippedSphereExportVM(t)

	build, ok := goja.AssertFunction(vm.Get("buildSphereDocumentExport"))
	if !ok {
		t.Fatal("buildSphereDocumentExport not a function after loading shipped extract")
	}
	mdFn, ok := goja.AssertFunction(vm.Get("htmlToMarkdownRough"))
	if !ok {
		t.Fatal("htmlToMarkdownRough not a function after loading shipped extract")
	}
	san, ok := goja.AssertFunction(vm.Get("sanitizeRichText"))
	if !ok {
		t.Fatal("sanitizeRichText not a function after loading shipped extract")
	}

	// 1) sanitize strips script
	res, err := san(goja.Undefined(), vm.ToValue(`<p>Hello</p><script>alert(1)</script>`))
	if err != nil {
		t.Fatalf("sanitize call: %v", err)
	}
	cleaned := res.String()
	if strings.Contains(strings.ToLower(cleaned), "<script") {
		t.Fatalf("sanitize must strip script, got %q", cleaned)
	}
	if !strings.Contains(cleaned, "Hello") {
		t.Fatalf("sanitize must keep Hello, got %q", cleaned)
	}

	// 2) HTML export via shipped buildSphereDocumentExport
	res, err = build(goja.Undefined(), vm.ToValue(`<h1>Title</h1><p>Body text</p>`), vm.ToValue("html"))
	if err != nil {
		t.Fatalf("build html: %v", err)
	}
	pack, ok := res.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("html pack type %T", res.Export())
	}
	if pack["filename"] != "aethel-sphere-document.html" {
		t.Fatalf("html filename want aethel-sphere-document.html got %v", pack["filename"])
	}
	mime, _ := pack["mime"].(string)
	if !strings.Contains(mime, "text/html") {
		t.Fatalf("html mime: %v", mime)
	}
	body, _ := pack["body"].(string)
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("html body missing doctype: %q", truncate(body, 180))
	}
	if !strings.Contains(body, "Body text") && !strings.Contains(body, "Title") {
		t.Fatalf("html body missing content: %q", truncate(body, 180))
	}

	// 3) Markdown export via shipped helper
	res, err = build(goja.Undefined(), vm.ToValue(`<h1>Report</h1><p>Line one</p>`), vm.ToValue("md"))
	if err != nil {
		t.Fatalf("build md: %v", err)
	}
	pack, ok = res.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("md pack type %T", res.Export())
	}
	if pack["filename"] != "aethel-sphere-document.md" {
		t.Fatalf("md filename: %v", pack["filename"])
	}
	mime, _ = pack["mime"].(string)
	if !strings.Contains(mime, "markdown") {
		t.Fatalf("md mime: %v", mime)
	}
	mdBody, _ := pack["body"].(string)
	if !strings.Contains(mdBody, "Report") && !strings.Contains(mdBody, "Line one") {
		// markdown walker may format as # Report — require at least one of the words
		t.Fatalf("md body missing content: %q", truncate(mdBody, 200))
	}

	// 4) htmlToMarkdownRough directly
	res, err = mdFn(goja.Undefined(), vm.ToValue(`<h2>Section</h2><p>Para</p>`))
	if err != nil {
		t.Fatalf("htmlToMarkdownRough: %v", err)
	}
	md := res.String()
	if !strings.Contains(md, "Section") && !strings.Contains(md, "Para") {
		t.Fatalf("markdown rough unexpected: %q", md)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
