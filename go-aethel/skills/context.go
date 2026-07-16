package skills

import (
	"time"

	"go-aethel/intelligence"
	"go-aethel/osint"
	"go-aethel/personal"
	"go-aethel/security"
)

type skillsState struct {
	intelSources     *intelligence.IntelligenceSourceRegistry
	personal         *personal.PersonalStore
	intelMonitor     *osint.GlobalWatchMonitor
	intel            *intelligence.IntelligenceStore
	osint            *osint.OSINTEngine
	GetMounts        func() []security.MountGrant
	AddMount         func(dir string, access security.MountAccess, duration time.Duration) error
	MountAllows      func(path string, access security.MountAccess) bool
	RecordFileChange func(path string, added, removed int)
}

var state *skillsState

func InitState(
	intelSourcesRegistry *intelligence.IntelligenceSourceRegistry,
	personalStore *personal.PersonalStore,
	intelMonitorMonitor *osint.GlobalWatchMonitor,
	intelStore *intelligence.IntelligenceStore,
	osintEngine *osint.OSINTEngine,
	getMountsFn func() []security.MountGrant,
	addMountFn func(dir string, access security.MountAccess, duration time.Duration) error,
	mountAllowsFn func(path string, access security.MountAccess) bool,
	recordFileChangeFn func(path string, added, removed int),
) {
	state = &skillsState{
		intelSources:     intelSourcesRegistry,
		personal:         personalStore,
		intelMonitor:     intelMonitorMonitor,
		intel:            intelStore,
		osint:            osintEngine,
		GetMounts:        getMountsFn,
		AddMount:         addMountFn,
		MountAllows:      mountAllowsFn,
		RecordFileChange: recordFileChangeFn,
	}
}
