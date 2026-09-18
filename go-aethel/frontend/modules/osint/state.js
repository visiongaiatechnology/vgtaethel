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
  {lat: 50.4501, lon: 30.5234, name: "Kyiv Maidan Nezalezhnosti"},
  {lat: 35.6762, lon: 139.6503, name: "Tokyo Shibuya Crossing"},
  {lat: 1.3521, lon: 103.8198, name: "Singapore Marina Bay"},
  {lat: 37.5665, lon: 126.9780, name: "Seoul City Hall Plaza"},
  {lat: 25.0330, lon: 121.5654, name: "Taipei 101 area"},
  {lat: 31.7683, lon: 35.2137, name: "Jerusalem Old City View"},
  {lat: 32.0853, lon: 34.7818, name: "Tel Aviv Coastline"},
  {lat: 35.6892, lon: 51.3890, name: "Tehran Azadi Tower"},
  {lat: 24.7136, lon: 46.6753, name: "Riyadh Kingdom Centre"},
  {lat: 30.0444, lon: 31.2357, name: "Cairo Tahrir Square"},
  {lat: 41.0082, lon: 28.9784, name: "Istanbul Bosphorus Strait"}
];

export const satData = [
  {lat: 46.35, lon: -86.25, name: "OKEAN-3", alt: 558855, time: "now"},
  {lat: 51.50, lon: -0.12, name: "ISS (ZARYA)", alt: 420000, time: "now"},
  {lat: 28.60, lon: 80.50, name: "STARLINK-3142", alt: 540000, time: "now"},
  {lat: 12.30, lon: 45.10, name: "NOAA-20", alt: 824000, time: "now"},
  {lat: -15.40, lon: 110.20, name: "COSMOS-2558", alt: 450000, time: "now"}
];

