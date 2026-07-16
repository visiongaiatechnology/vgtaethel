// STATUS: DIAMANT VGT SUPREME
// Validated, local-only operator preferences for Global Watch.

const STORAGE_KEY = 'aethel.globalWatch.preferences.v3';
const DEFAULTS = Object.freeze({ renderQuality: 'balanced', idleRotation: true, hazardAnimation: true, hazardFPS: 6, autoRefreshSeconds: 60, feedLimit: 200, clusterMode: 'balanced' });
const ALLOWED = Object.freeze({
    renderQuality: new Set(['performance', 'balanced', 'ultra']),
    hazardFPS: new Set([2, 4, 6, 8]),
    autoRefreshSeconds: new Set([0, 30, 60, 120, 300]),
    feedLimit: new Set([100, 200, 500, 1000]),
    clusterMode: new Set(['compact', 'balanced', 'precise']),
});

function sanitize(candidate) {
    const source = candidate && typeof candidate === 'object' ? candidate : {};
    return Object.freeze({
        renderQuality: ALLOWED.renderQuality.has(source.renderQuality) ? source.renderQuality : DEFAULTS.renderQuality,
        idleRotation: typeof source.idleRotation === 'boolean' ? source.idleRotation : DEFAULTS.idleRotation,
        hazardAnimation: typeof source.hazardAnimation === 'boolean' ? source.hazardAnimation : DEFAULTS.hazardAnimation,
        hazardFPS: ALLOWED.hazardFPS.has(Number(source.hazardFPS)) ? Number(source.hazardFPS) : DEFAULTS.hazardFPS,
        autoRefreshSeconds: ALLOWED.autoRefreshSeconds.has(Number(source.autoRefreshSeconds)) ? Number(source.autoRefreshSeconds) : DEFAULTS.autoRefreshSeconds,
        feedLimit: ALLOWED.feedLimit.has(Number(source.feedLimit)) ? Number(source.feedLimit) : DEFAULTS.feedLimit,
        clusterMode: ALLOWED.clusterMode.has(source.clusterMode) ? source.clusterMode : DEFAULTS.clusterMode,
    });
}

export function loadGlobalWatchPreferences() {
    try { return sanitize(JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')); } catch (_) { return DEFAULTS; }
}

export function saveGlobalWatchPreferences(candidate) {
    const preferences = sanitize(candidate);
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences)); } catch (_) { /* runtime state remains valid */ }
    return preferences;
}

export function globalWatchDefaults() { return DEFAULTS; }
