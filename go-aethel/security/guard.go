package security

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Capability defines the granular permission type
type Capability string

const (
	CapSysExec       Capability = "system.exec"
	CapFsRead        Capability = "fs.read"
	CapFsWrite       Capability = "fs.write"
	CapFsDelete      Capability = "fs.delete"
	CapFsMount       Capability = "fs.mount"
	CapGuiMoveMouse  Capability = "gui.move_mouse"
	CapGuiClick      Capability = "gui.click"
	CapGuiType       Capability = "gui.type"
	CapGuiPressKey   Capability = "gui.press_key"
	CapMediaControl  Capability = "media.control"
	CapBrowserOpen   Capability = "browser.open_url"
	CapBrowserCtl    Capability = "browser.control"
	CapBrowserRead   Capability = "browser.read_page"
	CapScreenRead    Capability = "screen.read"
	CapMemoryRead    Capability = "memory.read"
	CapMemoryWrite   Capability = "memory.write"
	CapSecretUse     Capability = "secret.use"
	CapTaskCreate    Capability = "task.create"
	CapTaskSchedule  Capability = "task.schedule"
	CapMessagingSend Capability = "messaging.send"
	CapCodexHandoff  Capability = "codex.delegate_task"
	CapWeatherRead   Capability = "weather.read"
	CapMarketRead    Capability = "market.read"
	CapSphereWrite   Capability = "sphere.writer.write"
	CapIntelRead     Capability = "intelligence.read"
	CapIntelWrite    Capability = "intelligence.write"
	CapIntelSources  Capability = "intelligence.manage_sources"
	CapUINavigation  Capability = "ui.navigate"
)

// RiskLevel defines the danger category of an action
type RiskLevel string

const (
	RiskSafe      RiskLevel = "Safe"
	RiskLow       RiskLevel = "Low"
	RiskModerate  RiskLevel = "Moderate"
	RiskHigh      RiskLevel = "High"
	RiskCritical  RiskLevel = "Critical"
	RiskForbidden RiskLevel = "Forbidden"
)

// Scope defines the limits of a permission lease
type Scope struct {
	Apps             []string `json:"apps,omitempty"`
	Actions          []string `json:"actions,omitempty"`
	ForbiddenKeys    []string `json:"forbidden_keys,omitempty"`
	ForbiddenTargets []string `json:"forbidden_targets,omitempty"`
}

