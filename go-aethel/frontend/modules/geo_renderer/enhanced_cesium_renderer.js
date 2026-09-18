import { IGeoRenderer, RENDERER_MODES, SENSOR_SHADERS } from './renderer_interface.js';
import { GeoCameraController } from './geo_camera_controller.js';
import { GeoEntityRenderer } from './geo_entity_renderer.js';
import { GeoLayerManager } from './geo_layer_manager.js';
import { MAP_PROVIDERS, sharedSceneState } from './geo_scene_state.js';
import { loadGlobalWatchPreferences } from '../global_watch_preferences.js';
import { normalizeGeoQualityProfile } from './quality_profiles.js';
import { RenderGovernor } from './render_governor.js';
import { GeoPerformanceController } from './performance_controller.js';
import { loadLocalWorldAtlas } from '../osint/texture_atlas.js';

/**
 * EnhancedCesiumRenderer:
 * High-fidelity 3D Geospatial Engine (God's Eye View Architecture).
 * Enables seamless transition from Planetary orbit down to building-scale 3D inspection (< 1km).
 */
export class EnhancedCesiumRenderer extends IGeoRenderer {
  constructor(container, options = {}) {
    super(container, options);
    this.mode = RENDERER_MODES.CESIUM_3D;
    this.viewer = null;
    this.stageDiv = null;
    this.hudEl = null;
    this.entities = [];
    this.relations = [];
    this.sceneState = options.sceneState || sharedSceneState;

    this.cameraController = new GeoCameraController(null, this.sceneState);
    this.entityRenderer = new GeoEntityRenderer(null, this.sceneState);
    this.layerManager = new GeoLayerManager(null, this.sceneState);

    this.activeShader = SENSOR_SHADERS.NORMAL;
    this.postProcessStage = null;
    this.telemetryInterval = null;
    this.screenHandler = null;
    this.renderGovernor = null;
    this.performanceController = null;
    this.lifecycleCleanup = [];
    this.budgetLevel = 0;
    this.lastInitializationError = '';
    this.qualityProfile = this.resolveQualityProfile();
  }

