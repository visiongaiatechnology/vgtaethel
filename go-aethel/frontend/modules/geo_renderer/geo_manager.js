import { RENDERER_MODES } from './renderer_interface.js';
import { EnhancedCesiumRenderer } from './enhanced_cesium_renderer.js';
import { sharedSceneState } from './geo_scene_state.js';

/**
 * GeoManager:
 * Central GEOINT Coordinator managing Enhanced 3D and Sovereign Renderers,
 * live entity streams, contact roster radar, and seamless scene state transitions.
 */
export class GeoManager {
  constructor() {
    this.container = null;
    this.currentRenderer = null;
    this.mode = RENDERER_MODES.CESIUM_3D;
    this.entities = [];
    this.regions = [];
    this.links = [];
    this.sceneState = sharedSceneState;
    this.pollInterval = null;
    this.pollController = null;
    this.pollInFlight = false;
    this.entityETag = '';
    this.relationETag = '';
    this.destroyed = false;
    this.visibilityHandler = null;
    this.pollFunction = null;
    this.trackEventHandler = null;
    this.focusEventHandler = null;
    this.lastStatusAt = 0;
    this.sceneUnsubscribe = null;
    this.onSelectEntity = () => {};
  }

  async init(container, onSelectEntity) {
    this.container = container;
    this.destroyed = false;
    this.onSelectEntity = onSelectEntity || (() => {});
    if (!this.sceneUnsubscribe) {
      this.sceneUnsubscribe = this.sceneState.subscribe(event => {
        if (event === 'camera' || event === 'time-window' || event === 'filters') this.requestDataRefresh();
      });
    }

    const ready = await this.setMode(RENDERER_MODES.CESIUM_3D);
    if (!ready) return false;
    this.startDataPolling();
    this.bindUIControls();
    return true;
  }

  async setMode(mode = RENDERER_MODES.CESIUM_3D) {
    this.mode = RENDERER_MODES.CESIUM_3D;
    this.sceneState.setRendererMode(RENDERER_MODES.CESIUM_3D);

    if (!this.currentRenderer) {
      this.currentRenderer = new EnhancedCesiumRenderer(this.container, {
        onSelectEntity: this.onSelectEntity,
        sceneState: this.sceneState,
      });
      const ready = await this.currentRenderer.init();
      if (!ready) {
        const detail = this.currentRenderer.lastInitializationError;
        this.currentRenderer.destroy();
        this.currentRenderer = null;
        this.renderInitializationFailure(detail);
        return false;
      }
    }

    this.currentRenderer.setData(this.regions, this.links, this.entities);
    this.currentRenderer.setLayers(this.sceneState.layers);
    this.currentRenderer.applyShader(this.sceneState.sensorMode);
    this.updateModeButtonUI();
    return true;
  }

  renderInitializationFailure(detail = '') {
    if (!this.container) return;
    let panel = document.getElementById('gw-cesium-init-error');
    if (!panel) {
      panel = document.createElement('div');
      panel.id = 'gw-cesium-init-error';
      panel.className = 'gw-contact-empty font-mono';
      this.container.appendChild(panel);
    }
    panel.textContent = detail
      ? `CESIUM ENGINE OFFLINE · ${detail}`
      : 'CESIUM ENGINE OFFLINE · Lokale Runtime konnte nicht initialisiert werden.';
  }

  setShader(shaderType) {
    this.sceneState.setSensorMode(shaderType);
    if (this.currentRenderer) {
      this.currentRenderer.applyShader(shaderType);
    }
  }

  setLayer(layerName, enabled) {
    this.sceneState.setLayer(layerName, enabled);
    if (this.currentRenderer) {
      this.currentRenderer.setLayers({ [layerName]: enabled });
    }
  }

  setMapProvider(providerKey, apiKey = '') {
    if (this.currentRenderer && this.currentRenderer.setMapProvider) {
      return this.currentRenderer.setMapProvider(providerKey, apiKey);
    }
  }

  focus(lat, lon, zoomOrScale) {
    if (this.currentRenderer) {
      this.currentRenderer.focus(lat, lon, zoomOrScale);
    }
    this.fetchContactRoster(lat, lon);
  }

  flyTo(lat, lon, altitude, pitch, heading) {
    if (this.currentRenderer && this.currentRenderer.flyTo) {
      this.currentRenderer.flyTo(lat, lon, altitude, pitch, heading);
    } else {
      this.focus(lat, lon);
    }
    this.fetchContactRoster(lat, lon);
  }

