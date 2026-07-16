package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CognitiveTier selects the least expensive execution path that can still
// satisfy the objective. Classification is deterministic and does not spend
// provider tokens.
type CognitiveTier string

const (
	CognitiveDirect      CognitiveTier = "direct"
	CognitiveAssisted    CognitiveTier = "assisted"
	CognitiveOperational CognitiveTier = "operational"
	CognitiveExtended    CognitiveTier = "extended"
)

type CognitivePhase string

const (
	PhasePerceiving CognitivePhase = "perceiving"
	PhasePlanning   CognitivePhase = "planning"
	PhaseExecuting  CognitivePhase = "executing"
	PhaseVerifying  CognitivePhase = "verifying"
	PhaseReflecting CognitivePhase = "reflecting"
	PhaseCompleted  CognitivePhase = "completed"
	PhasePaused     CognitivePhase = "paused"
	PhaseFailed     CognitivePhase = "failed"
)

type TokenBudget struct {
	MaxInputTokens  int  `json:"max_input_tokens"`
	MaxOutputTokens int  `json:"max_output_tokens"`
	MaxTotalTokens  int  `json:"max_total_tokens"`
	InputTokens     int  `json:"input_tokens"`
	OutputTokens    int  `json:"output_tokens"`
	CachedTokens    int  `json:"cached_tokens"`
	Exhausted       bool `json:"exhausted,omitempty"`
}

type RunEvidence struct {
	StepID       string    `json:"step_id"`
	ToolName     string    `json:"tool_name,omitempty"`
	Expectation  string    `json:"expectation"`
	ResultDigest string    `json:"result_digest,omitempty"`
	Verified     bool      `json:"verified"`
	StateChanged bool      `json:"state_changed,omitempty"`
	ObservedAt   time.Time `json:"observed_at"`
}

type ContinuityState struct {
	Tier                  CognitiveTier  `json:"tier"`
	Phase                 CognitivePhase `json:"phase"`
	Goal                  string         `json:"goal"`
	ExpectedEffects       []string       `json:"expected_effects,omitempty"`
	Evidence              []RunEvidence  `json:"evidence,omitempty"`
	Reflection            string         `json:"reflection,omitempty"`
	LastActionFingerprint string         `json:"last_action_fingerprint,omitempty"`
	RepeatedActionCount   int            `json:"repeated_action_count,omitempty"`
	TokenBudget           TokenBudget    `json:"token_budget"`
}

func classifyCognitiveTier(objective string, sphereActive bool) CognitiveTier {
	text := strings.ToLower(strings.TrimSpace(objective))
	operational := sphereActive || shouldEnableAgentTools(text)
	if operational {
		if len([]rune(text)) > 1800 || containsAny(text,
			"code kartografie", "code cartography", "migration", "refactor", "architektur",
			"architecture", "agenten-team", "agent team", "vollständig analys", "complete audit") {
			return CognitiveExtended
		}
		return CognitiveOperational
	}
	if len([]rune(text)) > 1200 || containsAny(text, "analysiere", "bewerte", "vergleiche", "begründe", "analyze", "compare", "evaluate") {
		return CognitiveAssisted
	}
	return CognitiveDirect
}

func defaultTokenBudget(tier CognitiveTier) TokenBudget {
	switch tier {
	case CognitiveDirect:
		return TokenBudget{MaxInputTokens: 16_000, MaxOutputTokens: 2_000, MaxTotalTokens: 18_000}
	case CognitiveAssisted:
		return TokenBudget{MaxInputTokens: 36_000, MaxOutputTokens: 6_000, MaxTotalTokens: 42_000}
	case CognitiveOperational:
		return TokenBudget{MaxInputTokens: 96_000, MaxOutputTokens: 16_000, MaxTotalTokens: 112_000}
	default:
		return TokenBudget{MaxInputTokens: 220_000, MaxOutputTokens: 36_000, MaxTotalTokens: 256_000}
	}
}

