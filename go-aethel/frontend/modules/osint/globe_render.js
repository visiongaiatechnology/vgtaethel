import * as GlobeProjection from '../global_watch_projection.js';
import {
  globeContainer, setGlobeContainer,
  localGlobeCanvas, setLocalGlobeCanvas,
  localGlobeCtx, setLocalGlobeCtx,
  localGlobeRotY, setLocalGlobeRotY,
  localGlobeRotX, setLocalGlobeRotX,
  localGlobeScale, setLocalGlobeScale,
  localGlobeSelectedIndex, setLocalGlobeSelectedIndex,
  localGlobeDragging, setLocalGlobeDragging,
  localGlobeLastX, setLocalGlobeLastX,
  localGlobeLastY, setLocalGlobeLastY,
  globeRenderQueued, setGlobeRenderQueued,
  earthSphereCache, setEarthSphereCache,
  earthTexReady,
  localAtlasBorders,
  activeFeedEvents,
  cameraData
} from './state.js';
import {
  hitTestPin,
  applyWheelZoom,
  resetGlobeView,
  computeGlobeFocusRotation,
  projectLatLon,
  isGeoEvent
} from './projection.js';
import {
  drawEarthGlobeTexture,
  getProjectedGridPaths,
  loadEarthTexture,
  loadLocalWorldAtlas,
  startGlobeIdleRotation,
  markGlobeInteraction,
  earthRasterDPRCap
} from './texture_atlas.js';
import { globeLayers, visibleLayers } from './layers.js';

export function requestGlobeRender() {
    if (globeRenderQueued) return;
    setGlobeRenderQueued(true);
    requestAnimationFrame(() => {
        setGlobeRenderQueued(false);
        drawPureLocalGlobe();
    });
}

export function forceGlobeResize() {
    if (!localGlobeCanvas || !globeContainer) return;
    try {
        const rect = globeContainer.getBoundingClientRect();
        const w = Math.max(320, Math.floor(rect.width));
        const h = Math.max(300, Math.floor(rect.height));
        if (localGlobeCanvas.width !== w || localGlobeCanvas.height !== h) {
            localGlobeCanvas.width = w;
            localGlobeCanvas.height = h;
        }
        requestGlobeRender();
    } catch (_) {}
}

export function focusGlobeOnLonLat(lon, lat, opts) {
    const t = computeGlobeFocusRotation(lon, lat);
    setLocalGlobeRotY(t.rotY);
    setLocalGlobeRotX(t.rotX);
    if (opts && opts.scale != null) {
        setLocalGlobeScale(Math.max(0.55, Math.min(2.6, Number(opts.scale))));
    } else if (localGlobeScale < 1.2) {
        setLocalGlobeScale(Math.min(1.85, localGlobeScale + 0.25));
    }
    setEarthSphereCache(null);
    requestGlobeRender();
}

export function highlightEventInList(idx, showSelectionDetailsFn) {
    const feedList = document.getElementById("gw-feed-list");
    if (!feedList || !activeFeedEvents[idx]) return;

    const cards = feedList.querySelectorAll(".gw-event-card");
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
    if (ev && (ev.lat != null || ev.lon != null)) {
        focusGlobeOnLonLat(ev.lon, ev.lat);
    }
}

export function clearPins() {
    window.__globePins = [];
    setLocalGlobeSelectedIndex(-1);
}