  focusRegion(regionKey) {
    if (this.currentRenderer && this.currentRenderer.focusRegion) {
      this.currentRenderer.focusRegion(regionKey);
    }
  }

  trackEntity(entityId) {
    if (this.currentRenderer) {
      this.currentRenderer.trackEntity(entityId);
    }
  }

  startDataPolling() {
    if (this.pollInterval) clearTimeout(this.pollInterval);

    const poll = async () => {
      if (this.destroyed || this.pollInFlight || document.hidden) {
        this.scheduleNextPoll(poll);
        return;
      }
      this.pollInFlight = true;
      this.pollController = new AbortController();
      try {
        const preferences = JSON.parse(localStorage.getItem('aethel.globalWatch.preferences.v5') || '{}');
        const limit = Math.max(100, Math.min(10000, Number(preferences.max3DEntities) || 2500));
        const entityHeaders = this.entityETag ? { 'If-None-Match': this.entityETag } : {};
        const relationHeaders = this.relationETag ? { 'If-None-Match': this.relationETag } : {};
        const [entityResponse, relationResponse] = await Promise.all([
          fetch(this.entityRequestURL(limit), { headers: entityHeaders, signal: this.pollController.signal }),
          fetch(`/v1/geoint/relations?hours=${Math.max(1, Math.min(168, Number(this.sceneState.timeWindowHours) || 72))}`, { headers: relationHeaders, signal: this.pollController.signal }),
        ]);
        if (entityResponse.ok) {
          const data = await entityResponse.json();
          this.entities = data.entities || [];
          this.entityETag = entityResponse.headers.get('ETag') || this.entityETag;
          this.updateEntityCountHUD();
        }
        if (relationResponse.ok) {
          const data = await relationResponse.json();
          this.links = data.relations || [];
          this.relationETag = relationResponse.headers.get('ETag') || this.relationETag;
        }
        if (this.currentRenderer && (entityResponse.status !== 304 || relationResponse.status !== 304)) {
          this.currentRenderer.setData(this.regions, this.links, this.entities);
        }
        if (Date.now() - this.lastStatusAt > 60000) void this.refreshLayerStatus();
      } catch (error) {
        if (error?.name !== 'AbortError') console.warn('[GEO_MANAGER] Data refresh degraded:', error);
      } finally {
        this.pollInFlight = false;
        this.pollController = null;
        this.scheduleNextPoll(poll);
      }
    };

    this.visibilityHandler = () => {
      if (!document.hidden) {
        if (this.pollInterval) clearTimeout(this.pollInterval);
        this.pollInterval = setTimeout(poll, 0);
      }
    };
    this.pollFunction = poll;
    document.addEventListener('visibilitychange', this.visibilityHandler);
    void poll();
  }

  scheduleNextPoll(poll) {
    if (this.destroyed) return;
    if (this.pollInterval) clearTimeout(this.pollInterval);
    let cadenceMs = 12000;
    try {
      const preferences = JSON.parse(localStorage.getItem('aethel.globalWatch.preferences.v5') || '{}');
      cadenceMs = Math.max(5000, Math.min(120000, Number(preferences.aircraftUpdateSeconds || 12) * 1000));
    } catch (_) {}
    this.pollInterval = setTimeout(poll, document.hidden ? Math.max(60000, cadenceMs) : cadenceMs);
  }

  entityRequestURL(limit) {
    const params = new URLSearchParams({ limit: String(limit), hours: String(Math.max(1, Math.min(168, Number(this.sceneState.timeWindowHours) || 72))) });
    const camera = this.sceneState.camera;
    if (Number(camera.altitude) < 5000000) {
      const latitudeSpan = Math.max(1, Math.min(80, Number(camera.altitude) / 90000));
      const longitudeSpan = Math.min(170, latitudeSpan / Math.max(0.2, Math.cos(Number(camera.lat) * Math.PI / 180)));
      const minLat = Math.max(-90, Number(camera.lat) - latitudeSpan);
      const maxLat = Math.min(90, Number(camera.lat) + latitudeSpan);
      const minLon = Number(camera.lon) - longitudeSpan;
      const maxLon = Number(camera.lon) + longitudeSpan;
      if (minLon >= -180 && maxLon <= 180) {
        params.set('min_lat', minLat.toFixed(4)); params.set('max_lat', maxLat.toFixed(4));
        params.set('min_lon', minLon.toFixed(4)); params.set('max_lon', maxLon.toFixed(4));
      }
    }
    return `/v1/geoint/entities?${params.toString()}`;
  }

