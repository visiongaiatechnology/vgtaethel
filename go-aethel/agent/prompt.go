package agent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// ────────────────────────────────────────────────────────────────────
// PROMPT STRUCTURE V2 — Stable Prefix + Dynamic Tail for DeepSeek Caching
// ────────────────────────────────────────────────────────────────────

// DeepSeekUsage captures token usage including cache metrics
type DeepSeekUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

// CacheHitRatio returns the cache efficiency (0.0–1.0)
func CacheHitRatio(u DeepSeekUsage) float64 {
	total := u.PromptCacheHitTokens + u.PromptCacheMissTokens
	if total == 0 {
		return 0
	}
	return float64(u.PromptCacheHitTokens) / float64(total)
}

// CacheAmpel returns a human-readable rating
func CacheAmpel(ratio float64) string {
	switch {
	case ratio >= 0.75:
		return "SEHR GUT"
	case ratio >= 0.50:
		return "GUT"
	case ratio >= 0.25:
		return "OKAY"
	default:
		return "SCHLECHT"
	}
}

// PromptParts is the central prompt assembly container
type PromptParts struct {
	// —— STABLE PREFIX (byte-identical across requests) ——
	StaticSystemCore string
	SecurityPolicy   string
	ToolContracts    string
	ProjectContext   string

	// —— DYNAMIC TAIL (changes per request) ——
	SessionSummary  string
	RecentHistory   string
	RetrievedMemory string
	PersonalContext string
	ToolResults     string
	CurrentUser     string
}

// StaticPrefixHashState tracks the hash of the static prefix for cache monitoring
var (
	staticPrefixHashState string
	staticPrefixTokensEst int
	staticPrefixHashMu    sync.Mutex
	promptBuilderVersion  = "v2.0"
	cacheWarmupDone       = false
	cacheWarmupMu         sync.Mutex
)

// ────────────────────────────────────────────────────────────────────
// STATIC PREFIX CORE (NIE dynamische Werte einfügen!)
// ────────────────────────────────────────────────────────────────────

const AETHEL_SYSTEM_CORE = `You are VGT AETHEL, a sovereign operating system AI with direct tool access.

IDENTITY:
- Core: GO-CORTEX v0.5
- Architecture: Sovereign AI with executive tool capabilities
- Operating Mode: Visible (full operator oversight)

TOOLS: You have access to tools for system operations.
Follow tool contracts strictly. Execute with precision.`

const AETHEL_SECURITY_POLICY = `SECURITY POLICY:
1. No self-modifying code without operator explicit approval.
2. All system commands require lease verification.
3. File operations outside workspace need mounted directory approval.
4. GUI automation requires operator presence (visible mode).
5. Network operations are restricted to approved endpoints.
6. Secret vault access requires purpose declaration.`

const TOOL_CONTRACTS = `TOOL CONTRACTS:
When a task requires an available tool, invoke the provider-native function/tool call. Never print JSON, a tool example, an arguments object, or a pseudo-envelope in normal assistant text.
The orchestrator only executes calls delivered through the native tool_calls channel; text such as {"path":"..."} has no effect and is forbidden.
Select the exact tool name from the registered definitions. For folder inspection use fs_list_dir; for a file use fs_read_file; for writing use fs_write_file.
For a complete source-code architecture map, use code_cartography. It recursively maps safe code files into a Markdown report; use its report as the basis before proposing architectural changes.
For an absolute folder outside the VGT workspace, call fs_mount_folder first with access "read" and the exact folder path, then call fs_list_dir. Never attempt to bypass the workspace jail; a mount is scoped and expires.
Tool arguments are produced by the provider SDK as structured data. Do not manually escape Windows paths in prose or JSON text.
After a tool result, evaluate the real result and either invoke the next required tool or give a final answer.
If a tool fails, report the error verbatim. Do not fabricate tool results.
Security overrides must be explicitly acknowledged.
Use vision_context when the operator asks what is visible on the desktop.
Use web_browser for web page text/DOM extraction.
Use weather_lookup for current weather questions about a city; do not simulate weather data.
Use market_lookup for current BTC, ETH, SOL or gold-price questions; GOLD is a labeled PAXG token proxy and must never be represented as official XAU fixing.
When Sphere Writer content is requested, call sphere_write_document with the complete document. A completion statement without that verified tool result is invalid.
Use media_control and youtube_control for media playback and YouTube navigation.`

