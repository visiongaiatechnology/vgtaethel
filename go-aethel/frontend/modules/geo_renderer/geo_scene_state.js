/**
 * Unified GEOINT Scene State for VGT Aethel.
 * Maintains persistent camera position, altitude, tracked targets, layer states,
 * and sensor modes to ensure seamless context preservation during renderer switching.
 */

export const CAMERA_SCALES = Object.freeze({
  ORBITAL: { name: 'ORBITAL', altitudeM: 20000000, label: '20,000 km' },
  GLOBAL: { name: 'GLOBAL', altitudeM: 8000000, label: '8,000 km' },
  CONTINENT: { name: 'CONTINENT', altitudeM: 3000000, label: '3,000 km' },
  REGIONAL: { name: 'REGIONAL', altitudeM: 500000, label: '500 km' },
  TACTICAL: { name: 'TACTICAL', altitudeM: 100000, label: '100 km' },
  CITY: { name: 'CITY', altitudeM: 20000, label: '20 km' },
  LOCAL: { name: 'LOCAL', altitudeM: 5000, label: '5 km' },
  GROUND: { name: 'GROUND', altitudeM: 800, label: '< 1 km' },
});

export const MAP_PROVIDERS = Object.freeze({
  GOOGLE_3D: 'GOOGLE_3D',         // Photorealistic 3D Tiles
  ARCGIS_SATELLITE: 'ARCGIS_SATELLITE', // High-Res Global Satellite (Default, unblocked)
  CARTO_DARK: 'CARTO_DARK',       // Tactical Dark Basemap
  CARTO_VOYAGER: 'CARTO_VOYAGER', // High-Res Clean Vector-Raster
  CESIUM_ION: 'CESIUM_ION',       // Bing Aerial with Labels (if token provided)
});

const SCENE_STORAGE_KEY = 'aethel.geoint.scene.v1';

function finiteNumber(value, fallback, minimum, maximum) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return fallback;
  return Math.min(maximum, Math.max(minimum, numeric));
}

function loadPersistedScene() {
  try {
    const value = JSON.parse(localStorage.getItem(SCENE_STORAGE_KEY) || '{}');
    return value && typeof value === 'object' ? value : {};
  } catch (_) {
    return {};
  }
}

export class GeoSceneState {
  constructor() {
    const persisted = loadPersistedScene();
    this.mode = 'CESIUM_3D';
    const providers = new Set(Object.values(MAP_PROVIDERS));
    this.mapProvider = providers.has(persisted.mapProvider) ? persisted.mapProvider : MAP_PROVIDERS.ARCGIS_SATELLITE;
    this.providerStatus = 'LIVE'; // 'LIVE' | 'DEGRADED' | 'FALLBACK' | 'OFFLINE' | 'ERROR'

    // Camera state
    this.camera = {
      lat: 50.9375, // Default Cologne, Germany overview
      lon: 6.9603,
      altitude: 4000000,
      heading: 0,
      pitch: -85, // Top-down overview facing Earth
      roll: 0,
      scalePreset: CAMERA_SCALES.CONTINENT.name,
    };
    if (persisted.camera && typeof persisted.camera === 'object') {
      this.camera.lat = finiteNumber(persisted.camera.lat, this.camera.lat, -90, 90);
      this.camera.lon = finiteNumber(persisted.camera.lon, this.camera.lon, -180, 180);
      this.camera.altitude = finiteNumber(persisted.camera.altitude, this.camera.altitude, 100, 35000000);
      this.camera.heading = finiteNumber(persisted.camera.heading, this.camera.heading, -360, 360);
      this.camera.pitch = finiteNumber(persisted.camera.pitch, -85, -90, -20);
      this.camera.scalePreset = this.deriveScalePreset(this.camera.altitude);
    }

    // Selection & Tracking
    this.selectedEntity = null;
    this.trackedEntityId = null;

    // Layer Visibility
    this.layers = {
      borders: true,
      daynight: true,
      aircraft: true,
      military: true,
      vessels: true,
      satellites: true,
      cameras: true,
      cctv: true,
      cables: true,
      news: true,
      events: true,
      earthquakes: true,
      fires: true,
      volcanoes: true,
      cities: true,
      risks: true,
      terrain: true,
      buildings: true,
    };
    if (persisted.layers && typeof persisted.layers === 'object') {
      for (const key of Object.keys(this.layers)) {
        if (typeof persisted.layers[key] === 'boolean') this.layers[key] = persisted.layers[key];
      }
    }

    this.filters = {
      entityTypes: [], relationCategories: [], minimumConfidence: 0, freshness: 'CURRENT', sourceIDs: [],
    };
    this.timeWindowHours = finiteNumber(persisted.timeWindowHours, 72, 1, 168);
    this.quality = { profile: 'balanced', budgetLevel: 0, dynamic: true };
    this.diagnostics = { enabled: false, averageFPS: 0, visibleEntities: 0, lastSyncAt: null };
    this.selectedRegion = null;
    this.annotations = [];

    // Sensor mode
    const sensorModes = new Set(['NORMAL', 'FLIR_WHITE_HOT', 'FLIR_IRONBOW', 'NVG_GREEN', 'CRT_HUD']);
    this.sensorMode = sensorModes.has(persisted.sensorMode) ? persisted.sensorMode : 'NORMAL';

    // Subscribers
    this.listeners = new Set();
  }

