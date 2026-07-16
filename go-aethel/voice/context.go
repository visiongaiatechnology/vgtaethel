package voice

type voiceState struct {
	getAPIKey    func() string
	getOpenAIKey func() string
}

var state *voiceState

func InitState(getAPIKeyFn func() string, getOpenAIKeyFn func() string) {
	state = &voiceState{
		getAPIKey:    getAPIKeyFn,
		getOpenAIKey: getOpenAIKeyFn,
	}
}