const GLOBAL_WATCH_WORKFLOW = `GLOBAL WATCH & UNIFIED INTELLIGENCE WORKFLOW:
- For questions about current world events, threats, alerts, sources, feeds, regions, or a location: first call global_watch_nexus_context. It reads ONLY the unified SharedIntelStore (same model as the globe, risk scores, alerts, and briefings). There is no parallel chat-only truth and no legacy LiveNexusContext for world state.
- Also use intelligence_region_status / intelligence_explain_score / intelligence_generate_brief when the operator asks about a region, score explanation, or structured brief — they use the same SharedIntelStore.
- Distinguish layers strictly in every answer: RAW OBSERVATION (what was found), INFERENCE/Assessment (unverified classification), VERIFIED FINDING (only after sealed evidence or operator review). Never call an observation or inference "confirmed" unless Assessment.Status is verified/corroborated or Case evidence is sealed and verified.
- To direct the operator to a location, use global_watch_focus with latitude and longitude. For named regions (europe, germany, …) prefer global_watch_focus_region. To switch the UI (Live Globe, Sphere, Personal Core, …) use navigate_ui. To set feed recency use global_watch_time_window (hours). To show a readable brief use global_watch_open_report with markdown body. To record a new source-labelled signal, use global_watch_observe or intelligence_propose_observation (ingests into the unified model as raw Observation).
- For a new investigation: create a Case with intelligence_create_case or osint_case_create (single id in root+shared); seal with osint_evidence_capture (include source_event_id when promoting from the feed); propose entities with osint_entity_propose; link with osint_relation_propose only via sealed evidence id; timeline via osint_timeline_generate; report via osint_report_generate.
- Re-ID: intelligence_request_reid then dual-control intelligence_approve_reid (two different approvers). Raw PII recovery is always not_eligible — only time-bound alias metadata unlock.
- Operator verification of inferences: intelligence_set_assessment_status (verified/disputed/...). Continuity: intelligence_identity_status.
- Connector path: intelligence_connector_fetch (default builtin-rss) fetches Observations and ingests into SharedIntelStore — still RAW until assessment/review.
- Personal impact ("betrifft mich / meine Projekte"): first intelligence_sync_personal (opt-in bridge from PersonalStore), then intelligence_personal_impact. Label all impact lines as recommendations, not facts. Never mix personal memories into Cases.
- Use global_watch_toggle_layer only to adjust presentation. Do not invent layer data.
- Use global_watch_schedule_briefing only when the operator explicitly requests recurring local reports and names a cadence. It is approval-gated; never enable monitoring silently.
- Do not add a custom feed unless the operator explicitly names and approves that source. It is approval-gated and only public HTTPS endpoints are eligible.
- A Re-ID request is not execution. The system stores case-scoped aliases, not raw identities; treat the Re-ID gate as not eligible unless a separately governed mechanism exists.
- Use Nexus only for explicitly approved, durable operator knowledge. Live unified intelligence context is current situational World State, not personal memory.`

const MODEL_BEHAVIOR_RULES = `BEHAVIOR RULES:
- Be precise. Minimize prose. Maximize information density.
- Answer in the operator's language.
- For greetings, small talk and ordinary questions, answer directly in chat. Do not use or propose a computer-control tool unless the operator explicitly requests an external action.
- Code is law. Write robust, complete code.
- The persistent Go runner controls continuation. Never emit textual continuation markers.
- Report file changes at the end of tool operations.
- Never guess — search Nexus first, then ask operator.`

// ────────────────────────────────────────────────────────────────────
// TAG HELPERS
// ────────────────────────────────────────────────────────────────────

func tag(name, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return fmt.Sprintf("<%s>\n%s\n</%s>", name, content, name)
}

// ────────────────────────────────────────────────────────────────────
// PROMPT ASSEMBLER
// ────────────────────────────────────────────────────────────────────