export function initPureLocalGlobe(showSelectionDetailsFn, showCameraDetailsFn) {
    const container = document.getElementById("osint-globe");
    if (!container) return;
    setGlobeContainer(container);

    void loadEarthTexture(requestGlobeRender);
    void loadLocalWorldAtlas(requestGlobeRender);

    let stage = document.getElementById('pure-local-globe-stage');
    if (!stage) {
        stage = document.createElement('div');
        stage.id = 'pure-local-globe-stage';
        stage.style.cssText = 'position:absolute; inset:0; width:100%; height:100%; background:rgba(0,0,0,0.15); border-radius:6px; overflow:hidden; z-index:1;';
        const canvas = document.createElement('canvas');
        canvas.id = 'pure-local-globe-canvas';
        canvas.style.cssText = 'width:100%; height:100%; display:block; cursor:grab;';
        const hint = document.createElement('div');
        hint.style.cssText = 'position:absolute; bottom:6px; left:8px; font-family:var(--font-mono); font-size:8px; color:rgba(0,240,255,0.5); pointer-events:none; z-index:2;';
        hint.textContent = 'LOCAL GLOBE • DRAG ROTATE • WHEEL ZOOM • CLICK PIN';
        let resetBtn = document.getElementById('globe-reset-btn');
        if (!resetBtn) {
            resetBtn = document.createElement('button');
            resetBtn.id = 'globe-reset-btn';
            resetBtn.type = 'button';
            resetBtn.textContent = 'RESET VIEW';
            resetBtn.style.cssText = 'position:absolute; top:6px; right:6px; font-size:7px; padding:1px 5px; background:rgba(0,0,0,0.6); border:1px solid rgba(0,240,255,0.3); color:#0ff; cursor:pointer; z-index:5;';
        }
        stage.append(canvas, hint, resetBtn);
        container.style.position = 'relative';
        container.insertBefore(stage, container.firstChild);
    }

    const canvas = document.getElementById('pure-local-globe-canvas');
    if (!canvas) return;
    setLocalGlobeCanvas(canvas);

    const globeHint = [...container.querySelectorAll('div')].find(element => element.children.length === 0 && element.textContent.includes('LOCAL GLOBE'));
    if (globeHint) globeHint.textContent = 'LOCAL ATLAS · DRAG 2-AXIS ROTATE · WHEEL ZOOM · CLICK PIN';

    setLocalGlobeCtx(canvas.getContext("2d", { alpha: true }));
	if (canvas.dataset.aethelGlobeBound === 'true') {
		startGlobeIdleRotation(requestGlobeRender);
		forceGlobeResize();
		return;
	}
	canvas.dataset.aethelGlobeBound = 'true';

    const resize = () => {
        const rect = container.getBoundingClientRect();
        canvas.width = Math.max(300, Math.floor(rect.width));
        canvas.height = Math.max(280, Math.floor(rect.height));
        requestGlobeRender();
    };
    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(container);

    canvas.addEventListener("pointerdown", (e) => {
        markGlobeInteraction();
        setLocalGlobeDragging(true);
        setLocalGlobeLastX(e.clientX);
        setLocalGlobeLastY(e.clientY);
        canvas.style.cursor = "grabbing";
        canvas.setPointerCapture(e.pointerId);
    });

    canvas.addEventListener("pointermove", (e) => {
        if (!localGlobeDragging) return;
        markGlobeInteraction();
        const dx = e.clientX - localGlobeLastX;
        const dy = e.clientY - localGlobeLastY;
        const rotation = GlobeProjection.applyDrag(localGlobeRotY, localGlobeRotX, dx, dy);
        setLocalGlobeRotY(rotation.rotY);
        setLocalGlobeRotX(rotation.rotX);
        setLocalGlobeLastX(e.clientX);
        setLocalGlobeLastY(e.clientY);
        requestGlobeRender();
    });

    window.addEventListener("pointerup", () => {
        if (localGlobeDragging) {
            setLocalGlobeDragging(false);
            setEarthSphereCache(null);
            if (localGlobeCanvas) localGlobeCanvas.style.cursor = "grab";
            requestGlobeRender();
        }
    });

    canvas.addEventListener("wheel", (e) => {
        e.preventDefault();
        markGlobeInteraction();
        setEarthSphereCache(null);
        setLocalGlobeScale(applyWheelZoom(localGlobeScale, e.deltaY));
        requestGlobeRender();
    }, { passive: false });

    startGlobeIdleRotation(requestGlobeRender);

    canvas.addEventListener("click", (e) => {
        const pins = window.__globePins || [];
        const hit = hitTestPin(e.offsetX, e.offsetY, pins, localGlobeScale);
        if (hit !== -1) {
            setLocalGlobeSelectedIndex(hit);
            highlightEventInList(hit, showSelectionDetailsFn);
            const ev = activeFeedEvents[hit];
            const pin = pins.find(pp => pp.idx === hit);
            if (ev && showSelectionDetailsFn) showSelectionDetailsFn(ev, pin);
        } else if (visibleLayers.cameras) {
            for (let cam of cameraData) {
                const pp = projectLatLon(cam.lat, cam.lon, localGlobeRotY, localGlobeRotX, localGlobeScale, canvas.width, canvas.height);
                const dx = pp.x - e.offsetX; const dy = pp.y - e.offsetY;
                if (dx*dx + dy*dy < 64) {
                    if (showCameraDetailsFn) showCameraDetailsFn(cam, pp);
                    return;
                }
            }
        }
    });

    const resetBtn = document.getElementById("globe-reset-btn");
    if (resetBtn) {
        resetBtn.addEventListener("click", () => {
            const r = resetGlobeView();
            setLocalGlobeRotY(r.rotY);
            setLocalGlobeRotX(r.rotX);
            setLocalGlobeScale(r.scale);
            setLocalGlobeSelectedIndex(r.selectedIndex);
            requestGlobeRender();
        });
    }

    requestGlobeRender();
    window.setInterval(requestGlobeRender, 30000);
}

