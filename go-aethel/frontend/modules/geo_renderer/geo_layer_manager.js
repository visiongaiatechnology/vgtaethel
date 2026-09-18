import { MAP_PROVIDERS } from './geo_scene_state.js';

export class GeoLayerManager {
  constructor(viewer, sceneState) {
    this.viewer = viewer;
    this.sceneState = sceneState;
    this.activeImageryLayer = null;
    this.googleTileset = null;
    this.terrainProvider = null;
    this.qualityProfile = null;
  }

  setViewer(viewer) {
    this.viewer = viewer;
  }

  applyQualityProfile(profile, maximumNetworkRequests = 24) {
    if (!profile || !window.Cesium) return;
    this.qualityProfile = profile;
    const C = window.Cesium;
    const requestLimit = Math.max(4, Math.min(64, Math.round(Number(maximumNetworkRequests) || 24)));
    if (C.RequestScheduler) {
      C.RequestScheduler.maximumRequests = requestLimit;
      C.RequestScheduler.maximumRequestsPerServer = Math.max(2, Math.min(18, Math.ceil(requestLimit / 3)));
    }
    if (this.googleTileset) {
      this.googleTileset.maximumScreenSpaceError = profile.maximumScreenSpaceError;
      this.googleTileset.dynamicScreenSpaceError = profile.dynamicScreenSpaceError === true;
      this.googleTileset.preloadWhenHidden = false;
      this.googleTileset.preloadFlightDestinations = profile.tilesQuality === 'high' || profile.tilesQuality === 'ultra';
      this.googleTileset.skipLevelOfDetail = profile.tilesQuality !== 'ultra';
      this.googleTileset.show = this.sceneState.mapProvider === MAP_PROVIDERS.GOOGLE_3D;
    }
    this.viewer?.scene?.requestRender();
  }

  async setMapProvider(providerKey, customApiKey = '') {
    if (!this.viewer || !window.Cesium) return false;
    const C = window.Cesium;

    // 1. Remove previous imagery layer if any
    if (this.activeImageryLayer) {
      try {
        this.viewer.imageryLayers.remove(this.activeImageryLayer, true);
      } catch (_) {}
      this.activeImageryLayer = null;
    }

    // 2. Hide Google 3D tileset if active
    if (this.googleTileset) {
      this.googleTileset.show = false;
    }

    try {
      if (providerKey === MAP_PROVIDERS.GOOGLE_3D && (customApiKey || window.__GOOGLE_MAPS_API_KEY__)) {
        const key = customApiKey || window.__GOOGLE_MAPS_API_KEY__;
        C.GoogleMaps.defaultApiKey = key;
        if (!this.googleTileset) {
          this.googleTileset = await C.createGooglePhotorealistic3DTileset();
          this.viewer.scene.primitives.add(this.googleTileset);
        }
        this.googleTileset.show = true;
        if (this.qualityProfile) this.applyQualityProfile(this.qualityProfile);
        this.viewer.scene.globe.show = false;
        this.sceneState.setMapProvider(MAP_PROVIDERS.GOOGLE_3D, 'LIVE');
        return true;
      }

      this.viewer.scene.globe.show = true;

      let imageryProvider = null;

      if (providerKey === MAP_PROVIDERS.ARCGIS_SATELLITE || !providerKey) {
        // Native ArcGIS MapServer / WebMercator Tiles
        try {
          imageryProvider = await C.ArcGisMapServerImageryProvider.fromUrl(
            'https://services.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer',
            { enablePickFeatures: false }
          );
        } catch (_) {
          imageryProvider = new C.UrlTemplateImageryProvider({
            url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
            tilingScheme: new C.WebMercatorTilingScheme(),
            maximumLevel: 19,
            credit: '© Esri, Maxar, Earthstar Geographics',
          });
        }
        this.sceneState.setMapProvider(MAP_PROVIDERS.ARCGIS_SATELLITE, 'LIVE');
      } else if (providerKey === MAP_PROVIDERS.CARTO_DARK) {
        imageryProvider = new C.UrlTemplateImageryProvider({
          url: 'https://{s}.basemaps.cartocdn.com/rastertiles/dark_all/{z}/{x}/{y}.png',
          subdomains: ['a', 'b', 'c', 'd'],
          tilingScheme: new C.WebMercatorTilingScheme(),
          maximumLevel: 19,
          credit: '© CARTO, © OpenStreetMap contributors',
        });
        this.sceneState.setMapProvider(MAP_PROVIDERS.CARTO_DARK, 'LIVE');
      } else if (providerKey === MAP_PROVIDERS.CARTO_VOYAGER) {
        imageryProvider = new C.UrlTemplateImageryProvider({
          url: 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}@2x.png',
          subdomains: ['a', 'b', 'c', 'd'],
          tilingScheme: new C.WebMercatorTilingScheme(),
          maximumLevel: 19,
          credit: '© CARTO, © OpenStreetMap contributors',
        });
        this.sceneState.setMapProvider(MAP_PROVIDERS.CARTO_VOYAGER, 'LIVE');
      } else if (providerKey === MAP_PROVIDERS.CESIUM_ION && C.Ion.defaultAccessToken) {
        imageryProvider = await C.createWorldImageryAsync({
          style: C.IonWorldImageryStyle.AERIAL_WITH_LABELS,
        });
        this.sceneState.setMapProvider(MAP_PROVIDERS.CESIUM_ION, 'LIVE');
      } else {
        imageryProvider = new C.UrlTemplateImageryProvider({
          url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
          tilingScheme: new C.WebMercatorTilingScheme(),
          maximumLevel: 19,
        });
        this.sceneState.setMapProvider(MAP_PROVIDERS.ARCGIS_SATELLITE, 'FALLBACK');
      }

      if (imageryProvider) {
        this.activeImageryLayer = this.viewer.imageryLayers.addImageryProvider(imageryProvider, 0);
      }

      this.initTerrain();
      this.viewer.scene.requestRender();
      return true;
    } catch (err) {
      console.warn('[MAP_STACK] Provider init error:', err);
      try {
        const fallback = new C.UrlTemplateImageryProvider({
          url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
          tilingScheme: new C.WebMercatorTilingScheme(),
          maximumLevel: 19,
        });
        this.activeImageryLayer = this.viewer.imageryLayers.addImageryProvider(fallback, 0);
        this.sceneState.setMapProvider(MAP_PROVIDERS.ARCGIS_SATELLITE, 'FALLBACK');
        this.viewer.scene.requestRender();
      } catch (_) {}
      return false;
    }
  }

  async initTerrain() {
    if (!this.viewer || !window.Cesium) return;
    const C = window.Cesium;

    // Use EllipsoidTerrainProvider for rock-solid stability
    try {
      this.terrainProvider = new C.EllipsoidTerrainProvider();
      this.viewer.terrainProvider = this.terrainProvider;
    } catch (_) {}
  }

  destroy() {
    if (this.viewer && this.activeImageryLayer) {
      try {
        this.viewer.imageryLayers.remove(this.activeImageryLayer, true);
      } catch (_) {}
      this.activeImageryLayer = null;
    }
    if (this.viewer && this.googleTileset) {
      try {
        this.viewer.scene.primitives.remove(this.googleTileset);
      } catch (_) {}
      this.googleTileset = null;
    }
  }
}
