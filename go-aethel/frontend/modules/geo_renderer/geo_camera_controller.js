import { CAMERA_SCALES } from './geo_scene_state.js';

export const REGION_PRESETS = Object.freeze({
  cologne: { lat: 50.9375, lon: 6.9603, altitude: 25000, pitch: -35, heading: 15 },
  nrw: { lat: 51.4332, lon: 7.6616, altitude: 120000, pitch: -50, heading: 0 },
  germany: { lat: 51.1657, lon: 10.4515, altitude: 850000, pitch: -65, heading: 0 },
  europe: { lat: 50.0, lon: 15.0, altitude: 2800000, pitch: -75, heading: 0 },
  middle_east: { lat: 31.0, lon: 45.0, altitude: 3200000, pitch: -70, heading: 0 },
  austin: { lat: 30.2672, lon: -97.7431, altitude: 25000, pitch: -35, heading: 15 },
  san_francisco: { lat: 37.7749, lon: -122.4194, altitude: 30000, pitch: -35, heading: 30 },
  new_york: { lat: 40.7484, lon: -73.9857, altitude: 35000, pitch: -35, heading: -20 },
  taiwan_strait: { lat: 24.0, lon: 120.0, altitude: 600000, pitch: -55, heading: 0 },
  hormuz: { lat: 26.5667, lon: 56.25, altitude: 450000, pitch: -50, heading: 0 },
  global: { lat: 30.0, lon: 10.0, altitude: 14000000, pitch: -90, heading: 0 },
});

export class GeoCameraController {
  constructor(viewer, sceneState) {
    this.viewer = viewer;
    this.sceneState = sceneState;
    this.isTracking = false;
    this.trackedEntity = null;
    this.orbitInterval = null;
  }

  setViewer(viewer) {
    this.viewer = viewer;
  }

  focus(lat, lon, altitude = 50000, duration = 1.5) {
    if (!this.viewer || !window.Cesium) return;
    const C = window.Cesium;

    // Derive optimal perspective pitch based on altitude
    const pitchDeg = altitude < 30000 ? -30 : altitude < 150000 ? -45 : altitude < 1000000 ? -65 : -90;

    this.viewer.camera.flyTo({
      destination: C.Cartesian3.fromDegrees(lon, lat, altitude),
      orientation: {
        heading: this.viewer.camera.heading || 0,
        pitch: C.Math.toRadians(pitchDeg),
        roll: 0.0,
      },
      duration: duration,
      easingFunction: C.EasingFunction.CUBIC_IN_OUT,
    });

    this.sceneState.updateCamera(lat, lon, altitude, undefined, pitchDeg);
  }

  flyTo(lat, lon, altitude, pitch = -35, heading = 0, duration = 2.0) {
    if (!this.viewer || !window.Cesium) return;
    const C = window.Cesium;

    this.viewer.camera.flyTo({
      destination: C.Cartesian3.fromDegrees(lon, lat, altitude),
      orientation: {
        heading: C.Math.toRadians(heading),
        pitch: C.Math.toRadians(pitch),
        roll: 0.0,
      },
      duration: duration,
      easingFunction: C.EasingFunction.CUBIC_IN_OUT,
    });

    this.sceneState.updateCamera(lat, lon, altitude, heading, pitch);
  }

  focusRegion(regionKey, duration = 2.0) {
    const preset = REGION_PRESETS[regionKey.toLowerCase()] || REGION_PRESETS.global;
    this.flyTo(preset.lat, preset.lon, preset.altitude, preset.pitch, preset.heading, duration);
  }

  track(entity) {
    if (!this.viewer || !window.Cesium) return;
    this.isTracking = true;
    this.trackedEntity = entity;
    this.sceneState.setTrackedEntity(entity ? entity.id : null);

    if (!entity || !entity._cesiumEntity) {
      this.viewer.trackedEntity = undefined;
      return;
    }

    this.viewer.trackedEntity = entity._cesiumEntity;
  }

  stopTracking() {
    this.isTracking = false;
    this.trackedEntity = null;
    this.sceneState.setTrackedEntity(null);
    if (this.viewer) {
      this.viewer.trackedEntity = undefined;
    }
  }

  zoomIn(factor = 0.5) {
    if (!this.viewer || !window.Cesium) return;
    const curAlt = this.getCurrentAltitude();
    const newAlt = Math.max(300, curAlt * factor);
    const carto = this.getCurrentCartographic();
    this.focus(carto.lat, carto.lon, newAlt, 0.8);
  }

  zoomOut(factor = 2.0) {
    if (!this.viewer || !window.Cesium) return;
    const curAlt = this.getCurrentAltitude();
    const newAlt = Math.min(25000000, curAlt * factor);
    const carto = this.getCurrentCartographic();
    this.focus(carto.lat, carto.lon, newAlt, 0.8);
  }

  resetGlobe() {
    this.focusRegion('global', 2.0);
  }

  getCurrentAltitude() {
    if (!this.viewer || !this.viewer.camera) return 5000000;
    return this.viewer.camera.positionCartographic.height;
  }

  getCurrentCartographic() {
    if (!this.viewer || !this.viewer.camera || !window.Cesium) {
      return { lat: 50.9375, lon: 6.9603, height: 1500000 };
    }
    const C = window.Cesium;
    const carto = this.viewer.camera.positionCartographic;
    return {
      lat: C.Math.toDegrees(carto.latitude),
      lon: C.Math.toDegrees(carto.longitude),
      height: carto.height,
    };
  }

  destroy() {
    if (this.orbitInterval) clearInterval(this.orbitInterval);
    this.orbitInterval = null;
    this.stopTracking();
    this.viewer = null;
  }
}
