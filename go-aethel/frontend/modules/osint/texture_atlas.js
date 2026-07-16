import { state } from '../state.js';
import * as GlobeProjection from '../global_watch_projection.js';
import {
  localGlobeRotY, setLocalGlobeRotY,
  localGlobeRotX,
	localGlobeScale, setLocalGlobeScale,
  localGlobeDragging,
  globalWatchPreferences,
  earthTexReady, setEarthTexReady,
  earthTexSource, setEarthTexSource,
  earthSphereCache, setEarthSphereCache,
  earthImgData, setEarthImgData,
  earthImgW, setEarthImgW,
  earthImgH, setEarthImgH,
  earthOffscreen, setEarthOffscreen,
  earthTexMeta, setEarthTexMeta,
  globeIdleRotationFrame, setGlobeIdleRotationFrame,
  globeIdleRotationLastFrame, setGlobeIdleRotationLastFrame,
  globeIdleRotationBlockedUntil, setGlobeIdleRotationBlockedUntil,
  GLOBE_IDLE_ROTATION_DELAY_MS,
  GLOBE_IDLE_ROTATION_FRAME_MS,
  GLOBE_IDLE_ROTATION_RADIANS_PER_SECOND,
	GLOBE_IDLE_NORMAL_SCALE,
  EARTH_TEX_MAX_W,
  EARTH_TEX_WORK_MAX_W,
  localAtlasBorders, setLocalAtlasBorders,
  incrementLocalAtlasVersion, localAtlasVersion,
  projectedAtlasCache, setProjectedAtlasCache,
  projectedGridCache, setProjectedGridCache,
  globeContainer
} from './state.js';
import {
  globeRadius,
  projectLatLon,
  unprojectScreenToLatLon,
} from './projection.js';

export function computeEarthRasterStep(radiusPx, dragging) {
  const r = Math.max(1, Number(radiusPx) || 1);
  const quality = globalWatchPreferences.renderQuality;
  if (dragging) {
    return quality === 'performance' ? 3 : 2;
  }
  if (quality === 'performance') return 2;
  return 1;
}

export function markGlobeInteraction() {
    setGlobeIdleRotationBlockedUntil(performance.now() + GLOBE_IDLE_ROTATION_DELAY_MS);
}

export function shouldRotateGlobe(now) {
    const bounds = globeContainer?.getBoundingClientRect?.();
    return Boolean(
        globalWatchPreferences.idleRotation &&
        !localGlobeDragging &&
        globeContainer?.isConnected &&
        bounds && bounds.width > 20 && bounds.height > 20 &&
		now >= globeIdleRotationBlockedUntil
    );
}

export function runGlobeIdleRotation(now, requestRenderFn) {
    let lastFrame = globeIdleRotationLastFrame;
    if (!lastFrame) {
        lastFrame = now;
        setGlobeIdleRotationLastFrame(now);
    }
    const elapsed = Math.min(250, Math.max(0, now - lastFrame));
    if (shouldRotateGlobe(now) && elapsed >= GLOBE_IDLE_ROTATION_FRAME_MS) {
        let nextRotY = localGlobeRotY + GLOBE_IDLE_ROTATION_RADIANS_PER_SECOND * (elapsed / 1000);
        if (nextRotY > Math.PI) nextRotY -= Math.PI * 2;
        setLocalGlobeRotY(nextRotY);
		const scaleDelta = GLOBE_IDLE_NORMAL_SCALE - localGlobeScale;
		if (Math.abs(scaleDelta) > 0.001) {
			const settle = Math.min(1, elapsed / 1400);
			setLocalGlobeScale(localGlobeScale + scaleDelta * settle);
		}
        setGlobeIdleRotationLastFrame(now);
        setEarthSphereCache(null);
        if (requestRenderFn) requestRenderFn();
    } else if (!shouldRotateGlobe(now)) {
        setGlobeIdleRotationLastFrame(now);
    }
}