// PermissionLease represents a temporary permission token
type PermissionLease struct {
	LeaseID             string     `json:"lease_id"`
	Capability          Capability `json:"capability"`
	Scope               Scope      `json:"scope"`
	CreatedAt           time.Time  `json:"created_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	RequiresVisibleMode bool       `json:"requires_visible_mode"`
	Revocable           bool       `json:"revocable"`
	ApprovedBy          string     `json:"approved_by"`     // e.g. "user"
	ApprovalMethod      string     `json:"approval_method"` // e.g. "ui", "voice"
}

// LeaseManager manages active permission leases
type LeaseManager struct {
	mu        sync.RWMutex
	leases    map[string]PermissionLease
	cachePath string
}

func NewLeaseManager(cachePath string) *LeaseManager {
	lm := &LeaseManager{
		leases:    make(map[string]PermissionLease),
		cachePath: cachePath,
	}
	_ = lm.loadFromDisk()
	return lm
}

func (lm *LeaseManager) loadFromDisk() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	data, err := ReadAuthorityFile(lm.cachePath)
	if err != nil {
		return err
	}

	var cached []PermissionLease
	if err := json.Unmarshal(data, &cached); err != nil {
		return err
	}

	now := time.Now()
	for _, l := range cached {
		if now.Before(l.ExpiresAt) {
			lm.leases[l.LeaseID] = l
		}
	}
	return nil
}

func (lm *LeaseManager) saveToDiskLocked() error {
	var list []PermissionLease
	now := time.Now()
	for _, l := range lm.leases {
		if now.Before(l.ExpiresAt) {
			list = append(list, l)
		}
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return WriteSealedFile(lm.cachePath, data)
}

func (lm *LeaseManager) AddLease(l PermissionLease) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	previous, replaced := lm.leases[l.LeaseID]
	lm.leases[l.LeaseID] = l
	if err := lm.saveToDiskLocked(); err != nil {
		if replaced {
			lm.leases[l.LeaseID] = previous
		} else {
			delete(lm.leases, l.LeaseID)
		}
		return err
	}
	return nil
}

func (lm *LeaseManager) RevokeLease(leaseID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	previous, exists := lm.leases[leaseID]
	if exists {
		delete(lm.leases, leaseID)
	}
	if exists {
		if err := lm.saveToDiskLocked(); err != nil {
			lm.leases[leaseID] = previous
			return false
		}
	}
	return exists
}

func (lm *LeaseManager) GetLeases() []PermissionLease {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	var list []PermissionLease
	now := time.Now()
	for _, l := range lm.leases {
		if now.Before(l.ExpiresAt) {
			list = append(list, l)
		}
	}
	return list
}

func (lm *LeaseManager) CheckLease(cap Capability, target string, action string) (bool, string) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	now := time.Now()
	for _, lease := range lm.leases {
		if lease.Capability == cap && now.Before(lease.ExpiresAt) {
			keyMatchedForbidden := false
			for _, fk := range lease.Scope.ForbiddenKeys {
				if fk != "" && strings.Contains(strings.ToLower(target), strings.ToLower(fk)) {
					keyMatchedForbidden = true
					break
				}
			}
			if keyMatchedForbidden {
				continue
			}
			// Check forbidden targets
			targetMatchedForbidden := false
			for _, ft := range lease.Scope.ForbiddenTargets {
				if ft != "" && strings.Contains(strings.ToLower(target), strings.ToLower(ft)) {
					targetMatchedForbidden = true
					break
				}
			}
			if targetMatchedForbidden {
				continue
			}

			// Check allowed actions if defined
			if len(lease.Scope.Actions) > 0 {
				actionMatched := false
				for _, act := range lease.Scope.Actions {
					if strings.EqualFold(act, action) {
						actionMatched = true
						break
					}
				}
				if !actionMatched {
					continue
				}
			}

			// Matched active lease!
			return true, lease.LeaseID
		}
	}
	return false, ""
}

// AuditEntry represents a recorded transaction log
type AuditEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Actor         string    `json:"actor"` // "aethel"
	Operation     string    `json:"operation"`
	Target        string    `json:"target"`
	Risk          RiskLevel `json:"risk"`
	LeaseID       string    `json:"lease_id,omitempty"`
	Decision      string    `json:"decision"` // "allowed", "blocked", "requested_approval", "override"
	Reason        string    `json:"reason"`
	InputSummary  string    `json:"input_summary"`
	ResultSummary string    `json:"result_summary,omitempty"`
	PrevHash      string    `json:"prev_hash"`
	Hash          string    `json:"hash"`
}

func (e *AuditEntry) computeHash(prevHash string) string {
	type HashPayload struct {
		Timestamp     string `json:"timestamp"`
		Actor         string `json:"actor"`
		Operation     string `json:"operation"`
		Target        string `json:"target"`
		Risk          string `json:"risk"`
		LeaseID       string `json:"lease_id,omitempty"`
		Decision      string `json:"decision"`
		Reason        string `json:"reason"`
		InputSummary  string `json:"input_summary"`
		ResultSummary string `json:"result_summary,omitempty"`
		PrevHash      string `json:"prev_hash"`
	}
	p := HashPayload{
		Timestamp:     e.Timestamp.Format(time.RFC3339),
		Actor:         e.Actor,
		Operation:     e.Operation,
		Target:        e.Target,
		Risk:          string(e.Risk),
		LeaseID:       e.LeaseID,
		Decision:      e.Decision,
		Reason:        e.Reason,
		InputSummary:  e.InputSummary,
		ResultSummary: e.ResultSummary,
		PrevHash:      prevHash,
	}
	data, _ := json.Marshal(p)
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// AuditLogger maintains the append-only cryptographic log trail
type AuditLogger struct {
	mu         sync.Mutex
	filePath   string
	lastHash   string
	isTampered bool
	entries    []AuditEntry
}

func NewAuditLogger(filePath string) *AuditLogger {
	al := &AuditLogger{
		filePath: filePath,
		lastHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	_ = al.LoadAndVerify()
	return al
}

func (al *AuditLogger) LoadAndVerify() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	data, sealed, err := ReadSealedFile(al.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			al.entries = []AuditEntry{}
			return nil
		}
		al.isTampered = true
		return err
	}

	var entries []AuditEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		al.isTampered = true
		return err
	}

	currentPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	for i, entry := range entries {
		expectedHash := entry.computeHash(currentPrevHash)
		if entry.Hash != expectedHash {
			al.isTampered = true
			return fmt.Errorf("audit log tampered at block %d", i)
		}
		if entry.PrevHash != currentPrevHash {
			al.isTampered = true
			return fmt.Errorf("audit log chain gap at block %d", i)
		}
		currentPrevHash = entry.Hash
	}

	needsRewrite := !sealed
	for i := range entries {
		if !strings.HasPrefix(entries[i].InputSummary, "sha256:") {
			entries[i].InputSummary = auditInputDigest(entries[i].InputSummary)
			needsRewrite = true
		}
	}
	if needsRewrite {
		currentPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
		for i := range entries {
			entries[i].PrevHash = currentPrevHash
			entries[i].Hash = entries[i].computeHash(currentPrevHash)
			currentPrevHash = entries[i].Hash
		}
	}
	al.lastHash = currentPrevHash
	al.entries = entries
	al.isTampered = false
	if needsRewrite {
		marshaled, marshalErr := json.MarshalIndent(entries, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		if writeErr := WriteSealedFile(al.filePath, marshaled); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func (al *AuditLogger) Log(actor, operation, target string, risk RiskLevel, leaseID, decision, reason, input string) (string, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.isTampered {
		return "", errors.New("audit chain is locked after an integrity failure")
	}
	previousHash := al.lastHash
	entry := AuditEntry{
		Timestamp:    time.Now(),
		Actor:        actor,
		Operation:    operation,
		Target:       target,
		Risk:         risk,
		LeaseID:      leaseID,
		Decision:     decision,
		Reason:       reason,
		InputSummary: auditInputDigest(input),
		PrevHash:     al.lastHash,
	}

	entry.Hash = entry.computeHash(al.lastHash)
	al.lastHash = entry.Hash

	al.entries = append(al.entries, entry)
	marshaled, err := json.MarshalIndent(al.entries, "", "  ")
	if err != nil {
		al.entries = al.entries[:len(al.entries)-1]
		al.lastHash = previousHash
		return "", err
	}
	err = WriteSealedFile(al.filePath, marshaled)
	if err != nil {
		al.entries = al.entries[:len(al.entries)-1]
		al.lastHash = previousHash
	}
	return entry.Hash, err
}

func auditInputDigest(input string) string {
	digest := sha256.Sum256([]byte(input))
	return fmt.Sprintf("sha256:%x bytes:%d", digest, len(input))
}

func (al *AuditLogger) ValidateChain() error {
	return al.LoadAndVerify()
}

func (al *AuditLogger) IsTampered() bool {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.isTampered
}

func (al *AuditLogger) Status() (bool, string) {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.isTampered, al.lastHash
}

func (al *AuditLogger) GetLogs() []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()
	return append([]AuditEntry(nil), al.entries...)
}

// ThreatReport represents the safety scan results
type ThreatReport struct {
	IsSafe     bool       `json:"is_safe"`
	Threats    []string   `json:"threats"`
	RiskScore  int        `json:"risk_score"`
	RiskLevel  RiskLevel  `json:"risk_level"`
	Capability Capability `json:"capability"`
}

// SecurityGuard handles standard regex safety scans and maps tools to capabilities
type SecurityGuard struct {
	reShellInjection *regexp.Regexp
	reDestructive    *regexp.Regexp
	rePathTraversal  *regexp.Regexp
	reNetExfil       *regexp.Regexp
}

func NewSecurityGuard() *SecurityGuard {
	return &SecurityGuard{
		reShellInjection: regexp.MustCompile(`([;&|><` + "`" + `$]|\$\(|\)\))`),
		reDestructive:    regexp.MustCompile(`(?i)\b(rm\s+-[rf]+|mkfs|dd|shred|wipe|format|:[\(\)\{\}\|:&;]+)\b`),
		rePathTraversal:  regexp.MustCompile(`(\.\./|\.\.\\|/etc/|C:\\Windows|/var/|/usr/)`),
		reNetExfil:       regexp.MustCompile(`(?i)\b(nc|netcat|ncat|curl|wget|ssh)\b`),
	}
}

func (sg *SecurityGuard) Scan(toolName string, args string) ThreatReport {
	var threats []string
	riskScore := 0
	riskLevel := RiskSafe
	var cap Capability

	// Global Check: Path Traversal
	if sg.rePathTraversal.MatchString(args) {
		threats = append(threats, "PATH_TRAVERSAL_DETECTED: System-Zugriff außerhalb der Sandbox blockiert.")
		riskScore += 80
		riskLevel = RiskCritical
	}

	switch toolName {
	case "sys_exec_cmd":
		cap = CapSysExec
		riskLevel = RiskHigh
		riskScore = 70

		var execArgs struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(args), &execArgs); err == nil {
			cmd := strings.ToLower(strings.TrimSpace(execArgs.Command))
			joinedArgs := strings.ToLower(strings.Join(execArgs.Args, " "))
			switch cmd {
			case "powershell", "powershell.exe", "pwsh", "cmd", "cmd.exe", "sh", "bash", "zsh":
				threats = append(threats, fmt.Sprintf("SHELL_INTERPRETER_DETECTED: %s", cmd))
				riskScore = 80
				riskLevel = RiskHigh
			}
			if cmd == "rm" && strings.Contains(joinedArgs, "-rf") {
				threats = append(threats, "DESTRUCTIVE_COMMAND_DETECTED: rm -rf")
				riskScore = 100
				riskLevel = RiskForbidden
			}
			if cmd == "dd" || cmd == "mkfs" || cmd == "format" || cmd == "shred" || cmd == "wipe" {
				threats = append(threats, fmt.Sprintf("DESTRUCTIVE_COMMAND_DETECTED: %s", cmd))
				riskScore = 100
				riskLevel = RiskForbidden
			}
		}

		if sg.reShellInjection.MatchString(args) {
			match := sg.reShellInjection.FindString(args)
			threats = append(threats, fmt.Sprintf("SHELL_INJECTION_DETECTED: Illegales Zeichen '%s'", match))
			riskScore = 90
			riskLevel = RiskCritical
		}

		if sg.reDestructive.MatchString(args) {
			match := sg.reDestructive.FindString(args)
			threats = append(threats, fmt.Sprintf("DESTRUCTIVE_COMMAND_DETECTED: Sabotageversuch '%s'", match))
			riskScore = 100
			riskLevel = RiskForbidden
		}

		if sg.reNetExfil.MatchString(args) {
			match := sg.reNetExfil.FindString(args)
			threats = append(threats, fmt.Sprintf("NETWORK_TOOL_DETECTED: Exfiltration über '%s'", match))
			riskScore = 80
			riskLevel = RiskHigh
		}

	case "fs_read_file":
		cap = CapFsRead
		riskLevel = RiskLow
		riskScore = 15

	case "fs_list_dir":
		cap = CapFsRead
		riskLevel = RiskLow
		riskScore = 10

	case "fs_write_file", "fs_replace_file_content":
		cap = CapFsWrite
		riskLevel = RiskModerate
		riskScore = 40

		lowerArgs := strings.ToLower(args)
		if strings.Contains(lowerArgs, ".sh") || strings.Contains(lowerArgs, ".exe") || strings.Contains(lowerArgs, ".bat") || strings.Contains(lowerArgs, ".cmd") {
			threats = append(threats, "EXECUTABLE_WRITE_ATTEMPT: Erstellung ausführbarer Dateien ist restricted.")
			riskScore = 85
			riskLevel = RiskCritical
		}

	case "fs_restore_snapshot":
		cap = CapFsWrite
		riskLevel = RiskCritical
		riskScore = 80

	case "fs_mount_folder":
		cap = CapFsMount
		riskLevel = RiskModerate
		riskScore = 50

	case "code_cartography":
		cap = CapFsWrite
		riskLevel = RiskModerate
		riskScore = 45

	case "nexus_save":
		cap = CapMemoryWrite
		riskLevel = RiskSafe
		riskScore = 5

	case "nexus_recall":
		cap = CapMemoryRead
		riskLevel = RiskSafe
		riskScore = 5

	case "personal_memory_save":
		cap = CapMemoryWrite
		riskLevel = RiskSafe
		riskScore = 5

	case "personal_memory_recall":
		cap = CapMemoryRead
		riskLevel = RiskSafe
		riskScore = 5

	case "task_set_checklist":
		cap = CapTaskCreate
		riskLevel = RiskSafe
		riskScore = 5

	case "task_update_checklist":
		cap = CapTaskCreate
		riskLevel = RiskSafe
		riskScore = 5

	case "agent_handoff":
		cap = CapCodexHandoff
		riskLevel = RiskModerate
		riskScore = 35

	case "web_browser":
		cap = CapBrowserOpen
		riskLevel = RiskModerate
		riskScore = 30

	case "weather_lookup":
		cap = CapWeatherRead
		riskLevel = RiskLow
		riskScore = 10

	case "market_lookup":
		cap = CapMarketRead
		riskLevel = RiskLow
		riskScore = 10

	case "mail_list_messages":
		cap = CapSecretUse
		riskLevel = RiskModerate
		riskScore = 40

	case "mail_send_message":
		cap = CapMessagingSend
		riskLevel = RiskCritical
		riskScore = 85

	case "sphere_write_document":
		cap = CapSphereWrite
		riskLevel = RiskLow
		riskScore = 15

	case "intelligence_status", "global_watch_nexus_context":
		cap = CapIntelRead
		riskLevel = RiskLow
		riskScore = 10

	case "global_watch_toggle_layer":
		cap = CapIntelWrite
		riskLevel = RiskModerate
		riskScore = 25

	case "global_watch_focus_region", "global_watch_time_window", "global_watch_open_report":
		cap = CapIntelWrite
		riskLevel = RiskLow
		riskScore = 10

	case "navigate_ui":
		cap = CapUINavigation
		riskLevel = RiskSafe
		riskScore = 2

	case "global_watch_schedule_briefing":
		cap = CapIntelWrite
		riskLevel = RiskModerate
		riskScore = 40

	case "global_watch_focus":
		cap = CapIntelWrite
		riskLevel = RiskModerate
		riskScore = 30

	case "osint_timeline_generate", "osint_report_generate", "intelligence_identity_status",
		"intelligence_sync_personal", "intelligence_personal_impact", "intelligence_source_health":
		cap = CapIntelRead
		riskLevel = RiskLow
		riskScore = 10

	case "intelligence_connector_fetch":
		cap = CapIntelSources
		riskLevel = RiskLow
		riskScore = 20

	case "intelligence_propose_observation", "global_watch_observe":
		cap = CapIntelWrite
		riskLevel = RiskModerate
		riskScore = 35

	case "intelligence_create_case", "intelligence_add_entity", "intelligence_link_entities",
		"osint_case_create", "osint_entity_propose", "osint_relation_propose", "osint_evidence_capture",
		"intelligence_set_assessment_status":
		cap = CapIntelWrite
		riskLevel = RiskModerate
		riskScore = 45

	case "intelligence_request_reid", "intelligence_approve_reid":
		cap = CapIntelWrite
		riskLevel = RiskHigh
		riskScore = 75

	case "osint_add_custom_feed", "osint_set_briefing_prompt":
		cap = CapIntelSources
		riskLevel = RiskModerate
		riskScore = 50

	case "media_control":
		cap = CapMediaControl
		riskLevel = RiskLow
		riskScore = 10

	case "youtube_control":
		cap = CapBrowserCtl
		riskLevel = RiskLow
		riskScore = 15

	case "vision_context":
		cap = CapScreenRead
		riskLevel = RiskModerate
		riskScore = 35

	case "gui_control":
		cap = CapGuiMoveMouse // Default
		riskLevel = RiskLow
		riskScore = 15

		// Extract action to refine capability & risk
		var guiArgs struct {
			Action string `json:"action"`
			Keys   string `json:"keys"`
		}
		if err := json.Unmarshal([]byte(args), &guiArgs); err == nil {
			switch guiArgs.Action {
			case "move", "position":
				cap = CapGuiMoveMouse
				riskLevel = RiskLow
				riskScore = 10
			case "click":
				cap = CapGuiClick
				riskLevel = RiskModerate
				riskScore = 30
			case "type":
				cap = CapGuiType
				riskLevel = RiskHigh
				riskScore = 65
			case "press":
				cap = CapGuiPressKey
				riskLevel = RiskHigh
				riskScore = 65

				// Forbidden keystroke checking (Alt+F4, Win+R, etc.)
				lowerKeys := strings.ToLower(guiArgs.Keys)
				if strings.Contains(lowerKeys, "%{f4}") || strings.Contains(lowerKeys, "alt+f4") ||
					strings.Contains(lowerKeys, "win+r") || strings.Contains(lowerKeys, "^esc") ||
					(strings.Contains(lowerKeys, "{f4}") && strings.Contains(lowerKeys, "%")) {
					threats = append(threats, "FORBIDDEN_SHORTCUT: Systemkritischer Shortcut blockiert.")
					riskScore = 100
					riskLevel = RiskForbidden
				}
			}
		}
	case "gui_window_control":
		cap = CapGuiMoveMouse
		riskLevel = RiskLow
		riskScore = 15

		var winArgs struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal([]byte(args), &winArgs); err == nil {
			switch winArgs.Action {
			case "list":
				cap = CapGuiMoveMouse
				riskLevel = RiskLow
				riskScore = 10
			case "focus":
				cap = CapGuiClick
				riskLevel = RiskModerate
				riskScore = 30
			case "move":
				cap = CapGuiClick
				riskLevel = RiskModerate
				riskScore = 40
			case "close":
				cap = CapFsDelete
				riskLevel = RiskHigh
				riskScore = 75
			}
		}
	}

	if riskScore > 100 {
		riskScore = 100
	}

	isSafe := len(threats) == 0 && riskLevel != RiskForbidden && riskLevel != RiskCritical

	return ThreatReport{
		IsSafe:     isSafe,
		Threats:    threats,
		RiskScore:  riskScore,
		RiskLevel:  riskLevel,
		Capability: cap,
	}
}

