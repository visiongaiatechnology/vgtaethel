// STATUS: DIAMANT VGT SUPREME
// Cesium command globe for SHADOW. All vectors remain evidence-bound.

import { generateArcPositions } from './geo_renderer/arc_geometry.js';

const MAX_LINKS = 80;
const MAX_REGIONS = 160;

function finiteCoordinate(value, minimum, maximum) {
    const numeric = Number(value);
    return Number.isFinite(numeric) && numeric >= minimum && numeric <= maximum ? numeric : null;
}

function regionColor(region) {
    if (region.conflict_level === 'WAR') return '#ff244d';
    if (region.conflict_level === 'ESCALATION') return '#ff7a2f';
    if (region.conflict_level === 'TENSION') return '#e6c84c';
    return '#38c878';
}

function linkColor(link) {
    if (String(link.action).includes('CYBER')) return '#b84dff';
    if (link.action === 'MILITARY_SUPPORT') return '#55beff';
    return '#ff4557';
}

function stableLinkID(link) {
    return `shadow-link:${String(link.attacker_name)}:${String(link.target_name)}:${String(link.action)}`;
}

export class ShadowCommandGlobe {
    constructor(container, onSelect) {
        this.container = container;
        this.host = container.querySelector('#shadow-conflict-globe');
        this.onSelect = typeof onSelect === 'function' ? onSelect : () => {};
        this.viewer = null;
        this.screenHandler = null;
        this.regions = [];
        this.links = [];
        this.regionEntities = new Map();
        this.linkEntities = new Map();
        this.timer = null;
        this.initializing = null;
    }

    setData(regions, links) {
        this.regions = Array.isArray(regions)
            ? regions.filter(region => Array.isArray(region.evidence_ids) && region.evidence_ids.length > 0).slice(0, MAX_REGIONS)
            : [];
        this.links = Array.isArray(links)
            ? links.filter(link => Array.isArray(link.evidence_ids) && link.evidence_ids.length > 0).slice(0, MAX_LINKS)
            : [];
        if (this.viewer) this.sync();
    }

    start() {
        if (this.timer) return;
        const tick = async () => {
            this.timer = null;
            if (!this.container.isConnected) return;
            const visible = !document.hidden && !this.container.closest('.hidden');
            if (visible) {
                await this.ensureViewer();
                this.viewer?.resize();
                this.viewer?.scene?.requestRender();
            }
            this.timer = window.setTimeout(tick, visible ? 1000 : 4000);
        };
        this.timer = window.setTimeout(tick, 0);
    }

    stop() {
        if (this.timer) window.clearTimeout(this.timer);
        this.timer = null;
    }

    async ensureViewer() {
        if (this.viewer) return true;
        if (this.initializing) return this.initializing;
        this.initializing = this.initializeViewer();
        try { return await this.initializing; } finally { this.initializing = null; }
    }

    async initializeViewer() {
        const C = window.Cesium;
        if (!C || !this.host) return false;
        try {
            this.viewer = new C.Viewer(this.host, {
                animation: false, baseLayerPicker: false, baseLayer: false, fullscreenButton: false,
                geocoder: false, homeButton: false, infoBox: false, sceneModePicker: false,
                selectionIndicator: false, timeline: false, navigationHelpButton: false,
                scene3DOnly: true, requestRenderMode: true,
                maximumRenderTimeChange: Number.POSITIVE_INFINITY,
            });
            const scene = this.viewer.scene;
            scene.globe.baseColor = C.Color.fromCssColorString('#080704');
            scene.globe.enableLighting = false;
            scene.globe.maximumScreenSpaceError = 3;
            scene.highDynamicRange = true;
            try {
                const provider = await C.ArcGisMapServerImageryProvider.fromUrl(
                    'https://services.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer',
                    { enablePickFeatures: false },
                );
                const layer = this.viewer.imageryLayers.addImageryProvider(provider);
                layer.brightness = 0.56;
                layer.contrast = 1.16;
                layer.saturation = 0.42;
                layer.gamma = 0.86;
            } catch (error) {
                window.runtime?.LogWarning?.(`[SHADOW_CESIUM] imagery degraded: ${String(error)}`);
            }
            this.viewer.camera.setView({
                destination: C.Cartesian3.fromDegrees(15, 28, 12500000),
                orientation: { heading: 0, pitch: C.Math.toRadians(-90), roll: 0 },
            });
            this.screenHandler = new C.ScreenSpaceEventHandler(scene.canvas);
            this.screenHandler.setInputAction(movement => {
                const picked = scene.pick(movement.position);
                const selection = picked?.id?._shadowSelection;
                if (!selection) return;
                this.onSelect(selection);
                const value = selection.value;
                const lat = selection.type === 'link'
                    ? (Number(value.attacker_latitude) + Number(value.target_latitude)) / 2
                    : Number(value.latitude);
                const lon = selection.type === 'link'
                    ? (Number(value.attacker_longitude) + Number(value.target_longitude)) / 2
                    : Number(value.longitude);
                this.viewer.camera.flyTo({ destination: C.Cartesian3.fromDegrees(lon, lat, 850000), duration: 1.25 });
            }, C.ScreenSpaceEventType.LEFT_CLICK);
            this.sync();
            return true;
        } catch (error) {
            window.runtime?.LogError?.(`[SHADOW_CESIUM] ${error instanceof Error ? error.message : String(error)}`);
            this.destroy();
            return false;
        }
    }

