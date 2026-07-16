package system

type systemState struct {
	release *ReleaseService
}

var state *systemState

func InitState(releaseService *ReleaseService) {
	state = &systemState{
		release: releaseService,
	}
}
