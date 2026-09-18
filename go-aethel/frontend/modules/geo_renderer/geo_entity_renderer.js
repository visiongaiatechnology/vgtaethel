import { getGlyphImage, classifyEntityGlyph } from './aircraft_icons.js';
import { cachedRiskMarkers, activeFeedEvents, cableData, citiesData, cameraData, satData, localAtlasBorders, localAtlasVersion } from '../osint/state.js';
import { isEarthquakeEvent, parseMagnitudeFromEvent, magnitudeBandColor, isVolcanoEvent } from '../osint/hazards.js';
import { generateArcPositions } from './arc_geometry.js';

const DEFAULT_RENDER_OPTIONS = Object.freeze({
  quality: Object.freeze({
    maxVisibleEntities: 2500,
    trailPoints: 16,
    trailDensity: 2,
    entityDrawDistance: 20000000,
    labelDistance: 500000,
    relationDistance: 12000000,
  }),
  trails: true,
  relations: true,
  budgetLevel: 0,
  visibility: Object.freeze({
    riskOverlayMinAltitude: 3000000,
    satelliteMinAltitude: 2200000,
    aircraftMaxAltitude: 1800000,
    vesselMaxAltitude: 2200000,
    cameraMaxAltitude: 450000,
    hazardMaxAltitude: 5500000,
    cityMaxAltitude: 2500000,
    relationMinAltitude: 120000,
    relationMaxAltitude: 16000000,
  }),
});

/**
 * GeoEntityRenderer:
 * High-performance 3D rendering of all VGT Aethel GEOINT & OSINT layers in Cesium:
 * - Dynamic GeoEntities: Aircraft, Mil-Air, Maritime AIS Vessels, Satellites, CCTV Cameras, FIRMS Fires
 * - Static & Live OSINT Layers: World Cities, Subsea Cables, AI Regional Risks, Intelligence Relations, Earthquakes, Volcanoes, News Events
 * - Dynamic LOD & Distance-Culling: Satellites visible ONLY from space (> 1200km); local details fade in on deep zoom.
 * - High-Contrast Uniform Badges: Large, readable labels with dark background pills.
 */
export class GeoEntityRenderer {
  constructor(viewer, sceneState) {
    this.viewer = viewer;
    this.sceneState = sceneState;
    this.entityMap = new Map();
    this.riskEntityMap = new Map();
    this.hazardEntityMap = new Map();
    this.infraEntityMap = new Map();
    this.relationEntityMap = new Map();
    this.options = { ...DEFAULT_RENDER_OPTIONS };
    this.cameraHeightM = 4000000;
    this.satelliteLODVisible = true;
    this.borderAtlasVersion = -1;
  }

  setViewer(viewer) {
    this.viewer = viewer;
    this.clear();
  }

  setOptions(next = {}) {
    if (!next || typeof next !== 'object') return;
    const quality = next.quality && typeof next.quality === 'object'
      ? { ...this.options.quality, ...next.quality }
      : this.options.quality;
    this.options = {
      quality,
      trails: typeof next.trails === 'boolean' ? next.trails : this.options.trails,
      relations: typeof next.relations === 'boolean' ? next.relations : this.options.relations,
      budgetLevel: Number.isFinite(Number(next.budgetLevel))
        ? Math.max(0, Math.min(3, Math.round(Number(next.budgetLevel))))
        : this.options.budgetLevel,
      visibility: next.visibility && typeof next.visibility === 'object'
        ? { ...this.options.visibility, ...next.visibility }
        : this.options.visibility,
    };
    this.applyCameraLOD(this.cameraHeightM);
  }

  visibleCount() {
    const maps = [this.entityMap, this.riskEntityMap, this.hazardEntityMap, this.infraEntityMap, this.relationEntityMap];
    let count = 0;
    for (const map of maps) {
      for (const entity of map.values()) {
        if (entity && typeof entity === 'object' && entity.show !== false) count += 1;
      }
    }
    return count;
  }

  applyCameraLOD(heightM) {
    this.cameraHeightM = Math.max(0, Number(heightM) || 0);
    const limits = this.options.visibility || DEFAULT_RENDER_OPTIONS.visibility;
    const withinMaximum = maximum => this.cameraHeightM <= Math.max(0, Number(maximum) || 0);
    const satelliteThreshold = Math.max(100000, Number(limits.satelliteMinAltitude) || 2200000);
    const satelliteHysteresis = satelliteThreshold * 0.08;
    this.satelliteLODVisible = this.satelliteLODVisible
      ? this.cameraHeightM >= satelliteThreshold - satelliteHysteresis
      : this.cameraHeightM >= satelliteThreshold + satelliteHysteresis;
    for (const entity of this.entityMap.values()) {
      const type = String(entity?._aethelEntity?.type || '');
      if (type === 'SATELLITE') entity.show = this.satelliteLODVisible && this.cameraHeightM <= 50000000;
      else if (type === 'CAMERA') entity.show = withinMaximum(limits.cameraMaxAltitude);
      else if (type === 'VESSEL') entity.show = withinMaximum(limits.vesselMaxAltitude);
      else if (type === 'AIRCRAFT' || type === 'MIL_AIRCRAFT') entity.show = withinMaximum(limits.aircraftMaxAltitude);
      else if (['CONFLICT', 'CYBER', 'POLITICAL', 'ECONOMIC', 'ENERGY'].includes(type)) {
        entity.show = this.cameraHeightM >= Number(limits.relationMinAltitude) && this.cameraHeightM <= Number(limits.relationMaxAltitude);
      }
    }
    for (const entity of this.riskEntityMap.values()) {
      entity.show = this.cameraHeightM >= Number(limits.riskOverlayMinAltitude);
    }
    for (const entity of this.hazardEntityMap.values()) entity.show = withinMaximum(limits.hazardMaxAltitude);
    for (const [id, entity] of this.infraEntityMap.entries()) {
      if (id.startsWith('infra:city:') && entity && typeof entity === 'object') entity.show = withinMaximum(limits.cityMaxAltitude);
    }
    const relationVisible = this.cameraHeightM >= Number(limits.relationMinAltitude)
      && this.cameraHeightM <= Number(limits.relationMaxAltitude);
    for (const entity of this.relationEntityMap.values()) entity.show = relationVisible;
    this.viewer?.scene?.requestRender();
  }

