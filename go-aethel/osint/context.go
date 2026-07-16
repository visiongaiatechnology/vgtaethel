package osint

type osintState struct {
	osint *OSINTEngine
}

var state *osintState

func InitState(osintEngine *OSINTEngine) {
	state = &osintState{
		osint: osintEngine,
	}
}