  async init() {
    await loadLocalWorldAtlas(() => this.renderGovernor?.request('world-atlas'));
    // 1. Prepare the exclusive Cesium viewport.
    this.stageDiv = document.getElementById('cesium-3d-stage');
    if (!this.stageDiv) {
      this.stageDiv = document.createElement('div');
      this.stageDiv.id = 'cesium-3d-stage';
      this.stageDiv.className = 'cesium-stage-container';
      this.stageDiv.style.cssText = 'position:absolute; inset:0; width:100%; height:100%; background:#020409; border-radius:6px; overflow:hidden; z-index:2;';
      this.container.appendChild(this.stageDiv);
    } else {
      this.stageDiv.style.display = 'block';
      this.stageDiv.innerHTML = '';
    }

    // 2. Mount Telemetry HUD
    this.hudEl = document.createElement('div');
    this.hudEl.className = 'cesium-tactical-hud font-mono';
    this.hudEl.style.cssText = 'position:absolute; top:12px; left:16px; font-size:9px; color:var(--vgt-cyan); pointer-events:none; z-index:10; display:flex; flex-direction:column; gap:4px;';
    this.hudEl.innerHTML = `
      <div style="display:flex; gap:16px; align-items:center; background:rgba(4,10,24,0.85); padding:5px 10px; border-radius:4px; border:1px solid rgba(0,240,255,0.3); box-shadow:0 0 12px rgba(0,0,0,0.8);">
        <span style="font-weight:700; color:#fff; letter-spacing:0.1em;">◈ AETHEL ENHANCED 3D // GEOINT</span>
        <span id="cesium-hud-telemetry" style="color:var(--vgt-text-dim);">ALT: 4,500 km · SCALE: CONTINENT</span>
        <span id="cesium-hud-provider" style="color:#00ff66;">PROVIDER: ARCGIS SATELLITE (LIVE)</span>
        <span id="cesium-hud-shader" style="color:#ffaa00;">SENSOR: NORMAL</span>
      </div>
    `;
    this.stageDiv.appendChild(this.hudEl);

    // 3. Ensure Cesium JS is available
    await this.ensureCesium();

    const C = window.Cesium;
    if (!C) {
      console.error('[CESIUM] Cesium library unavailable');
      return false;
    }

    try {
      this.viewer = new C.Viewer(this.stageDiv, {
        animation: false,
        baseLayerPicker: false,
        fullscreenButton: false,
        geocoder: false,
        homeButton: false,
        infoBox: false,
        sceneModePicker: false,
        selectionIndicator: false,
        timeline: false,
        navigationHelpButton: false,
        scene3DOnly: true,
        baseLayer: false,
        requestRenderMode: true,
        maximumRenderTimeChange: Number.POSITIVE_INFINITY,
      });

      // Suppress blocking modal error panel and log gracefully instead
      if (this.viewer.cesiumWidget) {
        this.viewer.cesiumWidget.showErrorPanel = (title, message, error) => {
          console.warn('[CESIUM_NON_FATAL_WARNING]', title, message, error);
        };
      }

      const scene = this.viewer.scene;
      this.renderGovernor = new RenderGovernor(scene);
      this.applyQualityProfile(this.qualityProfile);
      scene.globe.show = true;
      scene.globe.enableLighting = false;
      scene.globe.baseColor = C.Color.fromCssColorString('#0a1a35');
      scene.globe.depthTestAgainstTerrain = false;
      scene.highDynamicRange = true;
      if (scene.globe) scene.globe.atmosphereBrightnessShift = 0.12;

      // Connect sub-managers
      this.cameraController.setViewer(this.viewer);
      this.entityRenderer.setViewer(this.viewer);
      this.layerManager.setViewer(this.viewer);

      // Restore the persisted provider; the layer manager performs a safe ArcGIS fallback.
      await this.layerManager.setMapProvider(this.sceneState.mapProvider || MAP_PROVIDERS.ARCGIS_SATELLITE);

      // Initial camera positioning: Europe / Germany overview, centered, facing Earth
      const cam = this.sceneState.camera;
      const initialLat = cam.lat || 50.9375;
      const initialLon = cam.lon || 6.9603;
      const initialAlt = Math.min(6000000, Math.max(1000000, cam.altitude || 4000000));
      const initialPitch = -85;

      this.viewer.camera.setView({
        destination: C.Cartesian3.fromDegrees(initialLon, initialLat, initialAlt),
        orientation: {
          heading: C.Math.toRadians(cam.heading || 0),
          pitch: C.Math.toRadians(initialPitch),
          roll: 0.0,
        },
      });

      // Handle Entity Click
      this.screenHandler = new C.ScreenSpaceEventHandler(scene.canvas);
      this.screenHandler.setInputAction((movement) => {
        const picked = scene.pick(movement.position);
        if (C.defined(picked) && picked.id && picked.id._aethelEntity) {
          const ent = picked.id._aethelEntity;
          const selected = this.entitySelection(ent);
          this.sceneState.setSelectedEntity(selected);
          this.onSelectEntity(selected);
          this.focusPickedEntity(ent);
        } else if (C.defined(picked) && picked.id && picked.id._aethelRelation) {
          const relation = picked.id._aethelRelation;
          const selected = this.relationSelection(relation);
          this.sceneState.setSelectedEntity(selected);
          this.onSelectEntity(selected);
          this.cameraController.flyTo(selected.lat, selected.lon, 700000, -60, 0, 1.2);
        }
      }, C.ScreenSpaceEventType.LEFT_CLICK);

      let releaseCameraHold = null;
      this.lifecycleCleanup.push(this.viewer.camera.moveStart.addEventListener(() => {
        if (!releaseCameraHold && this.renderGovernor) releaseCameraHold = this.renderGovernor.hold('camera');
      }));
      this.lifecycleCleanup.push(this.viewer.camera.moveEnd.addEventListener(() => {
        if (releaseCameraHold) releaseCameraHold();
        releaseCameraHold = null;
        const cartographic = this.viewer.camera.positionCartographic;
        this.sceneState.updateCamera(
          C.Math.toDegrees(cartographic.latitude), C.Math.toDegrees(cartographic.longitude), cartographic.height,
          C.Math.toDegrees(this.viewer.camera.heading), C.Math.toDegrees(this.viewer.camera.pitch), C.Math.toDegrees(this.viewer.camera.roll),
        );
        this.entityRenderer.applyCameraLOD(cartographic.height);
        this.renderGovernor?.request('camera-end');
      }));
      this.viewer.camera.percentageChanged = 0.015;
      this.lifecycleCleanup.push(this.viewer.camera.changed.addEventListener(() => {
        const height = this.viewer?.camera?.positionCartographic?.height;
        if (Number.isFinite(height)) this.entityRenderer.applyCameraLOD(height);
      }));
      const onVisibilityChange = () => {
        if (!document.hidden) this.renderGovernor?.request('document-visible');
      };
      document.addEventListener('visibilitychange', onVisibilityChange);
      this.lifecycleCleanup.push(() => document.removeEventListener('visibilitychange', onVisibilityChange));
      const onPreferences = (event) => this.applyPreferences(event.detail || loadGlobalWatchPreferences());
      window.addEventListener('aethel:global-watch-preferences', onPreferences);
      this.lifecycleCleanup.push(() => window.removeEventListener('aethel:global-watch-preferences', onPreferences));
      this.lifecycleCleanup.push(this.sceneState.subscribe((event, state) => {
        if (event === 'provider') {
          const provider = document.getElementById('cesium-hud-provider');
          if (provider) provider.textContent = `PROVIDER: ${state.mapProvider} · ${state.providerStatus}`;
        }
        if (event === 'layers' || event === 'filters' || event === 'time-window' || event === 'selection' || event === 'tracking') {
          if (this.viewer && this.viewer.scene && this.viewer.scene.globe) {
            this.viewer.scene.globe.enableLighting = false;
          }
          this.entityRenderer.sync(this.entities, state.layers, this.relations);
        }
        this.renderGovernor?.request(`scene-state:${event}`);
      }));

      this.performanceController = new GeoPerformanceController(this.viewer, (level) => {
        this.budgetLevel = level;
        this.sceneState.setQuality(this.qualityProfile.name, level, this.qualityProfile.dynamicQuality);
        this.applyQualityProfile(this.qualityProfile, level);
      });
      this.performanceController.start(this.qualityProfile);
      this.applyPreferences(loadGlobalWatchPreferences());
      this.entityRenderer.applyCameraLOD(initialAlt);

      // Start Telemetry HUD Loop
      this.startTelemetryLoop();
      this.entityRenderer.sync(this.entities, this.sceneState.layers, this.relations);
      this.renderGovernor.request('initialized');
      return true;
    } catch (err) {
      console.error('[CESIUM] Initialization failed:', err);
      this.lastInitializationError = err instanceof Error ? `${err.name}: ${err.message}` : String(err || 'Unknown initialization error');
      window.runtime?.LogError?.(`[CESIUM] ${this.lastInitializationError}`);
      this.destroy();
      return false;
    }
  }

