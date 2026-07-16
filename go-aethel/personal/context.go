package personal

type personalState struct {
	personal       *PersonalStore
	getAPIKey      func() string
	getOpenAIKey   func() string
	getDeepSeekKey func() string
	getGeminiKey   func() string
	getClaudeKey   func() string
}

var state *personalState

func InitState(
	personalStore *PersonalStore,
	getAPIKeyFn func() string,
	getOpenAIKeyFn func() string,
	getDeepSeekKeyFn func() string,
	getGeminiKeyFn func() string,
	getClaudeKeyFn func() string,
) {
	state = &personalState{
		personal:       personalStore,
		getAPIKey:      getAPIKeyFn,
		getOpenAIKey:   getOpenAIKeyFn,
		getDeepSeekKey: getDeepSeekKeyFn,
		getGeminiKey:   getGeminiKeyFn,
		getClaudeKey:   getClaudeKeyFn,
	}
}
