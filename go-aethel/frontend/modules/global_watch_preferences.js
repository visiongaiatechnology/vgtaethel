// STATUS: DIAMANT VGT SUPREME
// Validated, local-only operator preferences for Global Watch & 3D Geospatial Engine.

const STORAGE_KEY = 'aethel.globalWatch.preferences.v7';
const LEGACY_STORAGE_KEY = 'aethel.globalWatch.preferences.v6';
const DEFAULTS = Object.freeze({
    renderQuality: 'ultra',
    idleRotation: true,
    hazardAnimation: true,
    hazardFPS: 6,
    autoRefreshSeconds: 60,
    feedLimit: 200,
    clusterMode: 'balanced',
    max3DEntities: 10000,
    enableAtmosphere: true,
    enableTrails: true,
    enableConflictTensions: true,
    targetFPS: 60,
    dynamicQuality: true,
    diagnosticsHUD: false,
    postProcessing: true,
    rendererMode: 'enhanced',
    terrainQuality: 'ultra',
    tilesQuality: 'ultra',
    maximumScreenSpaceError: 1.25,
    dynamicScreenSpaceError: true,
    maximumNetworkRequests: 18,
    entityDrawDistance: 20000000,
    labelDrawDistance: 500000,
    modelDrawDistance: 50000,
    trailLength: 16,
    trailDensity: 2,
    aircraftUpdateSeconds: 12,
    satelliteUpdateSeconds: 10,
    aisUpdateSeconds: 60,
    osintUpdateSeconds: 30,
    geoClustering: 'adaptive',
    postProcessingLevel: 'low',
    antiAliasing: true,
    fog: true,
    shadows: false,
    lighting: true,
    highResolutionTextures: false,
    sensorEffects: true,
    hudDensity: 'standard',
    automaticLOD: true,
    riskOverlayMinAltitude: 3000000,
    satelliteMinAltitude: 500000,
    aircraftMaxAltitude: 7000000,
    vesselMaxAltitude: 2200000,
    cameraMaxAltitude: 4500000,
    hazardMaxAltitude: 12000000,
    cityMaxAltitude: 2500000,
    relationMinAltitude: 120000,
    relationMaxAltitude: 16000000,
});

const ALLOWED = Object.freeze({
    renderQuality: new Set(['power_saver', 'performance', 'balanced', 'high', 'ultra', 'custom']),
    hazardFPS: new Set([2, 4, 6, 8]),
    autoRefreshSeconds: new Set([0, 30, 60, 120, 300]),
    feedLimit: new Set([100, 200, 500, 1000]),
    clusterMode: new Set(['compact', 'balanced', 'precise']),
    max3DEntities: new Set([250, 500, 1000, 2500, 5000, 10000]),
    targetFPS: new Set([24, 30, 45, 60]),
    rendererMode: new Set(['enhanced']),
    terrainQuality: new Set(['off', 'low', 'medium', 'high', 'ultra']),
    tilesQuality: new Set(['low', 'medium', 'high', 'ultra']),
    geoClustering: new Set(['off', 'light', 'adaptive', 'aggressive']),
    postProcessingLevel: new Set(['off', 'low', 'full']),
    hudDensity: new Set(['minimal', 'standard', 'dense']),
});