// PolicyEngine orchestrates rules, leases and scans
type PolicyEngine struct {
	guard  *SecurityGuard
	leases *LeaseManager
	audit  *AuditLogger
}

func NewPolicyEngine(guard *SecurityGuard, leases *LeaseManager, audit *AuditLogger) *PolicyEngine {
	return &PolicyEngine{
		guard:  guard,
		leases: leases,
		audit:  audit,
	}
}

func (pe *PolicyEngine) Evaluate(toolName string, args string, hasOverride bool) (bool, string, ThreatReport) {
	report := pe.guard.Scan(toolName, args)
	recordDecision := func(decision, reason, leaseID string) bool {
		if _, err := pe.audit.Log("aethel", toolName, string(report.Capability), report.RiskLevel, leaseID, decision, reason, args); err != nil {
			report.IsSafe = false
			report.RiskLevel = RiskForbidden
			report.RiskScore = 100
			report.Threats = append(report.Threats, "AUDIT_WRITE_FAILED: execution locked because the decision could not be recorded.")
			return false
		}
		return true
	}
	if pe.audit.IsTampered() {
		report.IsSafe = false
		report.RiskLevel = RiskForbidden
		report.RiskScore = 100
		report.Threats = append(report.Threats, "AUDIT_CHAIN_TAMPERED: execution locked pending operator review.")
		return false, "blocked", report
	}

	// 1. HARD BLOCK ON FORBIDDEN ACTION (Never overrideable)
	if report.RiskLevel == RiskForbidden {
		recordDecision("blocked", "Aktion ist permanent verboten.", "")
		return false, "blocked", report
	}

	// 2. CHECK ACTIVE LEASES FOR CAPABILITY
	hasLease, leaseID := pe.leases.CheckLease(report.Capability, args, toolName)
	if hasLease {
		if !recordDecision("allowed", "Berechtigung durch aktiven Lease erteilt.", leaseID) {
			return false, "blocked", report
		}
		return true, leaseID, report
	}

	// 3. SAFE & LOW RISK ACTIONS (EXECUTE AUTOMATICALLY)
	if report.RiskLevel == RiskSafe || report.RiskLevel == RiskLow {
		if !recordDecision("allowed", "Safe/Low-Risk Aktion automatisch freigegeben.", "") {
			return false, "blocked", report
		}
		return true, "", report
	}

	// 4. USER OVERRIDE IN EFFECT (ONE-TIME APPROVAL)
	if hasOverride {
		// Critical scanner findings are never executable.  An approval is a grant of
		// authority, not permission to bypass the command-injection boundary.
		if report.RiskLevel == RiskCritical && len(report.Threats) > 0 {
			recordDecision("blocked", "Kritischer Sicherheitsbefund ist nicht überschreibbar.", "")
			return false, "blocked", report
		}
		// Low/Moderate/High risks overridden
		if !recordDecision("override", "Einmalige Freigabe durch Operator.", "") {
			return false, "blocked", report
		}
		return true, "", report
	}

	// 5. REQUIRE APPROVAL
	if !recordDecision("requested_approval", "Zustimmung vom Operator ausstehend.", "") {
		return false, "blocked", report
	}
	return false, "needs_approval", report
}