export function startGlobeIdleRotation(requestRenderFn) {
    if (globeIdleRotationFrame) return;
	setGlobeIdleRotationBlockedUntil(performance.now());
    setGlobeIdleRotationLastFrame(performance.now());
    const timer = window.setInterval(() => runGlobeIdleRotation(performance.now(), requestRenderFn), GLOBE_IDLE_ROTATION_FRAME_MS);
    setGlobeIdleRotationFrame(timer);
}

export function earthRasterDPRCap() {
  if (globalWatchPreferences.renderQuality === 'ultra') return 1.75;
  if (globalWatchPreferences.renderQuality === 'performance') return 1;
  return 1.25;
}

export function earthTexWorkingMaxW() {
  return EARTH_TEX_WORK_MAX_W;
}

export function buildEarthTextureFromAtlas(borders, w, h) {
    w = w || 2048;
    h = h || 1024;
    const c = document.createElement('canvas');
    c.width = w;
    c.height = h;
    const ctx = c.getContext('2d', { willReadFrequently: true });
    const ocean = ctx.createLinearGradient(0, 0, 0, h);
    ocean.addColorStop(0, '#071428');
    ocean.addColorStop(0.15, '#0a2a4a');
    ocean.addColorStop(0.5, '#0c3d5e');
    ocean.addColorStop(0.85, '#0a2a4a');
    ocean.addColorStop(1, '#071428');
    ctx.fillStyle = ocean;
    ctx.fillRect(0, 0, w, h);
    for (let i = 0; i < 6; i++) {
        ctx.fillStyle = `rgba(0, 80, 120, ${0.03 + i * 0.01})`;
        ctx.fillRect(0, h * (0.35 + i * 0.05), w, h * 0.04);
    }
    const lonLatToXY = (lon, lat) => {
        const x = ((lon + 180) / 360) * w;
        const y = ((90 - lat) / 180) * h;
        return [x, y];
    };
    (borders || []).forEach((polygon, pi) => {
        (polygon || []).forEach((ring, ri) => {
            if (!ring || ring.length < 3) return;
            ctx.beginPath();
            for (let i = 0; i < ring.length; i++) {
                const lon = ring[i][0];
                const lat = ring[i][1];
                const [x, y] = lonLatToXY(lon, lat);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            }
            ctx.closePath();
            if (ri === 0) {
                const hue = 95 + (pi % 7) * 6;
                const lit = 22 + (pi % 5) * 3;
                ctx.fillStyle = `hsl(${hue}, 38%, ${lit}%)`;
                ctx.fill('evenodd');
                ctx.strokeStyle = 'rgba(30, 70, 45, 0.45)';
                ctx.lineWidth = 0.6;
                ctx.stroke();
            } else {
                ctx.fillStyle = 'rgba(8, 40, 70, 0.85)';
                ctx.fill();
            }
        });
    });
    const iceN = ctx.createLinearGradient(0, 0, 0, h * 0.12);
    iceN.addColorStop(0, 'rgba(220, 235, 245, 0.55)');
    iceN.addColorStop(1, 'rgba(220, 235, 245, 0)');
    ctx.fillStyle = iceN;
    ctx.fillRect(0, 0, w, h * 0.12);
    const iceS = ctx.createLinearGradient(0, h, 0, h * 0.88);
    iceS.addColorStop(0, 'rgba(220, 235, 245, 0.5)');
    iceS.addColorStop(1, 'rgba(220, 235, 245, 0)');
    ctx.fillStyle = iceS;
    ctx.fillRect(0, h * 0.88, w, h * 0.12);
    ctx.fillStyle = 'rgba(0, 180, 220, 0.04)';
    ctx.fillRect(0, h * 0.42, w, h * 0.16);
    return c;
}

export function cacheEarthImageData(canvas, sourceLabel) {
    if (!canvas) return;
    const ctx = canvas.getContext('2d', { willReadFrequently: true });
    const img = ctx.getImageData(0, 0, canvas.width, canvas.height);
    setEarthImgData(img.data);
    setEarthImgW(canvas.width);
    setEarthImgH(canvas.height);
    setEarthTexReady(true);
    setEarthSphereCache(null);
    setEarthTexMeta({
        width: canvas.width,
        height: canvas.height,
        source: sourceLabel || earthTexSource || 'unknown',
        bytesHint: canvas.width * canvas.height * 4
    });
}