  relationSelection(relation) {
    const latitude = (Number(relation.origin?.lat) + Number(relation.target?.lat)) / 2;
    const longitude = (Number(relation.origin?.lon) + Number(relation.target?.lon)) / 2;
    return {
      id: relation.id,
      title: `${relation.origin_name} → ${relation.target_name}`,
      summary: `${relation.action} · ${relation.assessment || 'Keine zusätzliche Bewertung.'}`,
      source: 'SHADOW OSINT',
      domain: relation.category === 'CYBER' ? 'cyber' : 'conflict',
      status: 'inference',
      assessment_status: relation.assessment_status,
      attribution_status: relation.attribution_status,
      confidence: relation.confidence,
      evidence_ids: relation.evidence_ids,
      source_ids: relation.source_ids,
      report_id: relation.report_id,
      location_precision: `${relation.origin_precision} → ${relation.target_precision}`,
      lat: Number.isFinite(latitude) ? latitude : null,
      lon: Number.isFinite(longitude) ? longitude : null,
      timestamp: relation.timestamp,
      provenance: 'shadow-evidence-adapter',
    };
  }

  entitySelection(entity) {
    const assessment = String(entity.assessment_status || 'RAW');
    return {
      id: entity.id,
      title: entity.label || entity.id,
      summary: entity.metadata?.assessment || entity.classification || 'Keine zusätzliche Bewertung.',
      source: entity.source || 'GEOINT',
      domain: String(entity.type || 'entity').toLowerCase(),
      status: ['SUPPORTED', 'ASSESSED', 'CORRELATED'].includes(assessment) ? 'inference' : assessment === 'CONFIRMED' ? 'verified' : 'raw',
      assessment_status: assessment,
      location_precision: entity.location_precision || 'UNKNOWN',
      confidence: Number(entity.confidence || 0),
      freshness: Number(entity.freshness || 0),
      evidence_ids: Array.isArray(entity.evidence) ? entity.evidence : [],
      source_ids: Array.isArray(entity.source_ids) ? entity.source_ids : [],
      lat: Number(entity.position?.lat),
      lon: Number(entity.position?.lon),
      altitude: Number(entity.position?.alt || 0),
      timestamp: entity.timestamp,
      track_id: ['AIRCRAFT', 'MIL_AIRCRAFT', 'VESSEL', 'SATELLITE'].includes(entity.type) ? entity.id : '',
      provenance: 'geoint-entity-bus',
      stream: entity.type === 'CAMERA' ? String(entity.stream || '') : '',
    };
  }