export function drawPureLocalGlobe() {
    if (!localGlobeCtx || !localGlobeCanvas) return;

    const containerEl = globeContainer || (localGlobeCanvas && localGlobeCanvas.parentElement);
    let cw = localGlobeCanvas.width;
    let ch = localGlobeCanvas.height;
    if (containerEl) {
        const dispW = Math.max(320, Math.floor(containerEl.clientWidth || 0));
        const dispH = Math.max(300, Math.floor(containerEl.clientHeight || 0));
        const dpr = Math.min(earthRasterDPRCap(), window.devicePixelRatio || 1);
        const bufW = Math.max(1, Math.floor(dispW * dpr));
        const bufH = Math.max(1, Math.floor(dispH * dpr));
        if (localGlobeCanvas.width !== bufW || localGlobeCanvas.height !== bufH) {
            localGlobeCanvas.width = bufW;
            localGlobeCanvas.height = bufH;
            localGlobeCanvas.style.width = dispW + 'px';
            localGlobeCanvas.style.height = dispH + 'px';
            setEarthSphereCache(null);
        }
        if (localGlobeCtx && localGlobeCtx.setTransform) {
            localGlobeCtx.setTransform(dpr, 0, 0, dpr, 0, 0);
        }
        cw = dispW;
        ch = dispH;
    }

    const ctx = localGlobeCtx;

    ctx.clearRect(0, 0, cw, ch);

    const cx = cw / 2;
    const cy = ch / 2;
    const r = Math.min(cw, ch) * 0.42 * localGlobeScale;

    const glow = ctx.createRadialGradient(cx - r*0.2, cy - r*0.3, r*0.1, cx, cy, r*1.15);
    glow.addColorStop(0, "rgba(0,240,255,0.06)");
    glow.addColorStop(0.6, "rgba(2,4,11,0.0)");
    ctx.fillStyle = glow;
    ctx.beginPath();
    ctx.arc(cx, cy, r * 1.18, 0, Math.PI * 2);
    ctx.fill();

    ctx.fillStyle = earthTexReady ? "#04101c" : "#02040b";
    ctx.strokeStyle = "rgba(0,240,255,0.75)";
    ctx.lineWidth = 2.2;
    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();

    const drewEarth = drawEarthGlobeTexture(ctx, cw, ch);

    ctx.strokeStyle = "rgba(0,240,255,0.95)";
    ctx.lineWidth = 0.9;
    ctx.beginPath();
    ctx.arc(cx, cy, r * 0.985, 0, Math.PI * 2);
    ctx.stroke();

    const isEmpty = (activeFeedEvents || []).length === 0;
    ctx.strokeStyle = drewEarth
        ? (isEmpty ? "rgba(0,240,255,0.10)" : "rgba(0,240,255,0.05)")
        : (isEmpty ? "rgba(0,240,255,0.22)" : "rgba(0,240,255,0.10)");
    ctx.lineWidth = 0.55;
    ctx.beginPath();
    getProjectedGridPaths(cw, ch).forEach(path => {
        ctx.moveTo(path[0].x, path[0].y);
        for (let index = 1; index < path.length; index++) ctx.lineTo(path[index].x, path[index].y);
    });
    ctx.stroke();

    if (!localAtlasBorders.length && !earthTexReady) {
        ctx.fillStyle = isEmpty ? "rgba(0,240,255,0.07)" : "rgba(0,240,255,0.035)";
        ctx.strokeStyle = isEmpty ? "rgba(0,240,255,0.35)" : "rgba(0,240,255,0.18)";
        ctx.lineWidth = 0.8;
        const conts = [
            {lat:20, lon:-80, w:38, h:52},
            {lat:35, lon:30, w:42, h:38},
            {lat:0, lon:20, w:28, h:35},
            {lat:-20, lon:130, w:22, h:25}
        ];
        conts.forEach(c => {
            const p = projectLatLon(c.lat, c.lon, localGlobeRotY, localGlobeRotX, localGlobeScale, cw, ch);
            if (!p.visible) return;
            ctx.beginPath();
            ctx.ellipse(p.x, p.y, c.w * localGlobeScale * 0.6, c.h * localGlobeScale * 0.55, 0, 0, Math.PI*2);
            ctx.fill();
            ctx.stroke();
        });
    }

    window.__globePins = [];

    for (const key in globeLayers) {
      if (globeLayers[key].visible) {
        globeLayers[key].draw(ctx, cw, ch, localGlobeRotY, localGlobeRotX, localGlobeScale);
      }
    }

    const coordsEl = document.getElementById('gw-map-coords');
    if (coordsEl) {
      const viewCenter = GlobeProjection.viewCenter(localGlobeRotY, localGlobeRotX);
      coordsEl.textContent = `LAT ${viewCenter.lat.toFixed(2)} · LON ${viewCenter.lon.toFixed(2)} · SCALE ${localGlobeScale.toFixed(2)} · GEO ${(activeFeedEvents||[]).filter(isGeoEvent).length}`;
    }
}