export const citiesData = [
  {lat: 52.5200, lon: 13.4050, name: "Berlin (GER)", isCapital: true},
  {lat: 50.9375, lon: 6.9603, name: "Köln (GER)"},
  {lat: 53.5511, lon: 9.9937, name: "Hamburg (GER)"},
  {lat: 48.1351, lon: 11.5820, name: "München (GER)"},
  {lat: 50.1109, lon: 8.6821, name: "Frankfurt (GER)"},
  {lat: 51.5074, lon: -0.1278, name: "London (UK)", isCapital: true},
  {lat: 48.8566, lon: 2.3522, name: "Paris (FRA)", isCapital: true},
  {lat: 40.7128, lon: -74.0060, name: "New York (USA)"},
  {lat: 38.8951, lon: -77.0364, name: "Washington D.C. (USA)", isCapital: true},
  {lat: 34.0522, lon: -118.2437, name: "Los Angeles (USA)"},
  {lat: 55.7558, lon: 37.6173, name: "Moskau (RUS)", isCapital: true},
  {lat: 50.4501, lon: 30.5234, name: "Kiew (UKR)", isCapital: true},
  {lat: 52.2297, lon: 21.0122, name: "Warschau (POL)", isCapital: true},
  {lat: 39.9042, lon: 116.4074, name: "Peking (CHN)", isCapital: true},
  {lat: 31.2304, lon: 121.4737, name: "Shanghai (CHN)"},
  {lat: 35.6762, lon: 139.6503, name: "Tokio (JPN)", isCapital: true},
  {lat: 25.0330, lon: 121.5654, name: "Taipeh (TWN)", isCapital: true},
  {lat: 37.5665, lon: 126.9780, name: "Seoul (KOR)", isCapital: true},
  {lat: 39.0392, lon: 125.7625, name: "Pjöngjang (PRK)", isCapital: true},
  {lat: 31.7683, lon: 35.2137, name: "Jerusalem (ISR)", isCapital: true},
  {lat: 32.0853, lon: 34.7818, name: "Tel Aviv (ISR)"},
  {lat: 35.6892, lon: 51.3890, name: "Teheran (IRN)", isCapital: true},
  {lat: 24.7136, lon: 46.6753, name: "Riad (SAU)", isCapital: true},
  {lat: 30.0444, lon: 31.2357, name: "Kairo (EGY)", isCapital: true},
  {lat: 41.0082, lon: 28.9784, name: "Istanbul (TUR)"},
  {lat: 39.9334, lon: 32.8597, name: "Ankara (TUR)", isCapital: true},
  {lat: 1.3521, lon: 103.8198, name: "Singapur (SGP)", isCapital: true},
  {lat: 28.6139, lon: 77.2090, name: "Neu-Delhi (IND)", isCapital: true},
  {lat: -33.8688, lon: 151.2093, name: "Sydney (AUS)"},
  {lat: -35.2809, lon: 149.1300, name: "Canberra (AUS)", isCapital: true},
  {lat: 52.3676, lon: 4.9041, name: "Amsterdam (NLD)", isCapital: true},
  {lat: 50.8503, lon: 4.3517, name: "Brussels (BEL)", isCapital: true},
  {lat: 48.2082, lon: 16.3738, name: "Vienna (AUT)", isCapital: true},
  {lat: 50.0755, lon: 14.4378, name: "Prague (CZE)", isCapital: true},
  {lat: 47.4979, lon: 19.0402, name: "Budapest (HUN)", isCapital: true},
  {lat: 41.9028, lon: 12.4964, name: "Rome (ITA)", isCapital: true},
  {lat: 40.4168, lon: -3.7038, name: "Madrid (ESP)", isCapital: true},
  {lat: 38.7223, lon: -9.1393, name: "Lisbon (PRT)", isCapital: true},
  {lat: 59.3293, lon: 18.0686, name: "Stockholm (SWE)", isCapital: true},
  {lat: 59.9139, lon: 10.7522, name: "Oslo (NOR)", isCapital: true},
  {lat: 60.1699, lon: 24.9384, name: "Helsinki (FIN)", isCapital: true},
  {lat: 55.6761, lon: 12.5683, name: "Copenhagen (DNK)", isCapital: true},
  {lat: 45.4215, lon: -75.6972, name: "Ottawa (CAN)", isCapital: true},
  {lat: 19.4326, lon: -99.1332, name: "Mexico City (MEX)", isCapital: true},
  {lat: -23.5505, lon: -46.6333, name: "Sao Paulo (BRA)"},
  {lat: -15.7939, lon: -47.8828, name: "Brasilia (BRA)", isCapital: true},
  {lat: -34.6037, lon: -58.3816, name: "Buenos Aires (ARG)", isCapital: true},
  {lat: -33.4489, lon: -70.6693, name: "Santiago (CHL)", isCapital: true},
  {lat: 33.3152, lon: 44.3661, name: "Baghdad (IRQ)", isCapital: true},
  {lat: 25.2048, lon: 55.2708, name: "Dubai (UAE)"},
  {lat: 13.7563, lon: 100.5018, name: "Bangkok (THA)", isCapital: true},
  {lat: -6.2088, lon: 106.8456, name: "Jakarta (IDN)", isCapital: true},
  {lat: -1.2921, lon: 36.8219, name: "Nairobi (KEN)", isCapital: true},
  {lat: -25.7479, lon: 28.2293, name: "Pretoria (ZAF)", isCapital: true},
  {lat: 6.5244, lon: 3.3792, name: "Lagos (NGA)"},
  {lat: -41.2866, lon: 174.7756, name: "Wellington (NZL)", isCapital: true}
];

