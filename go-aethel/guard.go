package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	CapBrowserOpen   Capability = "browser.open_url"
	CapBrowserRead   Capability = "browser.read_page"
	CapMemoryRead    Capability = "memory.read"
	CapMemoryWrite   Capability = "memory.write"
	CapSecretUse     Capability = "secret.use"
	CapTaskCreate    Capability = "task.create"
	CapTaskSchedule  Capability = "task.schedule"
	CapMessagingSend Capability = "messaging.send"
	CapCodexHandoff  Capability = "codex.delegate_task"
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
	ApprovedBy          string     `json:"approved_by"`      // e.g. "user"
	ApprovalMethod      string     `json:"approval_method"`  // e.g. "ui", "voice"
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

	data, err := os.ReadFile(lm.cachePath)
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

func (lm *LeaseManager) saveToDisk() error {
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

	_ = os.MkdirAll(filepath.Dir(lm.cachePath), 0755)
	return os.WriteFile(lm.cachePath, data, 0644)
}

func (lm *LeaseManager) AddLease(l PermissionLease) error {
	lm.mu.Lock()
	lm.leases[l.LeaseID] = l
	lm.mu.Unlock()
	return lm.saveToDisk()
}

func (lm *LeaseManager) RevokeLease(leaseID string) bool {
	lm.mu.Lock()
	_, exists := lm.leases[leaseID]
	if exists {
		delete(lm.leases, leaseID)
	}
	lm.mu.Unlock()
	if exists {
		_ = lm.saveToDisk()
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
		Timestamp    string `json:"timestamp"`
		Actor        string `json:"actor"`
		Operation    string `json:"operation"`
		Target       string `json:"target"`
		Risk         string `json:"risk"`
		LeaseID      string `json:"lease_id,omitempty"`
		Decision     string `json:"decision"`
		Reason       string `json:"reason"`
		InputSummary string `json:"input_summary"`
		PrevHash     string `json:"prev_hash"`
	}
	p := HashPayload{
		Timestamp:    e.Timestamp.Format(time.RFC3339),
		Actor:        e.Actor,
		Operation:    e.Operation,
		Target:       e.Target,
		Risk:         string(e.Risk),
		LeaseID:      e.LeaseID,
		Decision:     e.Decision,
		Reason:       e.Reason,
		InputSummary: e.InputSummary,
		PrevHash:     prevHash,
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

	data, err := os.ReadFile(al.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var entries []AuditEntry
	if err := json.Unmarshal(data, &entries); err != nil {
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

	al.lastHash = currentPrevHash
	return nil
}

func (al *AuditLogger) Log(actor, operation, target string, risk RiskLevel, leaseID, decision, reason, input string) (string, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	var entries []AuditEntry
	data, err := os.ReadFile(al.filePath)
	if err == nil {
		_ = json.Unmarshal(data, &entries)
	}

	entry := AuditEntry{
		Timestamp:    time.Now(),
		Actor:        actor,
		Operation:    operation,
		Target:       target,
		Risk:         risk,
		LeaseID:      leaseID,
		Decision:     decision,
		Reason:       reason,
		InputSummary: input,
		PrevHash:     al.lastHash,
	}

	entry.Hash = entry.computeHash(al.lastHash)
	al.lastHash = entry.Hash

	entries = append(entries, entry)
	marshaled, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", err
	}

	_ = os.MkdirAll(filepath.Dir(al.filePath), 0755)
	err = os.WriteFile(al.filePath, marshaled, 0644)
	return entry.Hash, err
}

func (al *AuditLogger) GetLogs() []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	var entries []AuditEntry
	data, err := os.ReadFile(al.filePath)
	if err == nil {
		_ = json.Unmarshal(data, &entries)
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	return entries
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

	case "fs_write_file":
		cap = CapFsWrite
		riskLevel = RiskModerate
		riskScore = 40

		lowerArgs := strings.ToLower(args)
		if strings.Contains(lowerArgs, ".sh") || strings.Contains(lowerArgs, ".exe") || strings.Contains(lowerArgs, ".bat") || strings.Contains(lowerArgs, ".cmd") {
			threats = append(threats, "EXECUTABLE_WRITE_ATTEMPT: Erstellung ausführbarer Dateien ist restricted.")
			riskScore = 85
			riskLevel = RiskCritical
		}

	case "fs_mount_folder":
		cap = CapFsMount
		riskLevel = RiskModerate
		riskScore = 50

	case "nexus_save":
		cap = CapMemoryWrite
		riskLevel = RiskSafe
		riskScore = 5

	case "nexus_recall":
		cap = CapMemoryRead
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

	// Determine actor context
	actor := "aethel"

	// 1. HARD BLOCK ON FORBIDDEN ACTION (Never overrideable)
	if report.RiskLevel == RiskForbidden {
		_, _ = pe.audit.Log(actor, toolName, string(report.Capability), report.RiskLevel, "", "blocked", "Aktion ist permanent verboten.", args)
		return false, "blocked", report
	}

	// 2. CHECK ACTIVE LEASES FOR CAPABILITY
	hasLease, leaseID := pe.leases.CheckLease(report.Capability, args, toolName)
	if hasLease {
		_, _ = pe.audit.Log(actor, toolName, string(report.Capability), report.RiskLevel, leaseID, "allowed", "Berechtigung durch aktiven Lease erteilt.", args)
		return true, leaseID, report
	}

	// 3. SAFE & LOW RISK ACTIONS (EXECUTE AUTOMATICALLY)
	if report.RiskLevel == RiskSafe || report.RiskLevel == RiskLow {
		_, _ = pe.audit.Log(actor, toolName, string(report.Capability), report.RiskLevel, "", "allowed", "Safe/Low-Risk Aktion automatisch freigegeben.", args)
		return true, "", report
	}

	// 4. USER OVERRIDE IN EFFECT (ONE-TIME APPROVAL)
	if hasOverride {
		// Verify if the override was allowed (Critical risks require explicit user decree)
		if report.RiskLevel == RiskCritical && len(report.Threats) > 0 {
			// User has acknowledged the threat warning
			_, _ = pe.audit.Log(actor, toolName, string(report.Capability), report.RiskLevel, "", "override", "Bedrohung explizit durch Operator ignoriert.", args)
			return true, "", report
		}
		// Low/Moderate/High risks overridden
		_, _ = pe.audit.Log(actor, toolName, string(report.Capability), report.RiskLevel, "", "override", "Einmalige Freigabe durch Operator.", args)
		return true, "", report
	}

	// 5. REQUIRE APPROVAL
	_, _ = pe.audit.Log(actor, toolName, string(report.Capability), report.RiskLevel, "", "requested_approval", "Zustimmung vom Operator ausstehend.", args)
	return false, "needs_approval", report
}
