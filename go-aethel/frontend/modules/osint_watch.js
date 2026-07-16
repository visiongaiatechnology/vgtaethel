// VGT AETHEL — Global Watch public entry
// Stable import surface for app.js / ui.js. Implementation lives under ./osint/.
// Behavior is owned by the split modules; this file only re-exports.

export {
  initGlobalWatch,
  refreshOSINTFeed,
  loadAndRenderRegionalRisks,
  forceGlobeResize,
  focusRegionByKey,
  showAethelToast,
  showAethelModal,
} from './osint/ui_controls.js';

// Re-export pure helpers used by tests / optional external callers
export {
  magnitudeColorBand,
  magnitudeBandColor,
  volcanoMarkerColor,
  parseMagnitudeFromEvent,
  isEarthquakeEvent,
  isVolcanoEvent,
  isEruptingVolcano,
  formatArticleDateTime,
} from './osint/hazards.js';

export {
  projectLatLon,
  unprojectScreenToLatLon,
  projectUnprojectRoundTrip,
  applyDomainFilter,
  hitTestPin,
  applyDragDelta,
  applyWheelZoom,
  resetGlobeView,
  computeGlobeFocusRotation,
  filterEventsByTimeWindow,
  filterEventsByRegion,
  eventMatchesRegion,
  eventTimestampMs,
  buildGlobePins,
  isGeoEvent,
  REGION_FOCUS_TABLE,
  REGION_GEO_BOUNDS,
  globeRadius,
} from './osint/projection.js';

export {
  computeEarthRasterStep,
  markGlobeInteraction,
  shouldRotateGlobe,
  earthRasterDPRCap,
  earthTexWorkingMaxW,
} from './osint/texture_atlas.js';

export {
  getGwTimeWindowHours,
  setGwTimeWindowHours,
  activeGlobalWatchDomain,
  stableGlobeEventID,
  subscribeGlobalWatchCommands,
} from './osint/feed_and_risks.js';

export {
  epistemicLayer,
  showSelectionDetails,
  showCameraDetails,
  promoteCurrentSelection,
  saveSelectionToNexus,
  aiBriefingForSelection,
  observeAtSelection,
} from './osint/selection_and_chat.js';

export {
  triggerAIBriefing,
  openGwReportReader,
  wireBriefingWorkspace,
} from './osint/briefing_and_reader.js';

export {
  requestGlobeRender,
  focusGlobeOnLonLat,
  drawPureLocalGlobe,
  initPureLocalGlobe,
} from './osint/globe_render.js';

export {
  EARTH_TEX_MAX_W,
  EARTH_TEX_WORK_MAX_W,
  REGION_GEO,
  cameraData,
  citiesData,
} from './osint/state.js';