export const cableData = [
  // Trans-Atlantic routes
  [[-74.0, 40.7], [-65.0, 42.0], [-45.0, 47.0], [-20.0, 50.0], [-5.0, 50.0], [0.0, 51.0]],
  [[-75.0, 36.8], [-60.0, 38.0], [-30.0, 42.0], [-10.0, 43.5], [-2.0, 48.0]],
  [[-80.0, 26.0], [-65.0, 28.0], [-40.0, 32.0], [-15.0, 36.0], [-6.0, 36.5]],
  // Mediterranean & Red Sea & Middle East routes
  [[-5.0, 36.0], [5.0, 38.0], [15.0, 36.0], [25.0, 34.0], [32.0, 31.5], [32.5, 30.0]],
  [[32.5, 30.0], [35.0, 25.0], [40.0, 18.0], [43.5, 12.5], [50.0, 12.0], [60.0, 15.0], [72.0, 18.0]],
  [[55.0, 25.0], [58.0, 24.0], [65.0, 20.0], [75.0, 12.0], [80.0, 8.0]],
  // Indian Ocean to Southeast Asia
  [[72.0, 18.0], [80.0, 8.0], [90.0, 5.0], [98.0, 3.0], [103.8, 1.3]],
  // Trans-Pacific routes
  [[121.5, 25.0], [130.0, 28.0], [140.0, 32.0], [160.0, 35.0], [-180.0, 36.0], [-150.0, 36.0], [-125.0, 38.0], [-122.4, 37.8]],
  [[139.7, 35.7], [150.0, 38.0], [170.0, 42.0], [-170.0, 45.0], [-140.0, 48.0], [-124.0, 47.6]],
  // Intra-Asia routes
  [[103.8, 1.3], [108.0, 10.0], [115.0, 18.0], [121.5, 25.0], [126.9, 37.5], [139.7, 35.7]],
  // European North Sea / Baltic routes
  [[0.0, 51.5], [4.0, 52.5], [8.0, 55.0], [12.0, 56.0], [18.0, 58.0], [24.0, 60.0]]
];

// Shared regional-risk geo (mirrors intelligence.RegionalRiskCatalog).
// ring: closed [lon, lat] pairs for semi-transparent risk overlay fill/outline.
function regionBox(minLat, maxLat, minLon, maxLon) {
  const lat = (minLat + maxLat) / 2;
  const lon = (minLon + maxLon) / 2;
  return {
    lat, lon,
    minLat, maxLat, minLon, maxLon,
    ring: [
      [minLon, minLat], [maxLon, minLat], [maxLon, maxLat], [minLon, maxLat], [minLon, minLat]
    ]
  };
}

export const REGION_GEO = {
  GERMANY: regionBox(47.2, 55.0, 5.8, 15.0),
  BERLIN: { lat: 52.52, lon: 13.405, minLat: 52.3, maxLat: 52.7, minLon: 13.0, maxLon: 13.8, ring: [[13.0, 52.3], [13.8, 52.3], [13.8, 52.7], [13.0, 52.7], [13.0, 52.3]] },
  FRANCE: regionBox(42.3, 51.1, -5.0, 9.5),
  USA: regionBox(24.5, 49.0, -125.0, -66.9),
  UKRAINE: regionBox(44.3, 52.4, 22.0, 40.2),
  UK: regionBox(49.9, 60.8, -8.6, 1.7),
  RUSSIA: regionBox(41.2, 77.7, 19.6, 180.0),
  IRAN: regionBox(25.0, 39.8, 44.0, 63.3),
  CHINA: regionBox(18.2, 53.6, 73.5, 134.8),
  ISRAEL: regionBox(29.4, 33.4, 34.2, 35.9),
  TAIWAN: regionBox(21.8, 25.4, 119.3, 122.1),
  POLAND: regionBox(49.0, 54.9, 14.1, 24.2),
  BALTICS: regionBox(53.9, 59.7, 20.9, 28.3),
};

/** Catalog IDs expected in Regionales Risiko HUD (must match backend catalog). */
export const REGIONAL_RISK_CATALOG_IDS = Object.freeze([
  'GERMANY', 'FRANCE', 'USA', 'UKRAINE', 'UK',
  'RUSSIA', 'IRAN', 'CHINA', 'ISRAEL', 'TAIWAN', 'POLAND', 'BALTICS'
]);

export let cachedRiskMarkers = [];
export function setCachedRiskMarkers(val) { cachedRiskMarkers = val; }
export let cachedRiskDetails = [];
export function setCachedRiskDetails(val) { cachedRiskDetails = Array.isArray(val) ? val : []; }