  requestDataRefresh() {
    this.entityETag = '';
    this.relationETag = '';
    if (this.pollInterval) clearTimeout(this.pollInterval);
    if (this.pollFunction && !this.destroyed) this.pollInterval = setTimeout(this.pollFunction, 0);
  }

  async refreshLayerStatus() {
    this.lastStatusAt = Date.now();
    try {
      const response = await fetch('/v1/geoint/status', { cache: 'no-store' });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const status = await response.json();
      const root = document.getElementById('gw-geoint-layer-status');
      if (!root) return;
      const fragment = document.createDocumentFragment();
      for (const [name, collector] of Object.entries(status.collectors || {})) {
        const row = document.createElement('div');
        row.className = 'gw-roster-row';
        const label = document.createElement('span');
        label.textContent = `${String(name).toUpperCase()} · ${Number(collector.active_count) || 0}`;
        const state = document.createElement('span');
        const rawState = String(collector.source_status || 'OFFLINE').toUpperCase();
        state.textContent = rawState === 'OK' ? 'LIVE' : rawState;
        state.dataset.status = state.textContent;
        row.append(label, state);
        fragment.appendChild(row);
      }
      const shadow = document.createElement('div');
      shadow.className = 'gw-roster-row';
      const shadowLabel = document.createElement('span');
      shadowLabel.textContent = `SHADOW RELATIONS · ${Number(status.shadow_relations) || 0}`;
      const shadowState = document.createElement('span');
      shadowState.textContent = status.shadow_sync_at ? 'LIVE' : 'STALE';
      shadow.append(shadowLabel, shadowState);
      fragment.appendChild(shadow);
      root.replaceChildren(fragment);
    } catch (_) {
      const root = document.getElementById('gw-geoint-layer-status');
      if (root) root.textContent = 'GEOINT STATUS · OFFLINE';
    }
  }

  async fetchContactRoster(lat, lon, radiusKm = 250) {
    try {
      const res = await fetch(`/v1/geoint/contacts?lat=${lat}&lon=${lon}&radius_km=${radiusKm}`);
      if (res.ok) {
        const roster = await res.json();
        this.renderContactRosterUI(roster);
      }
    } catch (e) {}
  }

  renderContactRosterUI(roster) {
    const rosterEl = document.getElementById('gw-contact-roster-list');
    if (!rosterEl) return;

    if (!roster || roster.total_contacts === 0) {
      const empty = document.createElement('div');
      empty.className = 'gw-contact-empty font-mono';
      empty.textContent = 'Keine Kontakte im 250km Radius';
      rosterEl.replaceChildren(empty);
      return;
    }

    const fragment = document.createDocumentFragment();
    const header = document.createElement('div');
    header.className = 'gw-roster-header font-mono';
    header.textContent = `RADAR KONTAKTE (${Number(roster.total_contacts) || 0}) · R: ${Number(roster.radius_km) || 0} km`;
    fragment.appendChild(header);

    const renderCategory = (title, items, color) => {
      if (!Array.isArray(items) || items.length === 0) return;
      const section = document.createElement('div');
      section.className = 'gw-roster-sec font-mono';
      section.style.color = color;
      section.textContent = `${title} (${items.length})`;
      fragment.appendChild(section);
      for (const it of items) {
        const row = document.createElement('div');
        row.className = 'gw-roster-row';
        row.dataset.id = String(it.entity_id || '');
        row.dataset.lat = String(Number(it.position?.lat));
        row.dataset.lon = String(Number(it.position?.lon));
        row.dataset.alt = String(Number(it.position?.alt) || 0);
        const label = document.createElement('span');
        label.className = 'gw-roster-lbl font-mono';
        label.textContent = String(it.label || it.entity_id || 'Unbenannter Kontakt');
        const distance = document.createElement('span');
        distance.className = 'gw-roster-dist font-mono';
        distance.textContent = `${Number(it.distance_km) || 0} km · ${Number(it.bearing_deg) || 0}°`;
        row.append(label, distance);
        fragment.appendChild(row);
      }
    };

    renderCategory('AIR CONTACTS', roster.air_contacts, '#00d2ff');
    renderCategory('MIL TACTICAL', roster.mil_air_contacts, '#ffaa00');
    renderCategory('MARITIME AIS', roster.sea_contacts, '#39ffd5');
    renderCategory('ORBITAL SATELLITES', roster.space_contacts, '#00ff66');
    renderCategory('CCTV CAMERAS', roster.camera_contacts, '#ffff00');
    renderCategory('HAZARDS / FIRES', roster.hazard_contacts, '#ff0055');
    renderCategory('OSINT SIGNALS', roster.osint_contacts, '#cc88ff');

    rosterEl.replaceChildren(fragment);

    rosterEl.querySelectorAll('.gw-roster-row').forEach(row => {
      row.addEventListener('click', () => {
        const lat = parseFloat(row.dataset.lat);
        const lon = parseFloat(row.dataset.lon);
        const alt = parseFloat(row.dataset.alt);
        const id = row.dataset.id;
        if (!isNaN(lat) && !isNaN(lon)) {
          this.flyTo(lat, lon, Math.max(15000, (alt || 0) + 10000), -45, 0);
          this.trackEntity(id);
        }
      });
    });
  }