  focusPickedEntity(entity) {
    const lat = Number(entity?.position?.lat);
    const lon = Number(entity?.position?.lon);
    if (!Number.isFinite(lat) || !Number.isFinite(lon)) return;
    const objectAltitude = Math.max(0, Number(entity.position?.alt) || 0);
    const focusAltitude = entity.type === 'CAMERA' ? 9000
      : entity.type === 'VESSEL' ? 22000
        : entity.type === 'SATELLITE' ? Math.max(1500000, objectAltitude + 700000)
          : ['EARTHQUAKE', 'VOLCANO', 'FIRE'].includes(entity.type) ? 120000
            : ['AIRCRAFT', 'MIL_AIRCRAFT'].includes(entity.type) ? Math.max(55000, objectAltitude + 35000)
              : 65000;
    this.cameraController.flyTo(lat, lon, focusAltitude, focusAltitude < 150000 ? -42 : -68, 0, 1.2);
  }

  async ensureCesium() {
    if (window.Cesium) return true;
    return new Promise((resolve) => {
      let count = 0;
      const t = setInterval(() => {
        if (window.Cesium) {
          clearInterval(t);
          resolve(true);
        } else if (++count > 40) {
          clearInterval(t);
          resolve(false);
        }
      }, 100);
    });
  }

  startTelemetryLoop() {
    if (this.telemetryInterval) clearTimeout(this.telemetryInterval);
    const C = window.Cesium;

    const tick = () => {
      if (!this.viewer || !C) return;
      try {
        const cam = this.viewer.camera;
        const altM = cam.positionCartographic.height;
        const altKm = Math.round(altM / 1000);
        const lat = C.Math.toDegrees(cam.positionCartographic.latitude).toFixed(2);
        const lon = C.Math.toDegrees(cam.positionCartographic.longitude).toFixed(2);
        const scalePreset = this.sceneState.deriveScalePreset(altM);
        const performanceState = this.performanceController?.diagnostics() || { averageFPS: 0, budgetLevel: 0 };
        const visibleEntities = this.entityRenderer.visibleCount();
        const tileCount = Number(this.viewer.scene.globe?._surface?._tilesToRender?.length || 0);
        this.sceneState.diagnostics = { ...this.sceneState.diagnostics, averageFPS: performanceState.averageFPS, visibleEntities, tileCount, budgetLevel: performanceState.budgetLevel };

        const telemEl = document.getElementById('cesium-hud-telemetry');
        const mapCoordsEl = document.getElementById('gw-map-coords');
        const altStr = altKm >= 1 ? `${altKm.toLocaleString()} km` : `${Math.round(altM)} m`;
        if (telemEl) {
          telemEl.textContent = `LAT: ${lat}° · LON: ${lon}° · ALT: ${altStr} · SCALE: ${scalePreset} · FPS: ${performanceState.averageFPS.toFixed(0)} · VISIBLE: ${visibleEntities}/${this.entities.length} · TILES: ${tileCount} · BUDGET: ${performanceState.budgetLevel}`;
        }
        if (mapCoordsEl) mapCoordsEl.textContent = `LAT ${lat}° · LON ${lon}° · SCALE ${scalePreset} · ALT ${altStr}`;
      } catch (_) {}
      if (this.viewer) this.telemetryInterval = setTimeout(tick, document.hidden ? 5000 : 1000);
    };
    this.telemetryInterval = setTimeout(tick, 1000);
  }

