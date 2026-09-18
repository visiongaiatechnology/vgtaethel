// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/browser/browser_session.go
// Purpose: Persistent Multi-Tab Browser Session State and AI Interaction Broker

package browser

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-aethel/sphere"
)

type SessionManager struct {
	mu     sync.RWMutex
	tabs   map[string]*sphere.BrowserTabState
	active string
}

func NewSessionManager() *SessionManager {
	initialTab := &sphere.BrowserTabState{
		TabID:         "tab-1",
		URL:           "https://www.google.de",
		Title:         "Google Suche",
		IsActive:      true,
		IsLoading:     false,
		AIControlling: false,
		LastUpdated:   time.Now().UTC(),
	}

	sm := &SessionManager{
		tabs: map[string]*sphere.BrowserTabState{
			"tab-1": initialTab,
		},
		active: "tab-1",
	}
	return sm
}

// GetTabs returns all open tabs.
func (sm *SessionManager) GetTabs() []*sphere.BrowserTabState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*sphere.BrowserTabState, 0, len(sm.tabs))
	for _, t := range sm.tabs {
		copyTab := *t
		result = append(result, &copyTab)
	}
	return result
}

// GetActiveTab returns the current active tab.
func (sm *SessionManager) GetActiveTab() (*sphere.BrowserTabState, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	tab, exists := sm.tabs[sm.active]
	if !exists {
		return nil, false
	}
	copyTab := *tab
	return &copyTab, true
}

// NewTab opens a new tab with a given URL.
func (sm *SessionManager) NewTab(url string, title string) *sphere.BrowserTabState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	tabID := fmt.Sprintf("tab-%d", time.Now().UnixNano()%100000)
	if url == "" {
		url = "https://www.google.de"
	}
	if title == "" {
		title = "Neuer Tab"
	}

	// Deactivate other tabs
	for _, t := range sm.tabs {
		t.IsActive = false
	}

	tab := &sphere.BrowserTabState{
		TabID:         tabID,
		URL:           url,
		Title:         title,
		IsActive:      true,
		IsLoading:     false,
		AIControlling: false,
		LastUpdated:   time.Now().UTC(),
	}

	sm.tabs[tabID] = tab
	sm.active = tabID
	return tab
}

// SwitchTab sets the active tab.
func (sm *SessionManager) SwitchTab(tabID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.tabs[tabID]; !exists {
		return fmt.Errorf("tab %s not found", tabID)
	}
	for id, t := range sm.tabs {
		t.IsActive = (id == tabID)
	}
	sm.active = tabID
	return nil
}

// CloseTab closes a tab and activates another if necessary.
func (sm *SessionManager) CloseTab(tabID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.tabs) <= 1 {
		return errors.New("cannot close last remaining browser tab")
	}

	delete(sm.tabs, tabID)
	if sm.active == tabID {
		for id, t := range sm.tabs {
			t.IsActive = true
			sm.active = id
			break
		}
	}
	return nil
}

// UpdateTabState updates the URL, title, or AI control state of a tab.
func (sm *SessionManager) UpdateTabState(tabID string, url string, title string, aiControlling bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	tab, exists := sm.tabs[tabID]
	if !exists {
		return fmt.Errorf("tab %s not found", tabID)
	}
	if strings.TrimSpace(url) != "" {
		tab.URL = url
	}
	if strings.TrimSpace(title) != "" {
		tab.Title = title
	}
	tab.AIControlling = aiControlling
	tab.LastUpdated = time.Now().UTC()
	return nil
}
