/**
 * Base interface and contracts for Aethel GeoRenderers.
 * Contract for the exclusive local-bundled Cesium GEOINT renderer.
 */

export const RENDERER_MODES = Object.freeze({
  CESIUM_3D: 'CESIUM_3D',
});

export const SENSOR_SHADERS = Object.freeze({
  NORMAL: 'NORMAL',
  FLIR_WHITE_HOT: 'FLIR_WHITE_HOT',
  FLIR_IRONBOW: 'FLIR_IRONBOW',
  NVG_GREEN: 'NVG_GREEN',
  CRT_HUD: 'CRT_HUD',
});

export class IGeoRenderer {
  constructor(container, options = {}) {
    this.container = container;
    this.options = options;
    this.mode = RENDERER_MODES.CESIUM_3D;
    this.activeShader = SENSOR_SHADERS.NORMAL;
    this.onSelectEntity = options.onSelectEntity || (() => {});
  }

  async init() {
    throw new Error('init() must be implemented by renderer');
  }

  setData(regions, links, entities) {
    throw new Error('setData() must be implemented by renderer');
  }

  setLayers(layerState) {
    throw new Error('setLayers() must be implemented by renderer');
  }

  focus(lat, lon, zoomOrScale) {
    throw new Error('focus() must be implemented by renderer');
  }

  trackEntity(entityId) {
    throw new Error('trackEntity() must be implemented by renderer');
  }

  applyShader(shaderType) {
    this.activeShader = shaderType;
  }

  resize() {
    // Optional resize handler
  }

  destroy() {
    // Cleanup resources
  }

  getMode() {
    return this.mode;
  }
}