  setData(regions, links, entities) {
    if (Array.isArray(entities)) {
      this.entities = entities;
    }
    this.relations = Array.isArray(links) ? links : [];
    this.entityRenderer.sync(this.entities, this.sceneState.layers, this.relations);
    this.renderGovernor?.request('data-sync');
  }

  setLayers(layerState) {
    if (layerState && typeof layerState === 'object') {
      for (const [k, v] of Object.entries(layerState)) {
        this.sceneState.setLayer(k, v);
      }
      this.entityRenderer.sync(this.entities, this.sceneState.layers, this.relations);
      this.renderGovernor?.request('layer-change');
    }
  }

  // Camera commands
  focus(lat, lon, zoomOrScale) {
    let alt = 50000;
    if (zoomOrScale) {
      const z = Number(zoomOrScale);
      alt = z > 5 ? Math.max(800, 40000000 / Math.pow(2, z)) : Math.max(2000, 3000000 / z);
    }
    this.cameraController.focus(lat, lon, alt);
  }

  flyTo(lat, lon, altitude, pitch = -35, heading = 0, duration = 2.0) {
    this.cameraController.flyTo(lat, lon, altitude, pitch, heading, duration);
  }

  focusRegion(regionKey) {
    this.cameraController.focusRegion(regionKey);
  }

  trackEntity(entityId) {
    if (!entityId) {
      this.cameraController.stopTracking();
      return;
    }
    const ent = this.entities.find(e => e.id === entityId);
    if (ent) {
      this.cameraController.track(ent);
    }
  }

  zoomIn() {
    this.cameraController.zoomIn();
  }

  zoomOut() {
    this.cameraController.zoomOut();
  }

  resetGlobe() {
    this.cameraController.resetGlobe();
  }

  async setMapProvider(providerKey, apiKey = '') {
    return this.layerManager.setMapProvider(providerKey, apiKey);
  }

