import {
  loadGlobalWatchPreferences,
  saveGlobalWatchPreferences,
  globalWatchDefaults,
} from '../global_watch_preferences.js';

// Re-export prefs helpers so UI modules can import from one shared state surface.
export { loadGlobalWatchPreferences, saveGlobalWatchPreferences, globalWatchDefaults };

export let globeContainer = null;
export function setGlobeContainer(el) { globeContainer = el; }

export let localGlobeCanvas = null;
export function setLocalGlobeCanvas(el) { localGlobeCanvas = el; }

export let localGlobeCtx = null;
export function setLocalGlobeCtx(ctx) { localGlobeCtx = ctx; }

export let activeFeedEvents = [];
export function setActiveFeedEvents(events) { activeFeedEvents = events; }

export let localGlobeRotY = -10.5 * Math.PI / 180;
export function setLocalGlobeRotY(val) { localGlobeRotY = val; }

export let localGlobeRotX = 50 * Math.PI / 180;
export function setLocalGlobeRotX(val) { localGlobeRotX = val; }

export let globalWatchCommandStream = null;
export function setGlobalWatchCommandStream(stream) { globalWatchCommandStream = stream; }

export let localGlobeScale = 1.05;
export function setLocalGlobeScale(val) { localGlobeScale = val; }

export let localGlobeSelectedIndex = -1;
export function setLocalGlobeSelectedIndex(val) { localGlobeSelectedIndex = val; }

export let localGlobeDragging = false;
export function setLocalGlobeDragging(val) { localGlobeDragging = val; }

export let localGlobeLastX = 0;
export function setLocalGlobeLastX(val) { localGlobeLastX = val; }

export let localGlobeLastY = 0;
export function setLocalGlobeLastY(val) { localGlobeLastY = val; }

export let localAtlasBorders = [];
export function setLocalAtlasBorders(borders) { localAtlasBorders = borders; }

export let localAtlasVersion = 0;
export function incrementLocalAtlasVersion() { localAtlasVersion++; }

export let projectedAtlasCache = null;
export function setProjectedAtlasCache(val) { projectedAtlasCache = val; }

export let projectedGridCache = null;
export function setProjectedGridCache(val) { projectedGridCache = val; }

export let projectedNightCache = null;
export function setProjectedNightCache(val) { projectedNightCache = val; }

export let globeRenderQueued = false;
export function setGlobeRenderQueued(val) { globeRenderQueued = val; }

export let globalWatchAutoRefreshTimer = null;
export function setGlobalWatchAutoRefreshTimer(timer) { globalWatchAutoRefreshTimer = timer; }

export let globalWatchHazardTimer = null;
export function setGlobalWatchHazardTimer(timer) { globalWatchHazardTimer = timer; }

export let lastGeneratedBriefing = '';
export function setLastGeneratedBriefing(val) { lastGeneratedBriefing = val; }

export let globalWatchPreferences = loadGlobalWatchPreferences();
export function updateGlobalWatchPreferences(next) {
    globalWatchPreferences = saveGlobalWatchPreferences(next);
    return globalWatchPreferences;
}

export const EARTH_TEX_MAX_W = 8192;
export const EARTH_TEX_WORK_MAX_W = 4096;

export let earthTexCanvas = null;
export let earthTexReady = false;
export function setEarthTexReady(val) { earthTexReady = val; }

export let earthTexSource = 'none';
export function setEarthTexSource(val) { earthTexSource = val; }

export let earthSphereCache = null;
export function setEarthSphereCache(val) { earthSphereCache = val; }

export let earthImgData = null;
export function setEarthImgData(data) { earthImgData = data; }

export let earthImgW = 0;
export function setEarthImgW(w) { earthImgW = w; }

export let earthImgH = 0;
export function setEarthImgH(h) { earthImgH = h; }

export let earthOffscreen = null;
export function setEarthOffscreen(off) { earthOffscreen = off; }

export let earthTexMeta = { width: 0, height: 0, source: 'none', bytesHint: 0 };
export function setEarthTexMeta(meta) { earthTexMeta = meta; }

export let earthHiQualityIdle = true;

export let globeIdleRotationFrame = 0;
export function setGlobeIdleRotationFrame(val) { globeIdleRotationFrame = val; }

export let globeIdleRotationLastFrame = 0;
export function setGlobeIdleRotationLastFrame(val) { globeIdleRotationLastFrame = val; }

export let globeIdleRotationBlockedUntil = 0;
export function setGlobeIdleRotationBlockedUntil(val) { globeIdleRotationBlockedUntil = val; }

export let globeKnownEventIDs = null;
export function setGlobeKnownEventIDs(val) { globeKnownEventIDs = val; }

export let globeTransientEventTimer = 0;
export function setGlobeTransientEventTimer(val) { globeTransientEventTimer = val; }

export let globeTransientEventID = '';
export function setGlobeTransientEventID(val) { globeTransientEventID = val; }

export const GLOBE_IDLE_ROTATION_DELAY_MS = 20000;
// 25 FPS is the measured balance between perceptually smooth idle motion and
// the CPU cost of the sovereign per-pixel sphere projection in WebView2.
export const GLOBE_IDLE_ROTATION_FRAME_MS = 40;
export const GLOBE_IDLE_ROTATION_RADIANS_PER_SECOND = 0.022;
export const GLOBE_IDLE_NORMAL_SCALE = 1.05;

export let cameraData = [
  {lat: 51.5074, lon: -0.1278, name: "London Eye (public cam area)"},
  {lat: 40.7128, lon: -74.0060, name: "New York Times Square (public view)"},
  {lat: 48.8566, lon: 2.3522, name: "Paris Tour Eiffel area"},
  {lat: -33.8688, lon: 151.2093, name: "Sydney Harbour Bridge"},
  {lat: 52.52, lon: 13.405, name: "Berlin Alexanderplatz (public)"},
  {lat: 55.7558, lon: 37.6173, name: "Moscow Red Square area"},
  {lat: 35.6762, lon: 139.6503, name: "Tokyo Shibuya Crossing"},
  {lat: 1.3521, lon: 103.8198, name: "Singapore Marina Bay"},
  {lat: 37.5665, lon: 126.9780, name: "Seoul City Hall Plaza"},
  {lat: 25.0330, lon: 121.5654, name: "Taipei 101 area"}
];

export const satData = [
  {lat: 46.35, lon: -86.25, name: "OKEAN-3", alt: 558855, time: "now"}
];

export const citiesData = [
  {lat:52.52, lon:13.405, name:"Berlin"},
  {lat:51.507, lon:-0.128, name:"London"},
  {lat:48.857, lon:2.352, name:"Paris"},
  {lat:40.713, lon:-74.006, name:"New York"},
  {lat:55.756, lon:37.617, name:"Moscow"},
  {lat:39.904, lon:116.407, name:"Beijing"},
  {lat:35.676, lon:139.65, name:"Tokyo"},
  {lat:-33.869, lon:151.209, name:"Sydney"}
];

export const cableData = [
  [[-70,40], [-30,40], [10,35], [100,20]]
];

export const REGION_GEO = {
  GERMANY: { lat: 51.2, lon: 10.5 },
  BERLIN: { lat: 52.52, lon: 13.405 },
  FRANCE: { lat: 46.6, lon: 2.2 },
  USA: { lat: 39.5, lon: -98.3 },
  UKRAINE: { lat: 48.4, lon: 31.2 },
  UK: { lat: 54.0, lon: -2.5 }
};

export let cachedRiskMarkers = [];
export function setCachedRiskMarkers(val) { cachedRiskMarkers = val; }