export function sampleEarthTexNearest(u, v) {
    if (!earthImgData || !earthImgW) return [12, 40, 70, 255];
    let uu = u - Math.floor(u);
    if (uu < 0) uu += 1;
    let vv = Math.max(0, Math.min(0.999999, v));
    const x = Math.min(earthImgW - 1, Math.floor(uu * earthImgW));
    const y = Math.min(earthImgH - 1, Math.floor(vv * earthImgH));
    const i = (y * earthImgW + x) * 4;
    return [earthImgData[i], earthImgData[i + 1], earthImgData[i + 2], earthImgData[i + 3]];
}

export function sampleEarthTex(u, v) {
    if (!earthImgData || !earthImgW) return [12, 40, 70, 255];
    let uu = u - Math.floor(u);
    if (uu < 0) uu += 1;
    let vv = Math.max(0, Math.min(1, v));
    const fx = uu * earthImgW - 0.5;
    const fy = vv * earthImgH - 0.5;
    let x0 = Math.floor(fx);
    let y0 = Math.floor(fy);
    const tx = fx - x0;
    const ty = fy - y0;
    x0 = ((x0 % earthImgW) + earthImgW) % earthImgW;
    const x1 = (x0 + 1) % earthImgW;
    y0 = Math.max(0, Math.min(earthImgH - 1, y0));
    const y1 = Math.max(0, Math.min(earthImgH - 1, y0 + 1));
    const i00 = (y0 * earthImgW + x0) * 4;
    const i10 = (y0 * earthImgW + x1) * 4;
    const i01 = (y1 * earthImgW + x0) * 4;
    const i11 = (y1 * earthImgW + x1) * 4;
    const out = [0, 0, 0, 255];
    for (let c = 0; c < 3; c++) {
        const a = earthImgData[i00 + c] * (1 - tx) + earthImgData[i10 + c] * tx;
        const b = earthImgData[i01 + c] * (1 - tx) + earthImgData[i11 + c] * tx;
        out[c] = (a * (1 - ty) + b * ty) | 0;
    }
    return out;
}