  sync(entities = [], layerVisibility = {}, relations = []) {
    if (!this.viewer || !window.Cesium) return;

    const filters = this.sceneState?.filters || {};
    const typeFilter = new Set(Array.isArray(filters.entityTypes) ? filters.entityTypes : []);
    const minimumConfidence = Math.max(0, Math.min(100, Number(filters.minimumConfidence) || 0));
    const cutoff = Date.now() - Math.max(1, Math.min(168, Number(this.sceneState?.timeWindowHours) || 72)) * 3600000;
    const quality = this.options.quality || DEFAULT_RENDER_OPTIONS.quality;
    const budgetFactor = [1, 0.72, 0.48, 0.28][this.options.budgetLevel] || 1;
    const maximum = Math.max(100, Math.round((Number(quality.maxVisibleEntities) || 2500) * budgetFactor));
    const eligibleEntities = entities
      .filter(entity => entity && entity.id && entity.position)
      .filter(entity => typeFilter.size === 0 || typeFilter.has(String(entity.type)))
      .filter(entity => Number(entity.confidence || 0) >= minimumConfidence)
      .filter(entity => {
        const timestamp = Date.parse(entity.timestamp || entity.updated_at || '');
        return !Number.isFinite(timestamp) || timestamp >= cutoff;
      })
      .sort((left, right) => Number(right.confidence || 0) - Number(left.confidence || 0));
    const visibleEntities = eligibleEntities.filter(entity => entity.type !== 'SATELLITE').slice(0, maximum);
    const visibleSatellites = eligibleEntities
      .filter(entity => entity.type === 'SATELLITE')
      .sort((left, right) => String(left.id).localeCompare(String(right.id)))
      .slice(0, maximum);

    // 1. Dynamic Live Entities (Aircraft, Mil, Vessels, Live Fires)
    this.syncGeoEntities(visibleEntities, layerVisibility);

    // 2. Satellites (Orbit tracks & space vehicles with space-only LOD)
    this.syncSatellites(visibleSatellites, layerVisibility);

    // 3. CCTV Surveillance Cameras (Static & Live mesh)
    this.syncCameras(visibleEntities, layerVisibility);

    // 4. AI Regional Risk Polygons & Centroid Pins
    this.syncRegionalRisks(layerVisibility);

    // 5. Epistemic Intelligence Relations & Evidence Arcs
    this.syncRelations(relations || [], layerVisibility);

    // 6. Earthquakes & Volcanoes
    this.syncHazards(layerVisibility);

    // 7. Submarine Undersea Cables & Strategic World Cities
    this.syncInfrastructure(layerVisibility);
    this.applyCameraLOD(this.viewer.camera?.positionCartographic?.height ?? this.cameraHeightM);
  }