// BuildStaticPrefix assembles the immutable prefix block
func BuildStaticPrefix(parts PromptParts) string {
	blocks := []string{
		AETHEL_SYSTEM_CORE,
	}

	if strings.TrimSpace(parts.SecurityPolicy) != "" {
		blocks = append(blocks, parts.SecurityPolicy)
	} else {
		blocks = append(blocks, AETHEL_SECURITY_POLICY)
	}

	if strings.TrimSpace(parts.ToolContracts) != "" {
		blocks = append(blocks, parts.ToolContracts)
	} else {
		blocks = append(blocks, TOOL_CONTRACTS)
	}

	blocks = append(blocks, MODEL_BEHAVIOR_RULES)
	blocks = append(blocks, GLOBAL_WATCH_WORKFLOW)

	if strings.TrimSpace(parts.ProjectContext) != "" {
		blocks = append(blocks, parts.ProjectContext)
	}

	prefix := strings.Join(blocks, "\n\n") + GetOSContextPrompt()

	// Update hash for monitoring
	h := sha256.Sum256([]byte(prefix))
	newHash := hex.EncodeToString(h[:])

	staticPrefixHashMu.Lock()
	if staticPrefixHashState != "" && staticPrefixHashState != newHash {
		log.Printf("[CACHE-BUG] Static-Prefix-Hash geändert! Alt=%s Neu=%s", staticPrefixHashState[:16], newHash[:16])
	}
	staticPrefixHashState = newHash
	staticPrefixTokensEst = len(strings.Fields(prefix)) + len(strings.Split(prefix, "\n"))
	staticPrefixHashMu.Unlock()

	return prefix
}

func GetOSContextPrompt() string {
	osName := runtime.GOOS
	osUpper := strings.ToUpper(osName)

	var sb strings.Builder
	sb.WriteString("\n\nOPERATING SYSTEM CONTEXT:\n")
	sb.WriteString(fmt.Sprintf("- Der Host-Computer läuft unter dem Betriebssystem: %s\n", osUpper))

	if osName == "windows" {
		sb.WriteString("- Nutze direkte Executables mit getrennten Argumenten. Shell-Interpreter wie PowerShell, cmd.exe, sh und bash sind für sys_exec_cmd blockiert.\n")
		sb.WriteString("- Verwende keine Shell-Verkettung, Redirection, Pipes oder Inline-Skripte. Fordere Operator-Freigabe für dedizierte Skills statt Shell-Workarounds.\n")
		sb.WriteString("- Pfade nutzen Backslashes (\\).\n")
	} else if osName == "darwin" {
		sb.WriteString("- Der Host-Computer ist ein macOS (Darwin) System.\n")
		sb.WriteString("- Nutze Unix/macOS-spezifische Befehle (bash/sh, ls, cd, open).\n")
		sb.WriteString("- Pfade nutzen Vorwärts-Slashes (/).\n")
	} else {
		sb.WriteString("- Der Host-Computer ist ein Linux-System.\n")
		sb.WriteString("- Nutze Linux-Befehle (bash/sh, ls, cd, xdotool).\n")
		sb.WriteString("- Pfade nutzen Vorwärts-Slashes (/).\n")
	}
	return sb.String()
}

func GetRuntimeContextPrompt() string {
	now := time.Now()
	zone, offsetSeconds := now.Zone()
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	return fmt.Sprintf("\n\nRUNTIME CONTEXT:\n- Lokale Zeit: %s\n- Zeitzone: %s (UTC%s%02d:%02d)\n- Nutze Zeitbezug nur, wenn er für Planung, Fristen oder Tageskontext relevant ist.",
		now.Format("2006-01-02 15:04:05 Monday"), zone, sign, offsetSeconds/3600, (offsetSeconds%3600)/60)
}