export function drawEarthGlobeTexture(ctx, cw, ch) {
    if (!earthTexReady || !earthImgData) return false;
    const r = globeRadius(localGlobeScale, cw, ch);
    const cx = cw / 2;
    const cy = ch / 2;
    const step = computeEarthRasterStep(r, localGlobeDragging);
    const rotY = localGlobeRotY;
    const rotX = localGlobeRotX;
    const sample = sampleEarthTex;

    const size = Math.ceil(r * 2) + 4;
    const cacheKey = [
        size, step, rotY.toFixed(4), rotX.toFixed(4),
        earthImgW, earthImgH, localGlobeDragging ? 'd' : 'i'
    ].join('|');
    if (earthSphereCache && earthSphereCache.key === cacheKey && earthSphereCache.canvas) {
        ctx.save();
        ctx.beginPath();
        ctx.arc(cx, cy, r, 0, Math.PI * 2);
        ctx.clip();
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = 'high';
        ctx.drawImage(earthSphereCache.canvas, cx - size / 2, cy - size / 2, size, size);
        drawEarthAtmosphereRim(ctx, cx, cy, r);
        ctx.restore();
        return true;
    }

    if (!earthOffscreen || earthOffscreen.width !== size) {
        const off = document.createElement('canvas');
        off.width = size;
        off.height = size;
        setEarthOffscreen(off);
    }
    const off = earthOffscreen;
    const octx = off.getContext('2d', { willReadFrequently: false });
    const imgOut = octx.createImageData(size, size);
    const out = imgOut.data;
    const ox = size / 2;
    const oy = size / 2;
    const originX = cx - ox;
    const originY = cy - oy;

    for (let py = 0; py < size; py += step) {
        for (let px = 0; px < size; px += step) {
            const geo = unprojectScreenToLatLon(
                originX + px, originY + py,
                rotY, rotX, localGlobeScale, cw, ch
            );
            if (!geo) continue;
            const [cr, cg, cb] = sample(geo.u, geo.v);
            const limb = 0.48 + 0.52 * geo.z2;
            const atmos = 1 + 0.08 * (1 - geo.z2);
            const R = Math.min(255, (cr * limb * atmos) | 0);
            const G = Math.min(255, (cg * limb * atmos) | 0);
            const B = Math.min(255, (cb * limb * atmos + 8 * (1 - geo.z2)) | 0);
            for (let dy = 0; dy < step && py + dy < size; dy++) {
                for (let dx = 0; dx < step && px + dx < size; dx++) {
                    const nx2 = (px + dx - ox) / r;
                    const ny2 = (oy - (py + dy)) / r;
                    if (nx2 * nx2 + ny2 * ny2 > 1.0) continue;
                    const idx = ((py + dy) * size + (px + dx)) * 4;
                    out[idx] = R;
                    out[idx + 1] = G;
                    out[idx + 2] = B;
                    out[idx + 3] = 255;
                }
            }
        }
    }
    octx.putImageData(imgOut, 0, 0);
    if (!localGlobeDragging) {
        const cacheCanvas = document.createElement('canvas');
        cacheCanvas.width = size;
        cacheCanvas.height = size;
        cacheCanvas.getContext('2d').drawImage(off, 0, 0);
        setEarthSphereCache({ key: cacheKey, canvas: cacheCanvas });
    } else {
        setEarthSphereCache(null);
    }

    ctx.save();
    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.clip();
    ctx.imageSmoothingEnabled = true;
    ctx.imageSmoothingQuality = step <= 2 ? 'high' : 'medium';
    ctx.drawImage(off, originX, originY);

    drawEarthAtmosphereRim(ctx, cx, cy, r);
    ctx.restore();
    return true;
}

export function drawEarthAtmosphereRim(ctx, cx, cy, r) {
    const rim = ctx.createRadialGradient(cx - r * 0.25, cy - r * 0.3, r * 0.2, cx, cy, r);
    rim.addColorStop(0.72, 'rgba(0,0,0,0)');
    rim.addColorStop(0.90, 'rgba(80, 180, 255, 0.08)');
    rim.addColorStop(0.97, 'rgba(140, 220, 255, 0.22)');
    rim.addColorStop(1, 'rgba(180, 235, 255, 0.34)');
    ctx.fillStyle = rim;
    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.fill();
}

export async function tryLoadEarthImageFromURL(url, requestRenderFn) {
    const img = new Image();
    img.decoding = 'async';
    img.crossOrigin = 'anonymous';
    await new Promise((resolve, reject) => {
        img.onload = () => resolve(true);
        img.onerror = () => reject(new Error('load failed: ' + url));
        img.src = url;
    });
    const geometry = GlobeProjection.validateEquirectangularDimensions(img.naturalWidth, img.naturalHeight);
    if (!geometry.valid) {
        throw new Error(`earth texture must be 2:1 equirectangular (${img.naturalWidth}x${img.naturalHeight}, ratio ${geometry.aspectRatio.toFixed(3)})`);
    }
    const c = document.createElement('canvas');
    const maxW = Math.min(EARTH_TEX_MAX_W, EARTH_TEX_WORK_MAX_W);
    const scale = Math.min(1, maxW / img.naturalWidth);
    c.width = Math.max(512, Math.floor(img.naturalWidth * scale));
    c.height = Math.max(256, Math.floor(img.naturalHeight * scale));
    const ictx = c.getContext('2d', { willReadFrequently: true, alpha: false });
    ictx.imageSmoothingEnabled = true;
    ictx.imageSmoothingQuality = 'high';
    ictx.drawImage(img, 0, 0, c.width, c.height);
    setEarthTexSource('jpg');
    cacheEarthImageData(c, `jpg:${c.width}x${c.height}`);
    if (requestRenderFn) requestRenderFn();
}