  applyShader(shaderType) {
    this.activeShader = shaderType;
    this.sceneState.setSensorMode(shaderType);
    const C = window.Cesium;
    if (!this.viewer || !C) return;

    const shaderTag = document.getElementById('cesium-hud-shader');
    if (shaderTag) shaderTag.textContent = `SENSOR: ${shaderType}`;

    if (this.postProcessStage) {
      try {
        this.viewer.scene.postProcessStages.remove(this.postProcessStage);
      } catch (_) {}
      this.postProcessStage = null;
    }

    if (shaderType === SENSOR_SHADERS.NORMAL || !this.qualityProfile.sensorEffects || this.qualityProfile.postProcessingLevel === 'off' || !this.qualityProfile.postProcessing || this.budgetLevel >= 2) {
      this.renderGovernor?.request('shader-disabled');
      return;
    }

    let fragmentShader = '';
    switch (shaderType) {
      case SENSOR_SHADERS.FLIR_WHITE_HOT:
        fragmentShader = `
          uniform sampler2D colorTexture;
          varying vec2 v_textureCoordinates;
          void main() {
            vec4 color = texture2D(colorTexture, v_textureCoordinates);
            float lum = dot(color.rgb, vec3(0.299, 0.587, 0.114));
            gl_FragColor = vec4(vec3(lum * 1.35), color.a);
          }`;
        break;

      case SENSOR_SHADERS.FLIR_IRONBOW:
        fragmentShader = `
          uniform sampler2D colorTexture;
          varying vec2 v_textureCoordinates;
          void main() {
            vec4 color = texture2D(colorTexture, v_textureCoordinates);
            float lum = dot(color.rgb, vec3(0.299, 0.587, 0.114));
            vec3 iron = clamp(vec3(
              sin(lum * 3.1415 - 0.2) * 1.5,
              sin(lum * 3.1415 * 0.7) * 1.2,
              cos(lum * 3.1415 * 0.5) * 1.8
            ), 0.0, 1.0);
            gl_FragColor = vec4(iron, color.a);
          }`;
        break;

      case SENSOR_SHADERS.NVG_GREEN:
        fragmentShader = `
          uniform sampler2D colorTexture;
          varying vec2 v_textureCoordinates;
          void main() {
            vec4 color = texture2D(colorTexture, v_textureCoordinates);
            float lum = dot(color.rgb, vec3(0.299, 0.587, 0.114));
            vec3 nvg = vec3(0.1, 1.0, 0.3) * lum * 1.4;
            gl_FragColor = vec4(nvg, color.a);
          }`;
        break;

      case SENSOR_SHADERS.CRT_HUD:
        fragmentShader = `
          uniform sampler2D colorTexture;
          varying vec2 v_textureCoordinates;
          void main() {
            vec4 color = texture2D(colorTexture, v_textureCoordinates);
            float scanline = sin(v_textureCoordinates.y * 750.0) * 0.08;
            vec3 crt = color.rgb - scanline;
            crt *= vec3(0.88, 1.08, 1.15);
            gl_FragColor = vec4(crt, color.a);
          }`;
        break;
    }

    if (fragmentShader) {
      try {
        this.postProcessStage = new C.PostProcessStage({ fragmentShader });
        this.viewer.scene.postProcessStages.add(this.postProcessStage);
        this.renderGovernor?.request('shader-change');
      } catch (_) {}
    }
  }

  resolveQualityProfile(preferences = loadGlobalWatchPreferences()) {
    const custom = preferences.renderQuality === 'custom';
    return normalizeGeoQualityProfile(preferences.renderQuality, {
      targetFPS: custom ? preferences.targetFPS : undefined,
      maxVisibleEntities: custom ? preferences.max3DEntities : undefined,
      atmosphere: custom ? preferences.enableAtmosphere : undefined,
      postProcessing: custom ? preferences.postProcessing : undefined,
      dynamicQuality: preferences.dynamicQuality,
      maximumScreenSpaceError: custom ? preferences.maximumScreenSpaceError : undefined,
      entityDrawDistance: custom ? preferences.entityDrawDistance : undefined,
      labelDistance: custom ? preferences.labelDrawDistance : undefined,
      modelDrawDistance: custom ? preferences.modelDrawDistance : undefined,
      trailPoints: custom ? preferences.trailLength : undefined,
      trailDensity: custom ? preferences.trailDensity : undefined,
      terrainQuality: custom ? preferences.terrainQuality : undefined,
      tilesQuality: custom ? preferences.tilesQuality : undefined,
      dynamicScreenSpaceError: custom ? preferences.dynamicScreenSpaceError : undefined,
      antiAliasing: custom ? preferences.antiAliasing : undefined,
      fog: custom ? preferences.fog : undefined,
      shadows: custom ? preferences.shadows : undefined,
      lighting: custom ? preferences.lighting : undefined,
      highResolutionTextures: custom ? preferences.highResolutionTextures : undefined,
      sensorEffects: custom ? preferences.sensorEffects : undefined,
      hudDensity: custom ? preferences.hudDensity : undefined,
      automaticLOD: preferences.automaticLOD,
      clustering: custom ? preferences.geoClustering : undefined,
      postProcessingLevel: custom ? preferences.postProcessingLevel : undefined,
    });
  }

