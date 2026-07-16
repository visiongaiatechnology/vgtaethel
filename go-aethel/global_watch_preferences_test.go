package main

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

//go:embed frontend/modules/global_watch_preferences.js
var globalWatchPreferencesJS []byte

func TestGlobalWatchPreferencesValidatePersistedOperatorControls(t *testing.T) {
	vm := goja.New()
	if _, err := vm.RunString(`
		var storedPreference = '';
		var localStorage = {
			getItem: function() { return storedPreference; },
			setItem: function(_, value) { storedPreference = value; }
		};
	`); err != nil {
		t.Fatalf("install local storage mock: %v", err)
	}
	source := strings.ReplaceAll(string(globalWatchPreferencesJS), "export ", "")
	if _, err := vm.RunString(source); err != nil {
		t.Fatalf("load preferences module: %v", err)
	}

	save, ok := goja.AssertFunction(vm.Get("saveGlobalWatchPreferences"))
	if !ok {
		t.Fatal("saveGlobalWatchPreferences is not callable")
	}
	value, err := save(goja.Undefined(), vm.ToValue(map[string]any{
		"renderQuality":      "invalid",
		"hazardAnimation":    false,
		"hazardFPS":          99,
		"autoRefreshSeconds": 300,
		"feedLimit":          1000,
		"clusterMode":        "precise",
	}))
	if err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	preferences := value.ToObject(vm)
	if got := preferences.Get("renderQuality").String(); got != "balanced" {
		t.Fatalf("invalid render quality must fall back, got %q", got)
	}
	if got := preferences.Get("hazardFPS").ToInteger(); got != 6 {
		t.Fatalf("invalid hazard FPS must fall back, got %d", got)
	}
	if got := preferences.Get("autoRefreshSeconds").ToInteger(); got != 300 {
		t.Fatalf("valid refresh cadence lost, got %d", got)
	}
	if got := preferences.Get("feedLimit").ToInteger(); got != 1000 {
		t.Fatalf("valid feed limit lost, got %d", got)
	}
	if preferences.Get("hazardAnimation").ToBoolean() {
		t.Fatal("explicitly disabled hazard animation was not preserved")
	}
}
