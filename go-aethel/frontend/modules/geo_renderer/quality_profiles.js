// STATUS: DIAMANT VGT SUPREME

export const GEO_QUALITY_PROFILES = Object.freeze({
  power_saver: Object.freeze({ targetFPS: 24, maximumScreenSpaceError: 6, resolutionScale: 0.72, maxVisibleEntities: 500, trailPoints: 8, labelDistance: 180000, relationDistance: 6500000, telemetryHz: 1, postProcessing: false, postProcessingLevel: 'off', atmosphere: false, dynamicQuality: true, terrainQuality: 'low', tilesQuality: 'low', clustering: 'aggressive', fog: false, antiAliasing: false }),
  balanced: Object.freeze({ targetFPS: 45, maximumScreenSpaceError: 3, resolutionScale: 0.9, maxVisibleEntities: 2500, trailPoints: 16, labelDistance: 500000, relationDistance: 12000000, telemetryHz: 1, postProcessing: true, postProcessingLevel: 'low', atmosphere: true, dynamicQuality: true, terrainQuality: 'medium', tilesQuality: 'medium', clustering: 'adaptive', fog: true, antiAliasing: true }),
  high: Object.freeze({ targetFPS: 60, maximumScreenSpaceError: 2, resolutionScale: 1, maxVisibleEntities: 5000, trailPoints: 24, labelDistance: 900000, relationDistance: 16000000, telemetryHz: 1, postProcessing: true, postProcessingLevel: 'full', atmosphere: true, dynamicQuality: true, terrainQuality: 'high', tilesQuality: 'high', clustering: 'light', fog: true, antiAliasing: true }),
  ultra: Object.freeze({ targetFPS: 60, maximumScreenSpaceError: 1.25, resolutionScale: 1.25, maxVisibleEntities: 10000, trailPoints: 30, labelDistance: 1500000, relationDistance: 20000000, telemetryHz: 1, postProcessing: true, postProcessingLevel: 'full', atmosphere: true, dynamicQuality: false, terrainQuality: 'ultra', tilesQuality: 'ultra', clustering: 'off', fog: true, antiAliasing: true }),
});

const PROFILE_ALIASES = Object.freeze({ performance: 'power_saver', power_saver: 'power_saver', balanced: 'balanced', high: 'high', ultra: 'ultra', custom: 'balanced' });

function clampNumber(value, minimum, maximum, fallback) {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? Math.min(maximum, Math.max(minimum, numeric)) : fallback;
}

export function normalizeGeoQualityProfile(profileName, overrides = {}) {
  const key = PROFILE_ALIASES[String(profileName || '').toLowerCase()] || 'balanced';
  const base = GEO_QUALITY_PROFILES[key];
  return Object.freeze({
    name: key,
    targetFPS: clampNumber(overrides.targetFPS, 15, 60, base.targetFPS),
    maximumScreenSpaceError: clampNumber(overrides.maximumScreenSpaceError, 1, 16, base.maximumScreenSpaceError),
    resolutionScale: clampNumber(overrides.resolutionScale, 0.5, 1.5, base.resolutionScale),
    maxVisibleEntities: Math.round(clampNumber(overrides.maxVisibleEntities, 100, 10000, base.maxVisibleEntities)),
    trailPoints: Math.round(clampNumber(overrides.trailPoints, 0, 30, base.trailPoints)),
    labelDistance: clampNumber(overrides.labelDistance, 50000, 5000000, base.labelDistance),
    relationDistance: clampNumber(overrides.relationDistance, 1000000, 25000000, base.relationDistance),
    entityDrawDistance: clampNumber(overrides.entityDrawDistance, 1000000, 50000000, 20000000),
    modelDrawDistance: clampNumber(overrides.modelDrawDistance, 5000, 500000, 50000),
    trailDensity: Math.round(clampNumber(overrides.trailDensity, 1, 8, 2)),
    terrainQuality: ['off', 'low', 'medium', 'high', 'ultra'].includes(overrides.terrainQuality) ? overrides.terrainQuality : base.terrainQuality,
    tilesQuality: ['low', 'medium', 'high', 'ultra'].includes(overrides.tilesQuality) ? overrides.tilesQuality : base.tilesQuality,
    dynamicScreenSpaceError: overrides.dynamicScreenSpaceError !== false,
    antiAliasing: typeof overrides.antiAliasing === 'boolean' ? overrides.antiAliasing : base.antiAliasing,
    fog: typeof overrides.fog === 'boolean' ? overrides.fog : base.fog,
    shadows: overrides.shadows === true,
    lighting: overrides.lighting !== false,
    highResolutionTextures: overrides.highResolutionTextures === true,
    sensorEffects: overrides.sensorEffects !== false,
    hudDensity: ['minimal', 'standard', 'dense'].includes(overrides.hudDensity) ? overrides.hudDensity : 'standard',
    automaticLOD: overrides.automaticLOD !== false,
    clustering: ['off', 'light', 'adaptive', 'aggressive'].includes(overrides.clustering) ? overrides.clustering : base.clustering,
    postProcessingLevel: ['off', 'low', 'full'].includes(overrides.postProcessingLevel) ? overrides.postProcessingLevel : base.postProcessingLevel,
    telemetryHz: 1,
    postProcessing: typeof overrides.postProcessing === 'boolean' ? overrides.postProcessing : base.postProcessing,
    atmosphere: typeof overrides.atmosphere === 'boolean' ? overrides.atmosphere : base.atmosphere,
    dynamicQuality: typeof overrides.dynamicQuality === 'boolean' ? overrides.dynamicQuality : base.dynamicQuality,
  });
}