function clamp(value, minimum, maximum, fallback) {
    const numeric = Number(value);
    return Number.isFinite(numeric) ? Math.min(maximum, Math.max(minimum, numeric)) : fallback;
}

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
        max3DEntities: ALLOWED.max3DEntities.has(Number(source.max3DEntities)) ? Number(source.max3DEntities) : DEFAULTS.max3DEntities,
        enableAtmosphere: typeof source.enableAtmosphere === 'boolean' ? source.enableAtmosphere : DEFAULTS.enableAtmosphere,
        enableTrails: typeof source.enableTrails === 'boolean' ? source.enableTrails : DEFAULTS.enableTrails,
        enableConflictTensions: typeof source.enableConflictTensions === 'boolean' ? source.enableConflictTensions : DEFAULTS.enableConflictTensions,
        targetFPS: ALLOWED.targetFPS.has(Number(source.targetFPS)) ? Number(source.targetFPS) : DEFAULTS.targetFPS,
        dynamicQuality: typeof source.dynamicQuality === 'boolean' ? source.dynamicQuality : DEFAULTS.dynamicQuality,
        diagnosticsHUD: typeof source.diagnosticsHUD === 'boolean' ? source.diagnosticsHUD : DEFAULTS.diagnosticsHUD,
        postProcessing: typeof source.postProcessing === 'boolean' ? source.postProcessing : DEFAULTS.postProcessing,
        rendererMode: ALLOWED.rendererMode.has(source.rendererMode) ? source.rendererMode : DEFAULTS.rendererMode,
        terrainQuality: ALLOWED.terrainQuality.has(source.terrainQuality) ? source.terrainQuality : DEFAULTS.terrainQuality,
        tilesQuality: ALLOWED.tilesQuality.has(source.tilesQuality) ? source.tilesQuality : DEFAULTS.tilesQuality,
        maximumScreenSpaceError: clamp(source.maximumScreenSpaceError, 1, 16, DEFAULTS.maximumScreenSpaceError),
        dynamicScreenSpaceError: typeof source.dynamicScreenSpaceError === 'boolean' ? source.dynamicScreenSpaceError : DEFAULTS.dynamicScreenSpaceError,
        maximumNetworkRequests: Math.round(clamp(source.maximumNetworkRequests, 4, 64, DEFAULTS.maximumNetworkRequests)),
        entityDrawDistance: clamp(source.entityDrawDistance, 1000000, 50000000, DEFAULTS.entityDrawDistance),
        labelDrawDistance: clamp(source.labelDrawDistance, 50000, 5000000, DEFAULTS.labelDrawDistance),
        modelDrawDistance: clamp(source.modelDrawDistance, 5000, 500000, DEFAULTS.modelDrawDistance),
        trailLength: Math.round(clamp(source.trailLength, 0, 30, DEFAULTS.trailLength)),
        trailDensity: Math.round(clamp(source.trailDensity, 1, 8, DEFAULTS.trailDensity)),
        aircraftUpdateSeconds: Math.round(clamp(source.aircraftUpdateSeconds, 5, 120, DEFAULTS.aircraftUpdateSeconds)),
        satelliteUpdateSeconds: Math.round(clamp(source.satelliteUpdateSeconds, 5, 300, DEFAULTS.satelliteUpdateSeconds)),
        aisUpdateSeconds: Math.round(clamp(source.aisUpdateSeconds, 15, 600, DEFAULTS.aisUpdateSeconds)),
        osintUpdateSeconds: Math.round(clamp(source.osintUpdateSeconds, 15, 600, DEFAULTS.osintUpdateSeconds)),
        geoClustering: ALLOWED.geoClustering.has(source.geoClustering) ? source.geoClustering : DEFAULTS.geoClustering,
        postProcessingLevel: ALLOWED.postProcessingLevel.has(source.postProcessingLevel) ? source.postProcessingLevel : DEFAULTS.postProcessingLevel,
        antiAliasing: typeof source.antiAliasing === 'boolean' ? source.antiAliasing : DEFAULTS.antiAliasing,
        fog: typeof source.fog === 'boolean' ? source.fog : DEFAULTS.fog,
        shadows: typeof source.shadows === 'boolean' ? source.shadows : DEFAULTS.shadows,
        lighting: typeof source.lighting === 'boolean' ? source.lighting : DEFAULTS.lighting,
        highResolutionTextures: typeof source.highResolutionTextures === 'boolean' ? source.highResolutionTextures : DEFAULTS.highResolutionTextures,
        sensorEffects: typeof source.sensorEffects === 'boolean' ? source.sensorEffects : DEFAULTS.sensorEffects,
        hudDensity: ALLOWED.hudDensity.has(source.hudDensity) ? source.hudDensity : DEFAULTS.hudDensity,
        automaticLOD: typeof source.automaticLOD === 'boolean' ? source.automaticLOD : DEFAULTS.automaticLOD,
        riskOverlayMinAltitude: Math.round(clamp(source.riskOverlayMinAltitude, 100000, 20000000, DEFAULTS.riskOverlayMinAltitude)),
        satelliteMinAltitude: Math.round(clamp(source.satelliteMinAltitude, 100000, 20000000, DEFAULTS.satelliteMinAltitude)),
        aircraftMaxAltitude: Math.round(clamp(source.aircraftMaxAltitude, 50000, 10000000, DEFAULTS.aircraftMaxAltitude)),
        vesselMaxAltitude: Math.round(clamp(source.vesselMaxAltitude, 50000, 10000000, DEFAULTS.vesselMaxAltitude)),
        cameraMaxAltitude: Math.round(clamp(source.cameraMaxAltitude, 5000, 5000000, DEFAULTS.cameraMaxAltitude)),
        hazardMaxAltitude: Math.round(clamp(source.hazardMaxAltitude, 100000, 20000000, DEFAULTS.hazardMaxAltitude)),
        cityMaxAltitude: Math.round(clamp(source.cityMaxAltitude, 50000, 10000000, DEFAULTS.cityMaxAltitude)),
        relationMinAltitude: Math.round(clamp(source.relationMinAltitude, 0, 5000000, DEFAULTS.relationMinAltitude)),
        relationMaxAltitude: Math.round(clamp(source.relationMaxAltitude, 500000, 25000000, DEFAULTS.relationMaxAltitude)),
    });
}

export function loadGlobalWatchPreferences() {
    try {
        const current = localStorage.getItem(STORAGE_KEY);
        if (current) return sanitize(JSON.parse(current));
        const legacy = JSON.parse(localStorage.getItem(LEGACY_STORAGE_KEY) || '{}');
        return sanitize({ ...legacy, renderQuality: 'ultra', max3DEntities: 10000, dynamicQuality: true, satelliteMinAltitude: 500000, aircraftMaxAltitude: 7000000, cameraMaxAltitude: 4500000, hazardMaxAltitude: 12000000 });
    } catch (_) { return DEFAULTS; }
}

export function saveGlobalWatchPreferences(candidate) {
    const preferences = sanitize(candidate);
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences)); } catch (_) { /* runtime state remains valid */ }
    return preferences;
}

export function globalWatchDefaults() { return DEFAULTS; }
