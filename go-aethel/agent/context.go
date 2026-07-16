package agent

import (
	"net/http"
	"go-aethel/provider"
	"go-aethel/security"
	"go-aethel/skills"
)

type agentState struct {
	runs      *RunEngine
	policy    *security.PolicyEngine
	skills    *skills.SkillRegistry
	providers *provider.ProviderRegistry
	audit     *security.AuditLogger
	getAPIKey func() string
}

var ChatHandler func(w http.ResponseWriter, r *http.Request)

var state *agentState

func InitState(
	runsEngine *RunEngine,
	policyEngine *security.PolicyEngine,
	skillsRegistry *skills.SkillRegistry,
	providersRegistry *provider.ProviderRegistry,
	auditLogger *security.AuditLogger,
	getAPIKeyFn func() string,
) {
	state = &agentState{
		runs:      runsEngine,
		policy:    policyEngine,
		skills:    skillsRegistry,
		providers: providersRegistry,
		audit:     auditLogger,
		getAPIKey: getAPIKeyFn,
	}
}
