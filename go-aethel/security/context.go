package security

type securityState struct {
	MountAllows func(path string, access MountAccess) bool
}

var state *securityState

func InitState(mountAllowsFn func(path string, access MountAccess) bool) {
	state = &securityState{
		MountAllows: mountAllowsFn,
	}
}
