package agent

import (
	"encoding/json"
	"strings"
)

// requestsUINavigation is intentionally stricter than keyword matching. A view
// name is not a navigation command unless the operator also expresses a view
// transition. This prevents intelligence questions from being consumed by
// navigate_ui merely because they mention Global Watch.
func requestsUINavigation(objective string) bool {
	text := normalizeAgentObjective(objective)
	if text == "" || !containsAny(text,
		"global watch", "live-globus", "live globus", "sphere", "sphaere",
		"personal core", "run center", "agent tracker", "settings", "einstellungen",
		"chat terminal", "neural core", "persona registry") {
		return false
	}
	return containsAny(text,
		"oeffne", "open ", "wechsle", "wechsel zu", "switch ", "gehe zu", "geh zu",
		"go to", "navigiere", "navigate", "bring mich", "zeige mir den tab",
		"zeige die ansicht", "show the view", "show the tab")
}

func objectiveRequestsGlobalWatch(objective string) bool {
	text := normalizeAgentObjective(objective)
	if text == "" {
		return false
	}
	if strings.Contains(text, "[global watch operator]") {
		return true
	}
	anchored := containsAny(text,
		"global watch", "weltlage", "nachrichtenlage", "lagebriefing", "osint",
		"geopolit", "aktuelle news", "weltgeschehen", "weltweit", "globale lage")
	if !anchored {
		return false
	}
	if !requestsUINavigation(text) {
		return true
	}
	return containsAny(text,
		"news", "nachricht", "neueste", "aktuell", "info", "briefing", "analys",
		"bewert", "risiko", "alert", "warnung", "was ist", "wie ist", "nutze")
}

func isGlobalWatchFollowUp(objective string) bool {
	text := normalizeAgentObjective(objective)
	if text == "" || len([]rune(text)) > 600 || requestsUINavigation(text) {
		return false
	}
	return strings.Contains(text, "?") || containsAny(text,
		"also", "und jetzt", "derzeit", "aktuell", "gruene bereich", "gruenen bereich",
		"wie sieht", "was bedeutet", "was davon", "schaetzt du", "einschaetzung",
		"ist alles", "gibt es", "welche davon", "mehr dazu")
}

func requestHasGlobalWatchContext(request ChatAgentStartRequest) bool {
	if request.ProfileID == "global_watch_operator" || objectiveRequestsGlobalWatch(request.Objective) {
		return true
	}
	if !isGlobalWatchFollowUp(request.Objective) {
		return false
	}
	remaining := 10
	for index := len(request.Messages) - 1; index >= 0 && remaining > 0; index-- {
		var message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if json.Unmarshal(request.Messages[index], &message) != nil || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(message.Role), "user") && strings.TrimSpace(message.Content) == strings.TrimSpace(request.Objective) {
			continue
		}
		remaining--
		if objectiveRequestsGlobalWatch(message.Content) || strings.Contains(normalizeAgentObjective(message.Content), "global_watch_nexus_context") {
			return true
		}
	}
	return false
}

func objectiveRequestsMailSend(objective string) bool {
	text := normalizeAgentObjective(objective)
	mailMentioned := containsAny(text, "mail", "e-mail", "email", "nachricht")
	return mailMentioned && containsAny(text, "sende", "senden", "send", "schicke", "verschicke", "abschicken", "versenden")
}

func objectiveRequestsMailRead(objective string) bool {
	text := normalizeAgentObjective(objective)
	if objectiveRequestsMailSend(text) {
		return false
	}
	if containsAny(text, "postfach", "inbox", "ungelesen") {
		return true
	}
	mailMentioned := containsAny(text, "e-mail", "email", "mail")
	readRequested := containsAny(text,
		"lies", "lese", "lesen", "pruefe", "prüfe", "check", "zeige",
		"neueste", "neue mail", "neue e-mail", "new mail", "read ")
	return mailMentioned && readRequested
}
