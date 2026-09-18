// STATUS: DIAMANT VGT SUPREME
package security

type securityState struct {
	MountAllows     func(path string, access MountAccess) bool
	ActiveWorkspace func() string
}

var state *securityState

func InitState(mountAllowsFn func(path string, access MountAccess) bool, activeWorkspaceFn ...func() string) {
	var wsFn func() string
	if len(activeWorkspaceFn) > 0 {
		wsFn = activeWorkspaceFn[0]
	}
	state = &securityState{
		MountAllows:     mountAllowsFn,
		ActiveWorkspace: wsFn,
	}
}

func SetActiveWorkspaceProvider(activeWorkspaceFn func() string) {
	if state != nil {
		state.ActiveWorkspace = activeWorkspaceFn
	}
}