  syncGeoEntities(entities, layerVisibility) {
    const C = window.Cesium;
    const currentIDs = new Set();
    const entityDistance = Number(this.options.quality?.entityDrawDistance) || 20000000;
    const labelDistance = Number(this.options.quality?.labelDistance) || 500000;
    const limits = this.options.visibility || DEFAULT_RENDER_OPTIONS.visibility;

    for (const ent of entities) {
      if (!ent.position) continue;

      let isVisible = true;
      let color = C.Color.CYAN;
      let glyphKind = classifyEntityGlyph(ent);
      let imgUri = getGlyphImage(glyphKind, '#00d2ff', 48).src;
      let labelText = ent.label || 'CONTACT';

      // Skip satellites here (handled with space-only LOD in syncSatellites)
      if (ent.type === 'SATELLITE') continue;
      // Skip cameras here (handled in syncCameras)
      if (ent.type === 'CAMERA') continue;
      // Natural hazards have one authoritative renderer; drawing them here duplicates labels.
      if (ent.type === 'EARTHQUAKE' || ent.type === 'VOLCANO') continue;

      switch (ent.type) {
        case 'AIRCRAFT':
          isVisible = layerVisibility.aircraft !== false;
          color = C.Color.fromCssColorString('#00d2ff');
          imgUri = getGlyphImage(glyphKind, '#00d2ff', 48).src;
          labelText = `✈ ${ent.label || ent.id}`;
          break;
        case 'MIL_AIRCRAFT':
          isVisible = layerVisibility.military !== false && layerVisibility.aircraft !== false;
          color = C.Color.fromCssColorString('#ffaa00');
          imgUri = getGlyphImage(glyphKind, '#ffaa00', 48).src;
          labelText = `◆ ${ent.label || ent.id}`;
          break;
        case 'VESSEL':
          isVisible = layerVisibility.vessels !== false;
          color = C.Color.fromCssColorString('#39ffd5');
          imgUri = getGlyphImage('ship', '#39ffd5', 40).src;
          labelText = `🚢 ${ent.label}`;
          break;
        case 'FIRE':
          isVisible = layerVisibility.fires !== false;
          color = C.Color.ORANGE;
          labelText = `🔥 ${ent.label || 'FIRE HOTSPOT'}`;
          break;
        case 'CONFLICT':
          isVisible = layerVisibility.risks !== false;
          color = C.Color.fromCssColorString(ent.classification === 'WAR' ? '#ff244d' : ent.classification === 'ESCALATION' ? '#ff7a2f' : '#e6c84c');
          imgUri = getGlyphImage('alert', ent.classification === 'WAR' ? '#ff244d' : '#e6c84c', 46).src;
          labelText = `◆ ${ent.label || 'CONFLICT'} · ${ent.classification || ent.assessment_status || 'ASSESSED'}`;
          break;
        case 'CYBER':
          isVisible = layerVisibility.events !== false;
          color = C.Color.fromCssColorString('#b84dff');
          imgUri = getGlyphImage('alert', '#b84dff', 44).src;
          labelText = `⌁ ${ent.label || 'CYBER'} · ${ent.assessment_status || 'ASSESSED'}`;
          break;
        case 'POLITICAL':
        case 'ECONOMIC':
        case 'ENERGY':
          isVisible = layerVisibility.events !== false;
          color = C.Color.fromCssColorString(ent.type === 'ENERGY' ? '#ffb020' : ent.type === 'ECONOMIC' ? '#ffdc78' : '#62a7ff');
          imgUri = getGlyphImage('alert', ent.type === 'ENERGY' ? '#ffb020' : '#62a7ff', 40).src;
          labelText = `${ent.type} · ${ent.label || ent.id}`;
          break;
        default:
          isVisible = true;
      }

      if (!isVisible) {
        if (this.entityMap.has(ent.id)) {
          try { this.viewer.entities.remove(this.entityMap.get(ent.id)); } catch (_) {}
          this.entityMap.delete(ent.id);
        }
        continue;
      }

      currentIDs.add(ent.id);

      const lat = Number(ent.position.lat);
      const lon = Number(ent.position.lon);
      const alt = Math.max(0, Number(ent.position.alt || 0));
      const position = C.Cartesian3.fromDegrees(lon, lat, alt);

      const heading = Number(ent.position.heading || 0);
      const headingRad = C.Math.toRadians(360 - heading);
      const typeDrawDistance = ent.type === 'VESSEL' ? Number(limits.vesselMaxAltitude)
        : ['AIRCRAFT', 'MIL_AIRCRAFT'].includes(ent.type) ? Number(limits.aircraftMaxAltitude)
          : ['CONFLICT', 'CYBER', 'POLITICAL', 'ECONOMIC', 'ENERGY'].includes(ent.type) ? Number(limits.relationMaxAltitude)
          : entityDistance;
      const effectiveEntityDistance = Math.min(entityDistance, typeDrawDistance || entityDistance);
      const routeActive = ['AIRCRAFT', 'MIL_AIRCRAFT'].includes(ent.type)
        && (this.sceneState?.selectedEntity?.id === ent.id || this.sceneState?.trackedEntityId === ent.id);
      const effectiveLabelDistance = routeActive ? labelDistance : Math.min(labelDistance, 180000);
      let routePositions = [];
      if (routeActive && this.options.trails && Array.isArray(ent.trail) && ent.trail.length > 1) {
        const maximumTrailPoints = Math.max(0, Math.min(30, Number(this.options.quality?.trailPoints) || 0));
        const density = Math.max(1, Math.min(8, Number(this.options.quality?.trailDensity) || 1));
        const boundedTrail = ent.trail.slice(-maximumTrailPoints * density).filter((_, index) => index % density === 0).slice(-maximumTrailPoints);
        routePositions = boundedTrail.map(point => C.Cartesian3.fromDegrees(Number(point.lon), Number(point.lat), Number(point.alt || alt)));
        routePositions.push(position);
      }

      let cEntity = this.entityMap.get(ent.id);

      if (!cEntity) {
        const entityConfig = {
          id: ent.id,
          name: ent.label,
          position: position,
          billboard: {
            image: imgUri,
            width: ent.type === 'MIL_AIRCRAFT' ? 30 : 26,
            height: ent.type === 'MIL_AIRCRAFT' ? 30 : 26,
            rotation: headingRad,
            alignedAxis: C.Cartesian3.UNIT_Z,
            scaleByDistance: new C.NearFarScalar(5.0e3, 1.2, 8.0e6, 0.45),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, effectiveEntityDistance),
          },
          label: {
            text: labelText,
            font: '600 10px ui-monospace, SFMono-Regular, Consolas, monospace',
            fillColor: color,
            outlineColor: C.Color.fromCssColorString('#02060c'),
            outlineWidth: 1,
            showBackground: true,
            backgroundColor: C.Color.fromCssColorString('rgba(2, 8, 16, 0.92)'),
            backgroundPadding: new C.Cartesian2(7, 4),
            style: C.LabelStyle.FILL_AND_OUTLINE,
            verticalOrigin: C.VerticalOrigin.BOTTOM,
            pixelOffset: new C.Cartesian2(0, -18),
            scaleByDistance: new C.NearFarScalar(10000, 1.0, effectiveLabelDistance, 0.72),
            translucencyByDistance: new C.NearFarScalar(10000, 1.0, effectiveLabelDistance, 0.45),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, effectiveLabelDistance),
          },
        };

        if (routePositions.length > 1) {
          entityConfig.polyline = {
            positions: routePositions,
            width: ent.type === 'MIL_AIRCRAFT' ? 2.0 : 1.2,
            material: color.withAlpha(0.55),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, Math.min(effectiveEntityDistance, labelDistance * 3)),
          };
        }
        if (ent.type === 'CONFLICT') {
          const radius = ent.classification === 'WAR' ? 420000 : ent.classification === 'ESCALATION' ? 300000 : 210000;
          entityConfig.ellipse = {
            semiMajorAxis: radius, semiMinorAxis: radius,
            material: color.withAlpha(0.14), outline: true, outlineColor: color.withAlpha(0.72),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, Number(limits.relationMaxAltitude)),
          };
        }

        try {
          cEntity = this.viewer.entities.add(entityConfig);
          cEntity._aethelEntity = ent;
          ent._cesiumEntity = cEntity;
          this.entityMap.set(ent.id, cEntity);
        } catch (_) {}
      } else {
        cEntity.position = position;
        if (cEntity.billboard) {
          cEntity.billboard.rotation = headingRad;
          cEntity.billboard.distanceDisplayCondition = new C.DistanceDisplayCondition(0, effectiveEntityDistance);
        }
        if (cEntity.label) {
          cEntity.label.distanceDisplayCondition = new C.DistanceDisplayCondition(0, effectiveLabelDistance);
          cEntity.label.scaleByDistance = new C.NearFarScalar(10000, 1.0, effectiveLabelDistance, 0.72);
        }
        if (routePositions.length > 1) {
          if (!cEntity.polyline) cEntity.polyline = new C.PolylineGraphics();
          cEntity.polyline.show = true;
          cEntity.polyline.positions = routePositions;
          cEntity.polyline.width = ent.type === 'MIL_AIRCRAFT' ? 2.0 : 1.2;
          cEntity.polyline.material = color.withAlpha(0.55);
          cEntity.polyline.distanceDisplayCondition = new C.DistanceDisplayCondition(0, Math.min(effectiveEntityDistance, labelDistance * 3));
        } else if (cEntity.polyline) {
          cEntity.polyline.show = false;
        }
        cEntity._aethelEntity = ent;
        ent._cesiumEntity = cEntity;
      }
    }

    for (const [id, cEnt] of this.entityMap.entries()) {
      if (!id.startsWith('sat:') && !id.startsWith('cam:') && !currentIDs.has(id)) {
        try { this.viewer.entities.remove(cEnt); } catch (_) {}
        this.entityMap.delete(id);
      }
    }
  }

  syncSatellites(entities, layerVisibility) {
    const C = window.Cesium;
    const isVisible = layerVisibility.satellites !== false;
    const currentSatIDs = new Set();

    if (!isVisible) {
      for (const [id, ent] of this.entityMap.entries()) {
        if (id.startsWith('sat:')) {
          try { this.viewer.entities.remove(ent); } catch (_) {}
          this.entityMap.delete(id);
        }
      }
      return;
    }

    const liveSats = entities.filter(e => e.type === 'SATELLITE');
    const allSats = [...liveSats];

    satData.forEach(s => {
      const sid = `sat:${s.name}`;
      if (!allSats.some(e => e.id === sid || e.label === s.name)) {
        allSats.push({
          id: sid,
          type: 'SATELLITE',
          label: s.name,
          position: { lat: s.lat, lon: s.lon, alt: s.alt || 550000 }
        });
      }
    });

    for (const sat of allSats) {
      if (!sat.position) continue;
      const sid = sat.id.startsWith('sat:') ? sat.id : `sat:${sat.id}`;
      currentSatIDs.add(sid);

      const lat = Number(sat.position.lat);
      const lon = Number(sat.position.lon);
      const alt = Number(sat.position.alt || 500000);
      const position = C.Cartesian3.fromDegrees(lon, lat, alt);
      const selected = this.sceneState?.selectedEntity?.id === sat.id
        || this.sceneState?.selectedEntity?.id === sid
        || this.sceneState?.trackedEntityId === sat.id
        || this.sceneState?.trackedEntityId === sid;

      let cSat = this.entityMap.get(sid);

      if (!cSat) {
        const isISS = sat.label && sat.label.includes('ISS');
        const imgUri = getGlyphImage('satellite', isISS ? '#00ffaa' : '#00ff66', 36).src;

        const config = {
          id: sid,
          name: sat.label,
          position: position,
          billboard: {
            image: imgUri,
            width: isISS ? 34 : 26,
            height: isISS ? 34 : 26,
            scaleByDistance: new C.NearFarScalar(2.0e6, 1.0, 3.0e7, 0.4),
            // POINT 4 FIX: Satellites ONLY visible when zoomed OUT in space (> 1,200 km)
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, 45000000),
          },
          label: {
            show: selected,
            text: `🛰 ${sat.label} (${Math.round(alt / 1000)}km)`,
            font: 'bold 12px monospace',
            fillColor: C.Color.fromCssColorString(isISS ? '#00ffaa' : '#00ff66'),
            outlineColor: C.Color.BLACK,
            outlineWidth: 3,
            showBackground: true,
            backgroundColor: C.Color.fromCssColorString('rgba(4, 10, 24, 0.85)'),
            backgroundPadding: new C.Cartesian2(6, 3),
            style: C.LabelStyle.FILL_AND_OUTLINE,
            verticalOrigin: C.VerticalOrigin.BOTTOM,
            pixelOffset: new C.Cartesian2(0, -14),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, 18000000),
          },
        };

        try {
          cSat = this.viewer.entities.add(config);
          cSat._aethelEntity = sat;
          sat._cesiumEntity = cSat;
          this.entityMap.set(sid, cSat);
        } catch (_) {}
      } else {
        cSat.position = position;
        if (cSat.label) cSat.label.show = selected;
        cSat._aethelEntity = sat;
      }
    }
    for (const [id, entity] of this.entityMap.entries()) {
      if (id.startsWith('sat:') && !currentSatIDs.has(id)) {
        try { this.viewer.entities.remove(entity); } catch (_) {}
        this.entityMap.delete(id);
      }
    }
  }

  syncCameras(entities, layerVisibility) {
    const C = window.Cesium;
    const maximumAltitude = Number(this.options.visibility?.cameraMaxAltitude) || 450000;
    const entityDrawDistance = Number(this.options.quality?.entityDrawDistance) || 20000000;
    const isVisible = (layerVisibility.cameras !== false) && (layerVisibility.cctv !== false);
    const currentCamIDs = new Set();

    if (!isVisible) {
      for (const [id, ent] of this.entityMap.entries()) {
        if (id.startsWith('cam:')) {
          try { this.viewer.entities.remove(ent); } catch (_) {}
          this.entityMap.delete(id);
        }
      }
      return;
    }

    const liveCams = entities.filter(e => e.type === 'CAMERA');
    const allCams = [...liveCams];

    cameraData.forEach((c, idx) => {
      const cid = `cam:static:${idx}`;
      if (!allCams.some(e => e.id === cid || e.label === c.name)) {
        allCams.push({
          id: cid,
          type: 'CAMERA',
          label: c.name,
          position: { lat: c.lat, lon: c.lon, alt: 10 },
          stream: c.stream || ''
        });
      }
    });

    for (const cam of allCams) {
      if (!cam.position) continue;
      const cid = cam.id.startsWith('cam:') ? cam.id : `cam:${cam.id}`;
      currentCamIDs.add(cid);

      const lat = Number(cam.position.lat);
      const lon = Number(cam.position.lon);
      const position = C.Cartesian3.fromDegrees(lon, lat, 15);

      let cCam = this.entityMap.get(cid);

      if (!cCam) {
        const imgUri = getGlyphImage('camera', '#ffff00', 36).src;
        const config = {
          id: cid,
          name: cam.label,
          position: position,
          billboard: {
            image: imgUri,
            width: 26,
            height: 26,
            scaleByDistance: new C.NearFarScalar(5.0e3, 1.2, 6.0e6, 0.4),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, entityDrawDistance),
          },
          label: {
            text: `📷 ${cam.label}`,
            font: 'bold 12px monospace',
            fillColor: C.Color.fromCssColorString('#ffff00'),
            outlineColor: C.Color.BLACK,
            outlineWidth: 3,
            showBackground: true,
            backgroundColor: C.Color.fromCssColorString('rgba(4, 10, 24, 0.85)'),
            backgroundPadding: new C.Cartesian2(6, 3),
            style: C.LabelStyle.FILL_AND_OUTLINE,
            verticalOrigin: C.VerticalOrigin.BOTTOM,
            pixelOffset: new C.Cartesian2(0, -12),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, Math.min(maximumAltitude, 250000)),
          },
        };

        try {
          cCam = this.viewer.entities.add(config);
          cCam._aethelEntity = cam;
          cam._cesiumEntity = cCam;
          this.entityMap.set(cid, cCam);
        } catch (_) {}
      } else {
        cCam.position = position;
        if (cCam.billboard) cCam.billboard.distanceDisplayCondition = new C.DistanceDisplayCondition(0, entityDrawDistance);
        cCam._aethelEntity = cam;
      }
    }
    for (const [id, entity] of this.entityMap.entries()) {
      if (id.startsWith('cam:') && !currentCamIDs.has(id)) {
        try { this.viewer.entities.remove(entity); } catch (_) {}
        this.entityMap.delete(id);
      }
    }
  }

  syncRegionalRisks(layerVisibility) {
    const C = window.Cesium;
    const isVisible = layerVisibility.risks !== false;
    const currentRiskIDs = new Set();

    if (!isVisible) {
      for (const ent of this.riskEntityMap.values()) {
        try { this.viewer.entities.remove(ent); } catch (_) {}
      }
      this.riskEntityMap.clear();
      return;
    }

    const markers = cachedRiskMarkers || [];
    for (const m of markers) {
      const riskId = `risk:${m.id || m.name || m.lat + '_' + m.lon}`;
      currentRiskIDs.add(riskId);

      const risk = Math.max(0, Math.min(100, Number(m.overall_risk) || 0));
      const hexColor = risk > 60 ? '#ff004f' : risk > 25 ? '#ff7b00' : '#34d399';
      const cColor = C.Color.fromCssColorString(hexColor);

      let cRisk = this.riskEntityMap.get(riskId);

      if (!cRisk) {
        const config = {
          id: riskId,
          name: `${m.name || m.id} (Risk: ${Math.round(risk)}%)`,
        };

        if (m.lat != null && m.lon != null) {
          config.position = C.Cartesian3.fromDegrees(Number(m.lon), Number(m.lat), 15000);
          config.point = {
            pixelSize: 10,
            color: cColor,
            outlineColor: C.Color.WHITE,
            outlineWidth: 2,
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, 18000000),
          };
          config.label = {
            text: `RISK · ${m.name || m.id} · ${Math.round(risk)}%`,
            font: '600 10px ui-monospace, SFMono-Regular, Consolas, monospace',
            fillColor: cColor,
            outlineColor: C.Color.BLACK,
            outlineWidth: 1,
            showBackground: true,
            backgroundColor: C.Color.fromCssColorString('rgba(3, 8, 14, 0.92)'),
            backgroundPadding: new C.Cartesian2(7, 4),
            style: C.LabelStyle.FILL_AND_OUTLINE,
            verticalOrigin: C.VerticalOrigin.BOTTOM,
            pixelOffset: new C.Cartesian2(0, -14),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, 14000000),
          };
        }

        if (m.lat != null && m.lon != null) {
          const radius = 220000 + risk * 6500;
          config.ellipse = {
            semiMajorAxis: radius,
            semiMinorAxis: radius * 0.72,
            material: cColor.withAlpha(0.12),
            outline: true,
            outlineColor: cColor.withAlpha(0.58),
          };
        }

        try {
          cRisk = this.viewer.entities.add(config);
          cRisk._aethelEntity = {
            id: riskId, type: 'REGIONAL_RISK', label: m.name || m.id || 'Regional risk',
            position: { lat: Number(m.lat), lon: Number(m.lon), alt: 15000 },
            confidence: Number(m.confidence || 0), assessment_status: 'ASSESSED',
            metadata: { assessment: m.assessment || m.driver || '' },
          };
          this.riskEntityMap.set(riskId, cRisk);
        } catch (_) {}
      } else {
        if (cRisk.point) cRisk.point.color = cColor;
        if (cRisk.label) {
          cRisk.label.text = `RISK · ${m.name || m.id} · ${Math.round(risk)}%`;
          cRisk.label.fillColor = cColor;
        }
        if (cRisk.ellipse) {
          cRisk.ellipse.material = cColor.withAlpha(0.12);
          cRisk.ellipse.outlineColor = cColor.withAlpha(0.58);
        }
      }
    }

    for (const [id, ent] of this.riskEntityMap.entries()) {
      if (!currentRiskIDs.has(id)) {
        try { this.viewer.entities.remove(ent); } catch (_) {}
        this.riskEntityMap.delete(id);
      }
    }
  }

  syncRelations(relations = [], layerVisibility = {}) {
    const C = window.Cesium;
    const isVisible = layerVisibility.risks !== false && this.options.relations !== false;
    const currentRelationIDs = new Set();

    if (!isVisible) {
      for (const [id, ent] of this.relationEntityMap.entries()) {
        try { this.viewer.entities.remove(ent); } catch (_) {}
      }
      this.relationEntityMap.clear();
      return;
    }

    const categoryFilter = new Set(Array.isArray(this.sceneState?.filters?.relationCategories) ? this.sceneState.filters.relationCategories : []);
    const minimumConfidence = Math.max(0, Math.min(100, Number(this.sceneState?.filters?.minimumConfidence) || 0));
    const maximumDistance = Math.min(Number(this.options.quality?.relationDistance) || 12000000, Number(this.options.visibility?.relationMaxAltitude) || 16000000);
    for (const relation of relations) {
      if (!relation || !relation.id || !Array.isArray(relation.evidence_ids) || relation.evidence_ids.length === 0) continue;
      if (categoryFilter.size > 0 && !categoryFilter.has(String(relation.category))) continue;
      if (Number(relation.confidence || 0) < minimumConfidence) continue;
      const relId = `rel:${relation.id}`;
      currentRelationIDs.add(relId);

      const src = relation.origin;
      const tgt = relation.target;
      if (!src || !tgt || src.lat == null || tgt.lat == null) continue;

      let cRel = this.relationEntityMap.get(relId);

      if (!cRel) {
        const arcPositions = generateArcPositions(C, src, tgt, 36, 450000);
        const midLat = (Number(src.lat) + Number(tgt.lat)) / 2;
        const midLon = (Number(src.lon) + Number(tgt.lon)) / 2;
        const colorHex = relation.category === 'CYBER' ? '#b84dff'
          : relation.category === 'ECONOMIC' ? '#ffdc78'
            : relation.category === 'SUPPORT' ? '#00f0ff'
              : '#ff004f';
        const color = C.Color.fromCssColorString(colorHex);
        const relationLabel = `${relation.origin_name || 'ORIGIN'} → ${relation.target_name || 'TARGET'} · ${relation.action || relation.category}`;

        const config = {
          id: relId,
          name: relationLabel,
          polyline: {
            positions: arcPositions,
            width: 3.5,
            material: new C.PolylineArrowMaterialProperty(color.withAlpha(0.85)),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, maximumDistance),
          },
        };

        try {
          cRel = this.viewer.entities.add(config);
          cRel._aethelRelation = relation;
          cRel._evidence_ids = relation.evidence_ids || [];
          cRel._assessment_status = relation.assessment_status || 'ESTIMATED';

          const tagId = `${relId}:tag`;
          const tag = this.viewer.entities.add({
            id: tagId,
            position: C.Cartesian3.fromDegrees(midLon, midLat, 450000),
            point: {
              pixelSize: 8,
              color: color,
              outlineColor: C.Color.WHITE,
              outlineWidth: 2,
              distanceDisplayCondition: new C.DistanceDisplayCondition(0, 14000000),
            },
            label: {
              text: `${relation.category === 'CYBER' ? '⌁' : '◆'} ${relation.action || relation.category} · ${Number(relation.confidence) || 0}%`,
              font: '600 10px ui-monospace, SFMono-Regular, Consolas, monospace',
              fillColor: color,
              outlineColor: C.Color.BLACK,
              outlineWidth: 1,
              showBackground: true,
              backgroundColor: C.Color.fromCssColorString('rgba(5, 6, 10, 0.94)'),
              backgroundPadding: new C.Cartesian2(7, 4),
              style: C.LabelStyle.FILL_AND_OUTLINE,
              verticalOrigin: C.VerticalOrigin.BOTTOM,
              pixelOffset: new C.Cartesian2(0, -10),
              distanceDisplayCondition: new C.DistanceDisplayCondition(0, Math.min(maximumDistance, 1800000)),
            },
          });
          tag._aethelRelation = relation;
          tag._evidence_ids = relation.evidence_ids || [];
          tag._assessment_status = relation.assessment_status || 'ESTIMATED';

          this.relationEntityMap.set(tagId, tag);
          this.relationEntityMap.set(relId, cRel);
        } catch (_) {}
      }
    }

    for (const [id, ent] of this.relationEntityMap.entries()) {
      if (!currentRelationIDs.has(id) && !id.endsWith(':tag')) {
        try { this.viewer.entities.remove(ent); } catch (_) {}
        this.relationEntityMap.delete(id);
        const tagID = `${id}:tag`;
        const tag = this.relationEntityMap.get(tagID);
        if (tag) {
          try { this.viewer.entities.remove(tag); } catch (_) {}
          this.relationEntityMap.delete(tagID);
        }
      }
    }
  }

  generateArc(Cesium, p1, p2, numPoints = 30, maxAltM = 450000) {
    return generateArcPositions(Cesium, p1, p2, numPoints, maxAltM);
  }

  syncHazards(layerVisibility) {
    const C = window.Cesium;
    const maximumAltitude = Number(this.options.visibility?.hazardMaxAltitude) || 5500000;
    const events = Array.isArray(window.__gwVisibleEvents) && window.__gwVisibleEvents.length > 0
      ? window.__gwVisibleEvents
      : activeFeedEvents || [];
    const currentHazardIDs = new Set();
    const earthquakeCutoff = Date.now() - 24 * 3600000;

    for (const ev of events) {
      const lat = ev.lat != null ? Number(ev.lat) : ev.latitude != null ? Number(ev.latitude) : null;
      const lon = ev.lon != null ? Number(ev.lon) : ev.longitude != null ? Number(ev.longitude) : null;
      if (lat == null || lon == null || !Number.isFinite(lat) || !Number.isFinite(lon)) continue;

      const isQuake = isEarthquakeEvent(ev);
      const isVolcano = isVolcanoEvent(ev);
      if (isQuake) {
        const eventTime = Date.parse(ev.published_at || ev.timestamp || ev.date || '');
        if (Number.isFinite(eventTime) && eventTime < earthquakeCutoff) continue;
      }

      if (isQuake && layerVisibility.earthquakes === false) continue;
      if (isVolcano && layerVisibility.volcanoes === false) continue;
      if (!isQuake && !isVolcano) continue;

      const hid = `hazard:${ev.id || `${lat}_${lon}`}`;
      currentHazardIDs.add(hid);

      let cHazard = this.hazardEntityMap.get(hid);

      if (!cHazard) {
        const mag = isQuake ? parseMagnitudeFromEvent(ev) : null;
        const col = isQuake ? magnitudeBandColor(mag) : { hex: '#f43f5e', band: 'red' };
        const cColor = C.Color.fromCssColorString(col.hex);
        const radius = isQuake ? Math.max(7, (mag || 3) * 2.8) : 9;

        const config = {
          id: hid,
          name: ev.title || (isQuake ? 'Earthquake' : 'Volcano'),
          position: C.Cartesian3.fromDegrees(lon, lat, 0),
          point: {
            pixelSize: radius,
            color: cColor,
            outlineColor: C.Color.WHITE,
            outlineWidth: 1.5,
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, maximumAltitude),
          },
          label: {
            text: isQuake ? `M${mag || '?'} · ${ev.title || 'Earthquake'}` : `🌋 ${ev.title || 'Volcano'}`,
            font: 'bold 12px monospace',
            fillColor: cColor,
            outlineColor: C.Color.BLACK,
            outlineWidth: 3,
            showBackground: true,
            backgroundColor: C.Color.fromCssColorString('rgba(4, 10, 24, 0.85)'),
            backgroundPadding: new C.Cartesian2(6, 3),
            style: C.LabelStyle.FILL_AND_OUTLINE,
            verticalOrigin: C.VerticalOrigin.BOTTOM,
            pixelOffset: new C.Cartesian2(0, -12),
            distanceDisplayCondition: new C.DistanceDisplayCondition(0, Math.min(maximumAltitude, 1200000)),
          },
        };

        try {
          cHazard = this.viewer.entities.add(config);
          cHazard._aethelEntity = {
            id: hid, type: isQuake ? 'EARTHQUAKE' : 'VOLCANO', label: ev.title || (isQuake ? 'Earthquake' : 'Volcano'),
            position: { lat, lon, alt: 0 }, timestamp: ev.published_at || ev.timestamp || ev.date,
            source: ev.source || ev.source_name || (isQuake ? 'USGS' : 'NASA EONET'),
            assessment_status: 'OBSERVED', confidence: Number(ev.confidence || 80),
            metadata: { assessment: ev.summary || '' },
          };
          this.hazardEntityMap.set(hid, cHazard);
        } catch (_) {}
      }
    }

    for (const [id, ent] of this.hazardEntityMap.entries()) {
      if (!currentHazardIDs.has(id)) {
        try { this.viewer.entities.remove(ent); } catch (_) {}
        this.hazardEntityMap.delete(id);
      }
    }
  }

  syncInfrastructure(layerVisibility) {
    const C = window.Cesium;
    const cityMaximumAltitude = Number(this.options.visibility?.cityMaxAltitude) || 2500000;

    // Self-hosted country outlines remain independent from the imagery provider.
    const bordersVisible = layerVisibility.borders !== false;
    if (bordersVisible && this.borderAtlasVersion !== localAtlasVersion && Array.isArray(localAtlasBorders) && localAtlasBorders.length) {
      for (const [id, ent] of this.infraEntityMap.entries()) {
        if (id.startsWith('infra:border:')) {
          try { this.viewer.entities.remove(ent); } catch (_) {}
          this.infraEntityMap.delete(id);
        }
      }
      let borderIndex = 0;
      localAtlasBorders.forEach(polygon => polygon.forEach(ring => {
        if (!Array.isArray(ring) || ring.length < 2) return;
        const coordinates = [];
        ring.forEach(point => {
          const lon = Number(point?.[0]);
          const lat = Number(point?.[1]);
          if (Number.isFinite(lon) && Number.isFinite(lat)) coordinates.push(lon, lat, 1400);
        });
        if (coordinates.length < 6) return;
        const borderId = `infra:border:${borderIndex++}`;
        try {
          const border = this.viewer.entities.add({
            id: borderId,
            polyline: {
              positions: C.Cartesian3.fromDegreesArrayHeights(coordinates),
              width: 1.35,
              material: C.Color.fromCssColorString('#72d9ff').withAlpha(0.68),
              arcType: C.ArcType.GEODESIC,
              distanceDisplayCondition: new C.DistanceDisplayCondition(0, 18000000),
            },
          });
          this.infraEntityMap.set(borderId, border);
        } catch (_) {}
      }));
      this.borderAtlasVersion = localAtlasVersion;
    } else if (!bordersVisible) {
      for (const [id, ent] of this.infraEntityMap.entries()) {
        if (id.startsWith('infra:border:')) {
          try { this.viewer.entities.remove(ent); } catch (_) {}
          this.infraEntityMap.delete(id);
        }
      }
      this.borderAtlasVersion = -1;
    }

    // 1. Undersea Submarine Fiber Optic Cables
    const cablesVisible = layerVisibility.cables !== false;
    const cableId = 'infra:cables';
    if (cablesVisible && !this.infraEntityMap.has(cableId) && Array.isArray(cableData)) {
      cableData.forEach((line, idx) => {
        const lineId = `infra:cable:${idx}`;
        const flat = [];
        line.forEach(pt => flat.push(Number(pt[0]), Number(pt[1])));
        try {
          const cableEnt = this.viewer.entities.add({
            id: lineId,
            polyline: {
              positions: C.Cartesian3.fromDegreesArray(flat),
              width: 2.0,
              material: C.Color.fromCssColorString('#ff00ff').withAlpha(0.65),
              distanceDisplayCondition: new C.DistanceDisplayCondition(0, 20000000),
            },
          });
          this.infraEntityMap.set(lineId, cableEnt);
        } catch (_) {}
      });
      this.infraEntityMap.set(cableId, true);
    } else if (!cablesVisible && this.infraEntityMap.has(cableId)) {
      for (const [id, ent] of this.infraEntityMap.entries()) {
        if (id.startsWith('infra:cable:')) {
          try { this.viewer.entities.remove(ent); } catch (_) {}
          this.infraEntityMap.delete(id);
        }
      }
      this.infraEntityMap.delete(cableId);
    }

    // 2. Strategic World Cities & Geopolitical Capitals
    const citiesVisible = layerVisibility.cities !== false;
    const citiesKey = 'infra:cities';
    if (citiesVisible && !this.infraEntityMap.has(citiesKey) && Array.isArray(citiesData)) {
      citiesData.forEach((city, idx) => {
        const cityId = `infra:city:${idx}`;
        try {
          const cityEnt = this.viewer.entities.add({
            id: cityId,
            name: city.name,
            position: C.Cartesian3.fromDegrees(Number(city.lon), Number(city.lat), 0),
            point: {
              pixelSize: city.isCapital ? 6 : 4.5,
              color: C.Color.fromCssColorString('#ffdc78'),
              outlineColor: C.Color.BLACK,
              outlineWidth: 1.5,
              distanceDisplayCondition: new C.DistanceDisplayCondition(0, cityMaximumAltitude),
            },
            label: {
              text: `📍 ${city.name}`,
              font: 'bold 12px sans-serif',
              fillColor: C.Color.fromCssColorString('#ffdc78'),
              outlineColor: C.Color.BLACK,
              outlineWidth: 3,
              showBackground: true,
              backgroundColor: C.Color.fromCssColorString('rgba(4, 10, 24, 0.85)'),
              backgroundPadding: new C.Cartesian2(6, 3),
              style: C.LabelStyle.FILL_AND_OUTLINE,
              verticalOrigin: C.VerticalOrigin.BOTTOM,
              pixelOffset: new C.Cartesian2(0, -10),
              distanceDisplayCondition: new C.DistanceDisplayCondition(0, Math.min(cityMaximumAltitude, city.isCapital ? 1200000 : 500000)),
            },
          });
          cityEnt._aethelEntity = {
            id: cityId, type: 'CITY', label: city.name,
            position: { lat: Number(city.lat), lon: Number(city.lon), alt: 0 },
            source: 'LOCAL REFERENCE', assessment_status: 'OBSERVED', confidence: 100,
          };
          this.infraEntityMap.set(cityId, cityEnt);
        } catch (_) {}
      });
      this.infraEntityMap.set(citiesKey, true);
    } else if (!citiesVisible && this.infraEntityMap.has(citiesKey)) {
      for (const [id, ent] of this.infraEntityMap.entries()) {
        if (id.startsWith('infra:city:')) {
          try { this.viewer.entities.remove(ent); } catch (_) {}
          this.infraEntityMap.delete(id);
        }
      }
      this.infraEntityMap.delete(citiesKey);
    }
  }

  clear() {
    if (this.viewer) {
      for (const cEnt of this.entityMap.values()) {
        try { this.viewer.entities.remove(cEnt); } catch (_) {}
      }
      for (const cEnt of this.riskEntityMap.values()) {
        try { this.viewer.entities.remove(cEnt); } catch (_) {}
      }
      for (const cEnt of this.hazardEntityMap.values()) {
        try { this.viewer.entities.remove(cEnt); } catch (_) {}
      }
      for (const cEnt of this.infraEntityMap.values()) {
        if (cEnt && typeof cEnt === 'object') {
          try { this.viewer.entities.remove(cEnt); } catch (_) {}
        }
      }
      for (const cEnt of this.relationEntityMap.values()) {
        try { this.viewer.entities.remove(cEnt); } catch (_) {}
      }
    }
    this.entityMap.clear();
    this.riskEntityMap.clear();
    this.hazardEntityMap.clear();
    this.infraEntityMap.clear();
    this.relationEntityMap.clear();
  }
}