export async function loadEarthTexture(requestRenderFn) {
    const base = (typeof state !== 'undefined' && state.API_BASE) ? state.API_BASE : '';
    const candidates = [
        './assets/earth_day_8k.jpg',
        './assets/earth_day_4k.jpg',
        './assets/earth_day.jpg',
        base + '/v1/assets/earth-texture',
        '/v1/assets/earth-texture',
        './assets/1.jpg'
    ];
    for (const url of candidates) {
        if (!url) continue;
        try {
            await tryLoadEarthImageFromURL(url, requestRenderFn);
            return;
        } catch (_) {}
    }
    if (localAtlasBorders && localAtlasBorders.length) {
        const baked = buildEarthTextureFromAtlas(localAtlasBorders, 2048, 1024);
        setEarthTexSource('atlas');
        cacheEarthImageData(baked, 'atlas:2048x1024');
        if (requestRenderFn) requestRenderFn();
    }
}

export function decodeTopoArc(topology, arcIndex) {
    const reversed = arcIndex < 0;
    const raw = topology.arcs?.[reversed ? ~arcIndex : arcIndex];
    if (!Array.isArray(raw)) return [];
    const scale = topology.transform?.scale || [1, 1];
    const translate = topology.transform?.translate || [0, 0];
    let x = 0, y = 0;
    const points = raw.map(delta => {
        x += Number(delta[0]) || 0;
        y += Number(delta[1]) || 0;
        return [x * scale[0] + translate[0], y * scale[1] + translate[1]];
    });
    return reversed ? points.reverse() : points;
}

export function stitchTopoRing(topology, arcIndexes) {
    const points = [];
    (arcIndexes || []).forEach(index => {
        const arc = decodeTopoArc(topology, index);
        if (arc.length) points.push(...(points.length ? arc.slice(1) : arc));
    });
    return points;
}

export function decodeWorldAtlas(topology) {
    const geometries = topology?.objects?.countries?.geometries;
    if (!Array.isArray(geometries)) return [];
    const borders = [];
    geometries.forEach(geometry => {
        const polygons = geometry.type === 'Polygon' ? [geometry.arcs] : geometry.type === 'MultiPolygon' ? geometry.arcs : [];
        polygons.forEach(polygon => {
            const rings = (polygon || []).map(ring => stitchTopoRing(topology, ring)).filter(ring => ring.length > 2);
            if (rings.length) borders.push(rings);
        });
    });
    return borders;
}

export async function loadLocalWorldAtlas(requestRenderFn) {
    try {
        const response = await fetch('./assets/world-atlas-110m.topojson', {cache: 'force-cache'});
        if (!response.ok) throw new Error('atlas asset unavailable');
        const topology = await response.json();
        const borders = decodeWorldAtlas(topology);
        setLocalAtlasBorders(borders);
        incrementLocalAtlasVersion();
        setProjectedAtlasCache(null);
        if (earthTexSource !== 'jpg') {
            void loadEarthTexture(requestRenderFn);
        } else if (requestRenderFn) {
            requestRenderFn();
        }
    } catch (error) {
        console.warn('Local World Atlas unavailable; using reduced local border fallback.', error);
        void loadEarthTexture(requestRenderFn);
    }
}

export function projectAtlasPoint(lat, lon, scale, cw, ch) {
    const p = projectLatLon(lat, lon, localGlobeRotY, localGlobeRotX, scale, cw, ch);
    return {
        x: p.x,
        y: p.y,
        front: p.visible,
    };
}

