package handlers

import (
	"encoding/json"
	"testing"

	"go-aethel/security"
	"go-aethel/skills"
)

type allowlistTestSkill struct{ name string }

func (s allowlistTestSkill) Name() string                            { return s.name }
func (s allowlistTestSkill) Description() string                     { return "test capability" }
func (s allowlistTestSkill) RiskLevel() security.RiskLevel           { return security.RiskLow }
func (s allowlistTestSkill) Execute(json.RawMessage) (string, error) { return "ok", nil }
func (s allowlistTestSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "additionalProperties": false}
}

func TestExplicitEmptyToolAllowlistExposesNoTools(t *testing.T) {
	previous := state
	registry := skills.NewSkillRegistry()
	registry.Register(allowlistTestSkill{name: "safe_test_tool"})
	state = &appState{skills: registry}
	t.Cleanup(func() { state = previous })

	empty := []string{}
	if tools := requestToolDefinitions(ChatRequest{ToolAllowlist: &empty}); len(tools) != 0 {
		t.Fatalf("explicit empty allowlist exposed tools: %+v", tools)
	}
	allowed := []string{"safe_test_tool"}
	if tools := requestToolDefinitions(ChatRequest{ToolAllowlist: &allowed}); len(tools) != 1 {
		t.Fatalf("explicit tool allowlist was not honored: %+v", tools)
	}
	if tools := requestToolDefinitions(ChatRequest{}); len(tools) != 1 {
		t.Fatalf("legacy direct chat unexpectedly lost registered tools: %+v", tools)
	}
}