  updateEntityCountHUD() {
    const totalEl = document.getElementById('gw-cnt-geoint-total');
    if (totalEl) totalEl.textContent = String(this.entities.length);
  }

  updateModeButtonUI() {
    const btn = document.getElementById('globe-mode-toggle-btn');
    if (btn) {
      btn.textContent = 'ENGINE: CESIUM 3D (ACTIVE)';
      btn.classList.add('active-3d');
      btn.disabled = true;
    }
  }

  bindUIControls() {
    const toggleBtn = document.getElementById('globe-mode-toggle-btn');
    if (toggleBtn) {
      toggleBtn.disabled = true;
    }

    const shaderSelect = document.getElementById('gw-sensor-shader-select');
    if (shaderSelect) {
      shaderSelect.onchange = (e) => this.setShader(e.target.value);
    }

    const providerSelect = document.getElementById('gw-map-provider-select');
    if (providerSelect) {
      providerSelect.onchange = (e) => this.setMapProvider(e.target.value);
    }
    const resetButton = document.getElementById('globe-reset-btn');
    if (resetButton) resetButton.onclick = () => this.currentRenderer?.resetGlobe();
    const legendSection = document.getElementById('gw-legend-section');
    const contactPanel = document.getElementById('gw-contact-hud');
    if (legendSection && contactPanel && contactPanel.parentElement !== legendSection) legendSection.appendChild(contactPanel);
    const extendedButton = document.getElementById('gw-extended-mode-btn');
    const researchButton = document.getElementById('gw-research-mode-btn');
    const setExtendedMode = enabled => {
        const view = document.getElementById('view-global-watch');
        if (!view) return;
        view.classList.toggle('gw-extended-mode', enabled);
        if (extendedButton) {
        extendedButton.setAttribute('aria-pressed', String(enabled));
          extendedButton.textContent = 'EXTENDED VIEW';
        }
        setTimeout(() => this.resize(), 180);
    };
    if (extendedButton) extendedButton.onclick = () => setExtendedMode(true);
    if (researchButton) researchButton.onclick = () => setExtendedMode(false);
    setExtendedMode(document.getElementById('view-global-watch')?.classList.contains('gw-extended-mode') !== false);
    if (!this.trackEventHandler) {
      this.trackEventHandler = event => this.trackEntity(String(event.detail?.entityID || ''));
      window.addEventListener('aethel:geo-track', this.trackEventHandler);
    }
    if (!this.focusEventHandler) {
      this.focusEventHandler = event => this.flyTo(Number(event.detail?.lat), Number(event.detail?.lon), Number(event.detail?.altitude) || 50000, -45, 0);
      window.addEventListener('aethel:geo-focus', this.focusEventHandler);
    }
  }

  resize() {
    if (this.currentRenderer) {
      this.currentRenderer.resize();
    }
  }

  destroy() {
    this.destroyed = true;
    if (this.pollInterval) {
      clearTimeout(this.pollInterval);
      this.pollInterval = null;
    }
    this.pollController?.abort();
    this.pollController = null;
    if (this.visibilityHandler) document.removeEventListener('visibilitychange', this.visibilityHandler);
    this.visibilityHandler = null;
    this.pollFunction = null;
    if (this.trackEventHandler) window.removeEventListener('aethel:geo-track', this.trackEventHandler);
    this.trackEventHandler = null;
    if (this.focusEventHandler) window.removeEventListener('aethel:geo-focus', this.focusEventHandler);
    this.focusEventHandler = null;
    if (this.sceneUnsubscribe) this.sceneUnsubscribe();
    this.sceneUnsubscribe = null;
    if (this.currentRenderer) {
      this.currentRenderer.destroy();
      this.currentRenderer = null;
    }
  }
}

export const sharedGeoManager = new GeoManager();
