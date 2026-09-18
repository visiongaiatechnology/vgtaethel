import {
  activeFeedEvents,
  setLocalGlobeSelectedIndex,
} from './state.js';
import { sharedGeoManager } from '../geo_renderer/geo_manager.js';

// Cesium is the sole Global Watch geospatial engine. This module keeps the
// stable command surface used by feed and selection modules.

export function requestGlobeRender() {
  const renderer = sharedGeoManager?.currentRenderer;
  if (!renderer) return;
  renderer.setData(sharedGeoManager.regions, sharedGeoManager.links, sharedGeoManager.entities);
}

export function forceGlobeResize() {
  if (sharedGeoManager && sharedGeoManager.currentRenderer && sharedGeoManager.currentRenderer.viewer) {
    try {
      sharedGeoManager.currentRenderer.viewer.resize();
    } catch (_) {}
  }
}

export function focusGlobeOnLonLat(lon, lat, opts = {}) {
  const latitude = Number(lat);
  const longitude = Number(lon);
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return;
  const scale = Number(opts.scale);
  const altitude = Number.isFinite(scale) ? Math.max(800, 4000000 / Math.max(0.5, scale)) : 750000;
  sharedGeoManager.flyTo(latitude, longitude, altitude, -45, 0);
}

export function highlightEventInList(idx, showSelectionDetailsFn) {
  const cards = document.querySelectorAll("#gw-feed-list .gw-event-card, #gw-hazard-list .gw-event-card");
  cards.forEach((c) => c.classList.remove("highlighted"));
  let targetCard = null;
  cards.forEach((c) => {
    if (Number(c.dataset.idx) === idx) targetCard = c;
  });
  if (targetCard) {
    targetCard.classList.add("highlighted");
    const summary = targetCard.querySelector(".gw-event-summary");
    if (summary) summary.style.display = "block";
    targetCard.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }

  const ev = activeFeedEvents[idx];
  if (showSelectionDetailsFn) showSelectionDetailsFn(ev, null);
  if (ev && (ev.lat != null || ev.lon != null)) {
    focusGlobeOnLonLat(ev.lon, ev.lat);
  }
}

export function clearPins() {
  setLocalGlobeSelectedIndex(-1);
  sharedGeoManager.sceneState.setSelectedEntity(null);
}