  applyPreferences(preferences) {
    this.qualityProfile = this.resolveQualityProfile(preferences);
    this.performanceController?.updateProfile(this.qualityProfile);
    this.applyQualityProfile(this.qualityProfile, 0);
    this.entityRenderer.setOptions({
      quality: this.qualityProfile,
      trails: preferences.enableTrails !== false,
      relations: preferences.enableConflictTensions !== false,
      visibility: {
        riskOverlayMinAltitude: preferences.riskOverlayMinAltitude,
        satelliteMinAltitude: preferences.satelliteMinAltitude,
        aircraftMaxAltitude: preferences.aircraftMaxAltitude,
        vesselMaxAltitude: preferences.vesselMaxAltitude,
        cameraMaxAltitude: preferences.cameraMaxAltitude,
        hazardMaxAltitude: preferences.hazardMaxAltitude,
        cityMaxAltitude: preferences.cityMaxAltitude,
        relationMinAltitude: preferences.relationMinAltitude,
        relationMaxAltitude: preferences.relationMaxAltitude,
      },
    });
    this.layerManager.applyQualityProfile(this.qualityProfile, preferences.maximumNetworkRequests);
    this.entityRenderer.sync(this.entities, this.sceneState.layers, this.relations);
    const diagnostics = document.getElementById('cesium-hud-telemetry');
    if (diagnostics) diagnostics.style.display = preferences.diagnosticsHUD === true ? '' : 'none';
    if (this.hudEl) this.hudEl.dataset.density = this.qualityProfile.hudDensity;
    this.applyShader(this.activeShader);
    this.renderGovernor?.request('quality-preferences');
  }

  applyQualityProfile(profile, budgetLevel = 0) {
    if (!this.viewer || !profile) return;
    const scene = this.viewer.scene;
    const scalePenalty = budgetLevel * 0.12;
    this.viewer.targetFrameRate = Math.max(15, profile.targetFPS - budgetLevel * 8);
    this.viewer.resolutionScale = Math.max(0.5, profile.resolutionScale - scalePenalty);
    scene.globe.maximumScreenSpaceError = profile.maximumScreenSpaceError + budgetLevel * 1.5;
    scene.globe.enableLighting = false;
    scene.globe.showGroundAtmosphere = profile.atmosphere && budgetLevel < 2;
    scene.fog.enabled = profile.fog && profile.atmosphere && budgetLevel < 3;
    scene.shadowMap.enabled = profile.shadows && budgetLevel === 0;
    if (Number.isFinite(scene.msaaSamples)) scene.msaaSamples = profile.antiAliasing && budgetLevel < 2 ? 4 : 1;
    scene.globe.tileCacheSize = profile.highResolutionTextures && budgetLevel === 0 ? 300 : 100;
    this.entityRenderer.setOptions({ quality: profile, budgetLevel });
    this.renderGovernor?.request('quality-budget');
  }

  resize() {
    if (this.viewer && this.viewer.resize) {
      this.viewer.resize();
    }
  }

  destroy() {
    if (this.telemetryInterval) {
      clearTimeout(this.telemetryInterval);
      this.telemetryInterval = null;
    }
    this.performanceController?.stop();
    this.performanceController = null;
    if (this.screenHandler && !this.screenHandler.isDestroyed()) this.screenHandler.destroy();
    this.screenHandler = null;
    for (const cleanup of this.lifecycleCleanup.splice(0)) {
      try { if (typeof cleanup === 'function') cleanup(); } catch (_) {}
    }
    this.entityRenderer.clear();
    this.cameraController.destroy();
    this.layerManager.destroy();
    this.renderGovernor?.destroy();
    this.renderGovernor = null;
    if (this.viewer) {
      try {
        this.viewer.destroy();
      } catch (_) {}
      this.viewer = null;
    }
    if (this.stageDiv) {
      this.stageDiv.style.display = 'none';
    }
  }
}