// BuildDynamicTail assembles the dynamic portion
func BuildDynamicTail(parts PromptParts) string {
	blocks := []string{}

	if s := tag("session_summary", parts.SessionSummary); s != "" {
		blocks = append(blocks, s)
	}
	if s := tag("recent_history", parts.RecentHistory); s != "" {
		blocks = append(blocks, s)
	}
	if s := tag("retrieved_memory", parts.RetrievedMemory); s != "" {
		blocks = append(blocks, s)
	}
	if s := tag("personal_context", parts.PersonalContext); s != "" {
		blocks = append(blocks, s)
	}
	if s := tag("tool_results", parts.ToolResults); s != "" {
		blocks = append(blocks, s)
	}
	if s := tag("current_user_message", parts.CurrentUser); s != "" {
		blocks = append(blocks, s)
	}

	return strings.Join(blocks, "\n\n")
}

// BuildDeepSeekMessages creates the optimized message array for DeepSeek API
func BuildDeepSeekMessages(parts PromptParts) []map[string]interface{} {
	staticPrefix := BuildStaticPrefix(parts)
	dynamicTail := BuildDynamicTail(parts)

	messages := []map[string]interface{}{
		{"role": "system", "content": staticPrefix},
		{"role": "user", "content": dynamicTail},
	}

	return messages
}

// ────────────────────────────────────────────────────────────────────
// CACHE MONITORING
// ────────────────────────────────────────────────────────────────────

// LogCacheMetrics logs usage + hit ratio after each DeepSeek response
func LogCacheMetrics(u DeepSeekUsage) {
	ratio := CacheHitRatio(u)
	ampel := CacheAmpel(ratio)

	staticPrefixHashMu.Lock()
	hash := staticPrefixHashState
	tokensEst := staticPrefixTokensEst
	staticPrefixHashMu.Unlock()

	hashShort := ""
	if len(hash) >= 16 {
		hashShort = hash[:16]
	}

	log.Printf("[DEEPSEEK-CACHE] TotalPrompt=%d | Hit=%d Miss=%d | Ratio=%.1f%% (%s) | PrefixHash=%s EstTokens=%d | BuilderVersion=%s",
		u.PromptTokens,
		u.PromptCacheHitTokens,
		u.PromptCacheMissTokens,
		ratio*100,
		ampel,
		hashShort,
		tokensEst,
		promptBuilderVersion,
	)
}

// ────────────────────────────────────────────────────────────────────
// CONVERSATION COMPACTION
// ────────────────────────────────────────────────────────────────────

const compactThreshold = 20 // Messages before compacting
const keepRecentTurns = 8   // Last N turns kept verbatim

// CompactHistory compresses old messages into a summary
func CompactHistory(messages []map[string]interface{}) (string, []map[string]interface{}) {
	if len(messages) <= compactThreshold {
		return "", messages
	}

	// Split: old messages to summarize, recent to keep
	splitIdx := len(messages) - keepRecentTurns
	if splitIdx < 0 {
		splitIdx = 0
	}

	oldMessages := messages[:splitIdx]
	recentMessages := messages[splitIdx:]

	// Build summary from old messages
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("Previous conversation summary:\n")

	turnCount := 0
	for _, msg := range oldMessages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		if role == "user" || role == "assistant" {
			if role == "user" {
				turnCount++
			}
			// Truncate long content in summary
			if len(content) > 200 {
				content = content[:197] + "..."
			}
			summaryBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", role, content))
		}
	}
	summaryBuilder.WriteString(fmt.Sprintf("(%d turns summarized)\n", turnCount))

	return summaryBuilder.String(), recentMessages
}

func CompactMessagesPreservingSystem(messages []map[string]interface{}) (string, []map[string]interface{}) {
	if len(messages) == 0 {
		return "", nil
	}
	if role, _ := messages[0]["role"].(string); role != "system" {
		return CompactHistory(messages)
	}
	summary, conversation := CompactHistory(messages[1:])
	result := make([]map[string]interface{}, 0, len(conversation)+1)
	result = append(result, messages[0])
	result = append(result, conversation...)
	return summary, result
}

// ────────────────────────────────────────────────────────────────────
// TOOL SCHEMA STABILIZATION
// ────────────────────────────────────────────────────────────────────