    sync() {
        if (!this.viewer || !window.Cesium) return;
        this.syncRegions();
        this.syncLinks();
        this.viewer.scene.requestRender();
    }

    syncRegions() {
        const C = window.Cesium;
        const active = new Set();
        for (const region of this.regions) {
            const lat = finiteCoordinate(region.latitude, -90, 90);
            const lon = finiteCoordinate(region.longitude, -180, 180);
            if (lat == null || lon == null || (lat === 0 && lon === 0)) continue;
            const id = `shadow-region:${String(region.region_id || region.region_name)}`;
            active.add(id);
            const color = C.Color.fromCssColorString(regionColor(region));
            let entity = this.regionEntities.get(id);
            if (!entity) {
                const radius = region.conflict_level === 'WAR' ? 420000 : region.conflict_level === 'ESCALATION' ? 300000 : 210000;
                entity = this.viewer.entities.add({
                    id,
                    position: C.Cartesian3.fromDegrees(lon, lat, 12000),
                    ellipse: { semiMajorAxis: radius, semiMinorAxis: radius, material: color.withAlpha(0.18), outline: true, outlineColor: color.withAlpha(0.8) },
                    point: { pixelSize: 9, color, outlineColor: C.Color.WHITE, outlineWidth: 1.5 },
                    label: {
                        text: `${region.region_name} · ${region.conflict_level}`,
                        font: '600 10px ui-monospace, Consolas, monospace', fillColor: color,
                        outlineColor: C.Color.BLACK, outlineWidth: 1, showBackground: true,
                        backgroundColor: C.Color.fromCssColorString('rgba(5,6,4,.93)'),
                        backgroundPadding: new C.Cartesian2(7, 4), pixelOffset: new C.Cartesian2(0, -17),
                        distanceDisplayCondition: new C.DistanceDisplayCondition(0, 5000000),
                    },
                });
                this.regionEntities.set(id, entity);
            }
            entity._shadowSelection = { type: 'region', value: region };
        }
        for (const [id, entity] of this.regionEntities) {
            if (!active.has(id)) { this.viewer.entities.remove(entity); this.regionEntities.delete(id); }
        }
    }

    syncLinks() {
        const C = window.Cesium;
        const active = new Set();
        for (const link of this.links) {
            const origin = { lat: finiteCoordinate(link.attacker_latitude, -90, 90), lon: finiteCoordinate(link.attacker_longitude, -180, 180) };
            const target = { lat: finiteCoordinate(link.target_latitude, -90, 90), lon: finiteCoordinate(link.target_longitude, -180, 180) };
            if (origin.lat == null || origin.lon == null || target.lat == null || target.lon == null) continue;
            const id = stableLinkID(link);
            active.add(id);
            const color = C.Color.fromCssColorString(linkColor(link));
            let entity = this.linkEntities.get(id);
            if (!entity) {
                entity = this.viewer.entities.add({
                    id,
                    polyline: {
                        positions: generateArcPositions(C, origin, target, 42, 520000), width: 3,
                        material: new C.PolylineArrowMaterialProperty(color.withAlpha(0.9)),
                        distanceDisplayCondition: new C.DistanceDisplayCondition(100000, 18000000),
                    },
                });
                this.linkEntities.set(id, entity);
            }
            entity._shadowSelection = { type: 'link', value: link };
        }
        for (const [id, entity] of this.linkEntities) {
            if (!active.has(id)) { this.viewer.entities.remove(entity); this.linkEntities.delete(id); }
        }
    }

    destroy() {
        this.stop();
        if (this.screenHandler && !this.screenHandler.isDestroyed()) this.screenHandler.destroy();
        this.screenHandler = null;
        if (this.viewer && !this.viewer.isDestroyed()) this.viewer.destroy();
        this.viewer = null;
        this.regionEntities.clear();
        this.linkEntities.clear();
    }
}