// Relocated from config.go
type MountAccess string

const (
	MountRead  MountAccess = "read"
	MountWrite MountAccess = "write"
)

type MountGrant struct {
	Path      string      `json:"path"`
	Access    MountAccess `json:"access"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// Relocated path validation functions
func ValidateWritePath(pathStr string) (string, error) {
	return ValidatePathForAccess(pathStr, MountWrite)
}

// OpenAuthorizedFile resolves an operator-authorized file path once and then
// performs subsequent operations through an os.Root handle. This prevents a
// final-component symlink swap from escaping the validated parent directory.
// The caller owns the returned root handle.
func OpenAuthorizedFile(pathStr string, access MountAccess) (*os.Root, string, string, error) {
	resolved, err := ValidatePathForAccess(pathStr, access)
	if err != nil {
		return nil, "", "", err
	}
	name := filepath.Base(resolved)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return nil, "", "", errors.New("authorized file path does not name a file")
	}
	if access == MountWrite {
		if err := os.MkdirAll(filepath.Dir(resolved), 0700); err != nil {
			return nil, "", "", err
		}
	}
	root, err := os.OpenRoot(filepath.Dir(resolved))
	if err != nil {
		return nil, "", "", err
	}
	return root, name, resolved, nil
}

func ValidatePathForAccess(pathStr string, access MountAccess) (string, error) {
	err := os.MkdirAll(WorkspaceDir, 0700)
	if err != nil {
		return "", err
	}

	absWorkspace, err := CanonicalDir(WorkspaceDir)
	if err != nil {
		return "", err
	}

	var absTarget string
	if filepath.IsAbs(pathStr) || strings.HasPrefix(pathStr, "/") || strings.HasPrefix(pathStr, "\\") {
		absTarget, err = filepath.Abs(pathStr)
	} else {
		absTarget, err = filepath.Abs(filepath.Join(WorkspaceDir, pathStr))
	}
	if err != nil {
		return "", err
	}

	resolvedTarget := absTarget
	if existing, err := CanonicalTarget(absTarget); err == nil {
		resolvedTarget = existing
	} else {
		return "", err
	}

	if IsPathInside(absWorkspace, resolvedTarget) {
		return resolvedTarget, nil
	}

	if state != nil && state.MountAllows != nil && state.MountAllows(resolvedTarget, access) {
		return resolvedTarget, nil
	}

	if runtime.GOOS == "windows" {
		return "", errors.New("SECURITY VIOLATION: path escaped the configured Windows workspace jail")
	}

	return "", errors.New("SECURITY VIOLATION: Path is not inside the workspace or any mounted directory")
}

const WorkspaceDir = "./vgt_workspace"

func CanonicalDir(pathStr string) (string, error) {
	abs, err := filepath.Abs(pathStr)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", err
	}
	return resolved, nil
}

func IsPathInside(baseDir, targetPath string) bool {
	if runtime.GOOS == "windows" {
		baseDir = strings.ToLower(baseDir)
		targetPath = strings.ToLower(targetPath)
	}
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func CanonicalTarget(pathStr string) (string, error) {
	abs, err := filepath.Abs(pathStr)
	if err != nil {
		return "", err
	}
	current := abs
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			parts := append([]string{resolved}, tail...)
			return filepath.Join(parts...), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return abs, nil
		}
		tail = append([]string{filepath.Base(current)}, tail...)
		current = parent
	}
}

// SafeResourceIDPattern validates run/session/resource IDs for path segments.
var SafeResourceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,79}$`)
var safeResourceIDPattern = SafeResourceIDPattern

func ValidateResourceID(id string) error {
	if !safeResourceIDPattern.MatchString(id) {
		return fmt.Errorf("invalid resource id")
	}
	return nil
}

func GetPowerShellPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	}
	return ""
}