func newContinuityState(objective string, sphereActive bool) ContinuityState {
	tier := classifyCognitiveTier(objective, sphereActive)
	probe := AgentRun{Objective: objective, SphereActive: sphereActive}
	return ContinuityState{
		Tier:            tier,
		Phase:           PhasePerceiving,
		Goal:            ClampRunDetail(objective),
		ExpectedEffects: requiredExecutionEffects(probe),
		TokenBudget:     defaultTokenBudget(tier),
	}
}

func (state *ContinuityState) recordUsage(input, output, cached int) bool {
	if state == nil || input < 0 || output < 0 || cached < 0 {
		return false
	}
	state.TokenBudget.InputTokens += input
	state.TokenBudget.OutputTokens += output
	state.TokenBudget.CachedTokens += cached
	total := state.TokenBudget.InputTokens + state.TokenBudget.OutputTokens
	state.TokenBudget.Exhausted = state.TokenBudget.InputTokens > state.TokenBudget.MaxInputTokens ||
		state.TokenBudget.OutputTokens > state.TokenBudget.MaxOutputTokens ||
		total > state.TokenBudget.MaxTotalTokens
	return state.TokenBudget.Exhausted
}

func (state *ContinuityState) registerAction(toolName string, arguments json.RawMessage) error {
	if state == nil {
		return nil
	}
	digest := sha256.Sum256(append(append([]byte(toolName), 0), arguments...))
	fingerprint := hex.EncodeToString(digest[:12])
	if fingerprint == state.LastActionFingerprint {
		state.RepeatedActionCount++
	} else {
		state.LastActionFingerprint = fingerprint
		state.RepeatedActionCount = 1
	}
	if state.RepeatedActionCount > 2 {
		return fmt.Errorf("continuity loop guard rejected repeated action %s", toolName)
	}
	return nil
}

func (state *ContinuityState) addEvidence(step RunStep) {
	if state == nil {
		return
	}
	expectation := strings.TrimSpace(step.ExpectedContains)
	if expectation == "" {
		expectation = "Tool completed without an execution error."
	}
	digest := ""
	if step.Result != "" {
		sum := sha256.Sum256([]byte(step.Result))
		digest = "sha256:" + hex.EncodeToString(sum[:])
	}
	state.Evidence = append(state.Evidence, RunEvidence{
		StepID: step.ID, ToolName: step.ToolName, Expectation: expectation,
		ResultDigest: digest, Verified: step.Status == StepVerified && step.Error == "" && step.Result != "",
		StateChanged: step.EvidenceChanged, ObservedAt: time.Now().UTC(),
	})
	if len(state.Evidence) > 96 {
		state.Evidence = state.Evidence[len(state.Evidence)-96:]
	}
}

func (state *ContinuityState) finalize(run AgentRun, succeeded bool) {
	if state == nil {
		return
	}
	verified := 0
	failed := 0
	for _, evidence := range state.Evidence {
		if evidence.Verified {
			verified++
		} else {
			failed++
		}
	}
	state.Phase = PhaseReflecting
	state.Reflection = fmt.Sprintf("Run %s with %d verified evidence records, %d unresolved records, %d model turns and %d tool calls.",
		map[bool]string{true: "completed", false: "failed"}[succeeded], verified, failed, run.AgentTurn, run.ToolCalls)
	if succeeded {
		state.Phase = PhaseCompleted
	} else {
		state.Phase = PhaseFailed
	}
}

func shouldUseOrchestrator(tier CognitiveTier, objective string, expectedEffects []string) bool {
	return tier == CognitiveOperational || tier == CognitiveExtended || len(expectedEffects) > 0 || shouldEnableAgentTools(objective)
}