// SortedToolDefinitions returns tool definitions with deterministic ordering
func SortedToolDefinitions(tools []map[string]interface{}) []map[string]interface{} {
	if len(tools) == 0 {
		return tools
	}

	// Sort by name
	sort.Slice(tools, func(i, j int) bool {
		nameI, _ := tools[i]["name"].(string)
		nameJ, _ := tools[j]["name"].(string)
		return nameI < nameJ
	})

	// For each tool, sort its JSON fields deterministically
	for i := range tools {
		if function, ok := tools[i]["function"].(map[string]interface{}); ok {
			if params, ok := function["parameters"].(map[string]interface{}); ok {
				if properties, ok := params["properties"].(map[string]interface{}); ok {
					// Create sorted property map
					sortedProps := make(map[string]interface{})
					keys := make([]string, 0, len(properties))
					for k := range properties {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, k := range keys {
						sortedProps[k] = properties[k]
					}
					params["properties"] = sortedProps
				}
				// Sort required array if present
				if required, ok := params["required"].([]interface{}); ok {
					sort.Slice(required, func(a, b int) bool {
						strA, _ := required[a].(string)
						strB, _ := required[b].(string)
						return strA < strB
					})
				}
			}
		}
	}

	return tools
}

// ────────────────────────────────────────────────────────────────────
// FILE CONTEXT WITH HASH REFERENCES
// ────────────────────────────────────────────────────────────────────

// FileRef represents a file with content hash for caching
type FileRef struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Summary string `json:"summary"`
}

// MakeFileRef creates a file reference with hash
func MakeFileRef(path string, maxSummaryLen int) (*FileRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	h := sha256.Sum256(data)
	hash := hex.EncodeToString(h[:])

	content := string(data)
	if maxSummaryLen > 0 && len(content) > maxSummaryLen {
		content = content[:maxSummaryLen]
	}
	// Take first meaningful line as summary
	lines := strings.SplitN(content, "\n", 2)
	summary := strings.TrimSpace(lines[0])
	if len(summary) > 200 {
		summary = summary[:197] + "..."
	}

	return &FileRef{
		Path:    path,
		SHA256:  hash,
		Summary: summary,
	}, nil
}

// ────────────────────────────────────────────────────────────────────
// CACHE WARMUP
// ────────────────────────────────────────────────────────────────────

const deepseekAPIURL = "https://api.deepseek.com/chat/completions"

// WarmupDeepSeekCache sends a minimal request to prime the prefix cache
func WarmupDeepSeekCache(apiKey string) {
	cacheWarmupMu.Lock()
	if cacheWarmupDone {
		cacheWarmupMu.Unlock()
		return
	}
	cacheWarmupMu.Unlock()

	if apiKey == "" || !strings.HasPrefix(apiKey, "sk-") {
		log.Printf("[CACHE-WARMUP] Übersprungen — kein DeepSeek-Key")
		return
	}

	parts := PromptParts{}
	staticPrefix := BuildStaticPrefix(parts)

	payload := map[string]interface{}{
		"model":      "deepseek-v4-flash",
		"messages":   []map[string]interface{}{{"role": "system", "content": staticPrefix}, {"role": "user", "content": "ACK only."}},
		"thinking":   map[string]interface{}{"type": "enabled"},
		"max_tokens": 1,
		"stream":     false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[CACHE-WARMUP] Marshal-Fehler: %v", err)
		return
	}

	go func() {
		req, err := http.NewRequest("POST", deepseekAPIURL, bytes.NewBuffer(body))
		if err != nil {
			log.Printf("[CACHE-WARMUP] Request-Fehler: %v", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[CACHE-WARMUP] HTTP-Fehler: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			var result struct {
				Usage DeepSeekUsage `json:"usage"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&result)
			log.Printf("[CACHE-WARMUP] Erfolgreich | Hit=%d Miss=%d | Ratio=%.1f%%",
				result.Usage.PromptCacheHitTokens,
				result.Usage.PromptCacheMissTokens,
				CacheHitRatio(result.Usage)*100,
			)
		} else {
			log.Printf("[CACHE-WARMUP] HTTP %d — übersprungen", resp.StatusCode)
		}

		cacheWarmupMu.Lock()
		cacheWarmupDone = true
		cacheWarmupMu.Unlock()
	}()
}