export function getProjectedAtlasPaths(cw, ch) {
    const key = [
        localAtlasVersion,
        cw,
        ch,
        localGlobeScale,
        localGlobeRotY,
        localGlobeRotX,
    ].join('|');
    if (projectedAtlasCache && projectedAtlasCache.key === key) return projectedAtlasCache.paths;

    const paths = [];
    localAtlasBorders.forEach(polygon => polygon.forEach(ring => {
        if (!ring || ring.length < 2) return;
        let current = [];
        for (let index = 0; index < ring.length; index++) {
            const point = projectAtlasPoint(ring[index][1], ring[index][0], localGlobeScale, cw, ch);
            if (!point.front) {
                if (current.length > 1) paths.push(current);
                current = [];
                continue;
            }
            current.push(point);
        }
        if (current.length > 1) paths.push(current);
    }));
    setProjectedAtlasCache({ key, paths });
    return paths;
}

export function getProjectedGridPaths(cw, ch) {
    const key = [cw, ch, localGlobeScale, localGlobeRotY, localGlobeRotX].join('|');
    if (projectedGridCache && projectedGridCache.key === key) return projectedGridCache.paths;
    const paths = [];
    const appendPath = (points) => {
        let current = [];
        points.forEach(([lat, lon]) => {
            const point = projectLatLon(lat, lon, localGlobeRotY, localGlobeRotX, localGlobeScale, cw, ch);
            if (!point.visible) {
                if (current.length > 1) paths.push(current);
                current = [];
                return;
            }
            current.push(point);
        });
        if (current.length > 1) paths.push(current);
    };
    for (let lon = -180; lon < 180; lon += 30) {
        const points = [];
        for (let lat = -80; lat <= 80; lat += 4) points.push([lat, lon]);
        appendPath(points);
    }
    for (let lat = -60; lat <= 60; lat += 30) {
        const points = [];
        for (let lon = -180; lon <= 180; lon += 4) points.push([lat, lon]);
        appendPath(points);
    }
    setProjectedGridCache({ key, paths });
    return paths;
}

export function drawLocalAtlasBorders(ctx, cw, ch) {
    if (!localAtlasBorders.length) return false;
    const r = Math.min(cw, ch) * 0.42 * localGlobeScale;
    const cx = cw / 2;
    const cy = ch / 2;
    const textured = earthTexReady;
    ctx.save();
    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.clip();
    const projectedPaths = getProjectedAtlasPaths(cw, ch);
    if (textured) {
        ctx.strokeStyle = 'rgba(200, 240, 255, 0.42)';
        ctx.lineWidth = Math.max(0.5, Math.min(1.05, 0.75 * localGlobeScale));
        ctx.lineJoin = 'round';
        ctx.lineCap = 'round';
        ctx.beginPath();
        projectedPaths.forEach(path => {
            ctx.moveTo(path[0].x, path[0].y);
            for (let index = 1; index < path.length; index++) ctx.lineTo(path[index].x, path[index].y);
        });
        ctx.stroke();
        ctx.restore();
        return true;
    }
    ctx.fillStyle = 'rgba(0, 140, 190, 0.14)';
    ctx.strokeStyle = 'rgba(0, 230, 255, 0.55)';
    ctx.lineWidth = Math.max(0.65, Math.min(1.2, 0.85 * localGlobeScale));
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';

    ctx.beginPath();
    projectedPaths.forEach(path => {
        ctx.moveTo(path[0].x, path[0].y);
        for (let index = 1; index < path.length; index++) ctx.lineTo(path[index].x, path[index].y);
    });
    ctx.stroke();

    localAtlasBorders.forEach(polygon => polygon.forEach(ring => {
        let allFront = true;
        const pts = [];
        for (let i = 0; i < ring.length; i++) {
            const point = projectAtlasPoint(ring[i][1], ring[i][0], localGlobeScale, cw, ch);
            if (!point.front) {
                allFront = false;
                break;
            }
            pts.push(point);
        }
        if (!allFront || pts.length < 3) return;
        ctx.beginPath();
        ctx.moveTo(pts[0].x, pts[0].y);
        for (let i = 1; i < pts.length; i++) ctx.lineTo(pts[i].x, pts[i].y);
        ctx.closePath();
        ctx.fill();
    }));
    ctx.restore();
    return true;
}