func reasoningPolicy(tier CognitiveTier, modelID string, requested ...string) (effort, visibility string) {
	visibility = "hidden"
	id := strings.ToLower(modelID)
	preference := "auto"
	if len(requested) > 0 && strings.TrimSpace(requested[0]) != "" {
		preference = strings.ToLower(strings.TrimSpace(requested[0]))
	}
	allowed := func(options ...string) string {
		if preference == "auto" {
			return ""
		}
		for _, option := range options {
			if preference == option {
				return preference
			}
		}
		return ""
	}
	if strings.Contains(id, "gpt-5.6") {
		if selected := allowed("none", "low", "medium", "high", "xhigh", "max"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "gpt-oss") {
		if selected := allowed("low", "medium", "high"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "qwen3") {
		if selected := allowed("none", "default"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "gemini") {
		if selected := allowed("minimal", "low", "medium", "high"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "deepseek") {
		if selected := allowed("none", "high", "max"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "claude-fable-5") || strings.Contains(id, "claude-opus-4-8") {
		if selected := allowed("low", "medium", "high", "xhigh", "max"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "claude-sonnet-4-6") {
		if selected := allowed("low", "medium", "high", "max"); selected != "" {
			return selected, visibility
		}
	}
	if strings.Contains(id, "qwen3.6") {
		if tier == CognitiveDirect {
			return "none", visibility
		}
		return "default", visibility
	}
	if strings.Contains(id, "deepseek") && tier == CognitiveDirect {
		return "none", visibility
	}
	switch tier {
	case CognitiveDirect:
		return "low", visibility
	case CognitiveAssisted, CognitiveOperational:
		return "medium", visibility
	default:
		return "high", visibility
	}
}

// compactAgentContext bounds provider context without another model call. The
// encrypted run journal remains complete; only the provider projection is
// compacted. A deterministic checkpoint carries goal and verified effects.
func compactAgentContext(run AgentRun, messages []json.RawMessage) []json.RawMessage {
	maxMessages := 8
	maxBytes := 24 * 1024
	switch run.Continuity.Tier {
	case CognitiveAssisted:
		maxMessages, maxBytes = 12, 48*1024
	case CognitiveOperational:
		maxMessages, maxBytes = 20, 96*1024
	case CognitiveExtended:
		maxMessages, maxBytes = 28, 160*1024
	}
	totalBytes := 0
	for _, message := range messages {
		totalBytes += len(message)
	}
	if len(messages) <= maxMessages && totalBytes <= maxBytes {
		return append([]json.RawMessage(nil), messages...)
	}

	start := len(messages)
	selectedBytes := 0
	for start > 0 && len(messages)-start < maxMessages {
		candidate := messages[start-1]
		if selectedBytes+len(candidate) > maxBytes && start < len(messages) {
			break
		}
		selectedBytes += len(candidate)
		start--
	}
	for start < len(messages) && rawMessageRole(messages[start]) == "tool" {
		start++
	}
	checkpoint, err := json.Marshal(map[string]interface{}{
		"role":    "user",
		"content": "AETHEL_CONTINUITY_CHECKPOINT\n" + continuityCheckpoint(run),
	})
	if err != nil {
		return append([]json.RawMessage(nil), messages[start:]...)
	}
	result := make([]json.RawMessage, 0, len(messages)-start+1)
	result = append(result, checkpoint)
	result = append(result, messages[start:]...)
	return result
}

func rawMessageRole(raw json.RawMessage) string {
	var envelope struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return ""
	}
	return envelope.Role
}

func continuityCheckpoint(run AgentRun) string {
	verified := make([]string, 0, len(run.Steps))
	for _, step := range run.Steps {
		if step.Kind == RunStepTool && step.Status == StepVerified && step.Error == "" {
			verified = append(verified, step.ToolName)
		}
	}
	return fmt.Sprintf("Goal: %s\nPhase: %s\nRequired effects: %s\nVerified tools: %s\nContinue from verified state; do not repeat completed actions.",
		run.Objective, run.Continuity.Phase, strings.Join(run.Continuity.ExpectedEffects, ", "), strings.Join(verified, ", "))
}