  updateCamera(lat, lon, altitude, heading, pitch, roll) {
    this.camera.lat = finiteNumber(lat, this.camera.lat, -90, 90);
    this.camera.lon = finiteNumber(lon, this.camera.lon, -180, 180);
    this.camera.altitude = finiteNumber(altitude, this.camera.altitude, 100, 50000000);
    if (heading != null) this.camera.heading = finiteNumber(heading, this.camera.heading, -360, 360);
    if (pitch != null) this.camera.pitch = finiteNumber(pitch, this.camera.pitch, -90, 90);
    if (roll != null) this.camera.roll = finiteNumber(roll, this.camera.roll, -180, 180);

    // Derive scale preset
    this.camera.scalePreset = this.deriveScalePreset(this.camera.altitude);
    this.notify('camera');
  }

  deriveScalePreset(altM) {
    if (altM > 12000000) return CAMERA_SCALES.ORBITAL.name;
    if (altM > 5000000) return CAMERA_SCALES.GLOBAL.name;
    if (altM > 1500000) return CAMERA_SCALES.CONTINENT.name;
    if (altM > 250000) return CAMERA_SCALES.REGIONAL.name;
    if (altM > 50000) return CAMERA_SCALES.TACTICAL.name;
    if (altM > 10000) return CAMERA_SCALES.CITY.name;
    if (altM > 2000) return CAMERA_SCALES.LOCAL.name;
    return CAMERA_SCALES.GROUND.name;
  }

  setSelectedEntity(entity) {
    this.selectedEntity = entity;
    this.notify('selection');
  }

  setTrackedEntity(entityId) {
    this.trackedEntityId = entityId;
    this.notify('tracking');
  }

  setLayer(layerName, visible) {
    const val = !!visible;
    this.layers[layerName] = val;
    if (layerName === 'cameras') this.layers.cctv = val;
    if (layerName === 'cctv') this.layers.cameras = val;
    if (layerName === 'news') this.layers.events = val;
    if (layerName === 'events') this.layers.news = val;
    this.notify('layers');
  }

  setSensorMode(mode) {
    const allowed = new Set(['NORMAL', 'FLIR_WHITE_HOT', 'FLIR_IRONBOW', 'NVG_GREEN', 'CRT_HUD']);
    this.sensorMode = allowed.has(mode) ? mode : 'NORMAL';
    this.notify('sensor');
  }

  setRendererMode(mode) {
    this.mode = 'CESIUM_3D';
    this.notify('mode');
  }

  setMapProvider(provider, status = 'LIVE') {
    this.mapProvider = provider;
    this.providerStatus = status;
    this.notify('provider');
  }

  setFilters(filters = {}) {
    this.filters = {
      entityTypes: Array.isArray(filters.entityTypes) ? [...new Set(filters.entityTypes.map(String))].slice(0, 32) : this.filters.entityTypes,
      relationCategories: Array.isArray(filters.relationCategories) ? [...new Set(filters.relationCategories.map(String))].slice(0, 16) : this.filters.relationCategories,
      minimumConfidence: finiteNumber(filters.minimumConfidence, this.filters.minimumConfidence, 0, 100),
      freshness: ['CURRENT', 'AGING', 'ALL'].includes(filters.freshness) ? filters.freshness : this.filters.freshness,
      sourceIDs: Array.isArray(filters.sourceIDs) ? [...new Set(filters.sourceIDs.map(String))].slice(0, 100) : this.filters.sourceIDs,
    };
    this.notify('filters');
  }

  setTimeWindow(hours) {
    this.timeWindowHours = finiteNumber(hours, this.timeWindowHours, 1, 168);
    this.notify('time-window');
  }

  setQuality(profile, budgetLevel = 0, dynamic = true) {
    this.quality = { profile: String(profile || 'balanced'), budgetLevel: finiteNumber(budgetLevel, 0, 0, 3), dynamic: dynamic !== false };
    this.notify('quality');
  }

  subscribe(fn) {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  notify(event) {
    this.persist();
    for (const fn of this.listeners) {
      try {
        fn(event, this);
      } catch (err) {
        console.error('[SCENE_STATE] Listener error:', err);
      }
    }
  }

  persist() {
    try {
      localStorage.setItem(SCENE_STORAGE_KEY, JSON.stringify({
        camera: this.camera, layers: this.layers, timeWindowHours: this.timeWindowHours,
        mapProvider: this.mapProvider, sensorMode: this.sensorMode,
      }));
    } catch (_) {}
  }
}

export const sharedSceneState = new GeoSceneState();
