import {
  cachedRiskMarkers,
  cameraData,
  satData,
  cableData,
  citiesData,
  activeFeedEvents,
  localGlobeSelectedIndex,
  globalWatchPreferences
} from './state.js';
import {
  parseMagnitudeFromEvent,
  magnitudeBandColor,
  isEarthquakeEvent,
  isVolcanoEvent,
  isEruptingVolcano,
  volcanoMarkerColor
} from './hazards.js';
import {
  projectLatLon,
  buildGlobePins,
  safeExternalURL
} from './projection.js';
import { drawLocalAtlasBorders } from './texture_atlas.js';

export const globeLayers = {
  borders: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      drawLocalAtlasBorders(ctx, cw, ch);
    }
  },
  daynight: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      const cx = cw / 2;
      const cy = ch / 2;
      const r = Math.min(cw, ch) * 0.42 * scale;
      const now = new Date();
      const utcHour = now.getUTCHours() + now.getUTCMinutes() / 60.0;
      const yearStartUTC = Date.UTC(now.getUTCFullYear(), 0, 0);
      const dayOfYear = Math.floor((now.getTime() - yearStartUTC) / 86400000);
      const fractionalYear = (2 * Math.PI / 365) * (dayOfYear - 1 + (utcHour - 12) / 24);
      const subsolarLat = (180 / Math.PI) * (
        0.006918
        - 0.399912 * Math.cos(fractionalYear)
        + 0.070257 * Math.sin(fractionalYear)
        - 0.006758 * Math.cos(2 * fractionalYear)
        + 0.000907 * Math.sin(2 * fractionalYear)
        - 0.002697 * Math.cos(3 * fractionalYear)
        + 0.00148 * Math.sin(3 * fractionalYear)
      );
      let subsolarLon = ((utcHour - 12) * 15) % 360;
      if (subsolarLon > 180) subsolarLon -= 360;
      if (subsolarLon < -180) subsolarLon += 360;
      const sun = projectLatLon(subsolarLat, subsolarLon, rotY, rotX, scale, cw, ch);
      let dx = sun.x - cx;
      let dy = sun.y - cy;
      const length = Math.hypot(dx, dy);
      if (length < 0.0001) {
        dx = 1;
        dy = 0;
      } else {
        dx /= length;
        dy /= length;
      }
      ctx.save();
      ctx.beginPath();
      ctx.arc(cx, cy, r, 0, Math.PI * 2);
      ctx.clip();
      const terminator = ctx.createLinearGradient(
        cx + dx * r, cy + dy * r,
        cx - dx * r, cy - dy * r
      );
      const sunDepth = Math.max(-1, Math.min(1, Number(sun.z2) || 0));
      const baseline = Math.max(0, -sunDepth) * 0.18;
      terminator.addColorStop(0, `rgba(2,4,18,${baseline.toFixed(3)})`);
      terminator.addColorStop(0.42, `rgba(2,4,18,${(baseline + 0.02).toFixed(3)})`);
      terminator.addColorStop(0.60, `rgba(2,4,18,${(baseline + 0.12).toFixed(3)})`);
      terminator.addColorStop(1, `rgba(2,4,18,${Math.min(0.46, baseline + 0.34).toFixed(3)})`);
      ctx.fillStyle = terminator;
      ctx.fillRect(cx - r, cy - r, r * 2, r * 2);
      ctx.restore();
    }
  },
  risks: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      (cachedRiskMarkers || []).forEach((m) => {
        if (m.lat == null || m.lon == null) return;
        const p = projectLatLon(m.lat, m.lon, rotY, rotX, scale, cw, ch);
        if (!p.visible) return;
        const risk = Math.max(0, Math.min(100, Number(m.overall_risk) || 0));
        const rad = 3.2 + (risk / 100) * 5.5;
        let fill = 'rgba(57,255,20,0.45)';
        let stroke = 'rgba(57,255,20,0.9)';
        if (risk > 60) {
          fill = 'rgba(255,0,79,0.5)';
          stroke = 'rgba(255,0,79,0.95)';
        } else if (risk > 25) {
          fill = 'rgba(255,123,0,0.48)';
          stroke = 'rgba(255,123,0,0.95)';
        }
        ctx.save();
        ctx.beginPath();
        ctx.arc(p.x, p.y, rad, 0, Math.PI * 2);
        ctx.fillStyle = fill;
        ctx.fill();
        ctx.strokeStyle = stroke;
        ctx.lineWidth = 1;
        ctx.stroke();
        if (scale >= 1.0) {
          ctx.fillStyle = 'rgba(255,255,255,0.9)';
          ctx.font = '6px monospace';
          ctx.fillText(String(Math.round(risk)), p.x + rad + 2, p.y + 2);
        }
        ctx.restore();
      });
    }
  },
  cameras: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      cameraData.forEach((cam) => {
        const p = projectLatLon(cam.lat, cam.lon, rotY, rotX, scale, cw, ch);
        if (p.visible) {
          ctx.fillStyle = "#ff0";
          ctx.fillRect(p.x - 5, p.y - 5, 10, 10);
          ctx.strokeStyle = "#000";
          ctx.lineWidth = 0.5;
          ctx.strokeRect(p.x - 5, p.y - 5, 10, 10);
          ctx.fillStyle = "#000";
          ctx.font = "6px monospace";
          ctx.fillText("C", p.x - 2, p.y + 2);
        }
      });
    }
  },
  satellites: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      satData.forEach(s => {
        const p = projectLatLon(s.lat, s.lon, rotY, rotX, scale, cw, ch);
        if (p.visible) {
          ctx.fillStyle = "#0f0";
          ctx.beginPath(); ctx.arc(p.x, p.y, 4, 0, Math.PI*2); ctx.fill();
          ctx.fillStyle = "#0f0"; ctx.font="6px monospace"; ctx.fillText("S", p.x+5, p.y);
        }
      });
    }
  },
  cables: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      ctx.strokeStyle = "rgba(255,0,255,0.5)"; ctx.lineWidth=1;
      cableData.forEach(line => {
        ctx.beginPath();
        line.forEach((pt,i) => {
          const p = projectLatLon(pt[1], pt[0], rotY, rotX, scale, cw, ch);
          if (i==0) ctx.moveTo(p.x,p.y); else ctx.lineTo(p.x,p.y);
        });
        ctx.stroke();
      });
    }
  },
  cities: {
    label: "CITIES",
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      ctx.fillStyle = "rgba(255,220,120,0.85)";
      citiesData.forEach(city => {
        const p = projectLatLon(city.lat, city.lon, rotY, rotX, scale, cw, ch);
        if (p.visible) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, 2.2 * scale, 0, Math.PI*2);
          ctx.fill();
          if (scale > 1.1) {
            ctx.fillStyle = "rgba(255,220,120,0.7)";
            ctx.font = "6px monospace";
            ctx.fillText(city.name, p.x + 4, p.y - 3);
            ctx.fillStyle = "rgba(255,220,120,0.85)";
          }
        }
      });
    }
  },
  earthquakes: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      const t = Date.now();
      const events = window.__gwVisibleEvents || activeFeedEvents || [];
      let count = 0;
      events.forEach((ev) => {
        if (!isEarthquakeEvent(ev)) return;
        const lat = ev.lat != null ? ev.lat : ev.latitude;
        const lon = ev.lon != null ? ev.lon : ev.longitude;
        if (lat == null || lon == null) return;
        const p = projectLatLon(Number(lat), Number(lon), rotY, rotX, scale, cw, ch);
        if (!p.visible) return;
        count++;
        const mag = parseMagnitudeFromEvent(ev);
        const col = magnitudeBandColor(mag);
        const baseR = 3.2 + Math.min(8, Math.max(0, (Number(mag) || 2) - 1) * 1.4);
        for (let ring = 0; ring < 3; ring++) {
          const phase = ((t / 900) + ring * 0.33) % 1;
          const rr = baseR + phase * (14 + baseR * 2.2);
          const alpha = (1 - phase) * 0.55;
          ctx.beginPath();
          ctx.arc(p.x, p.y, rr, 0, Math.PI * 2);
          ctx.strokeStyle = col.hex;
          ctx.globalAlpha = alpha;
          ctx.lineWidth = 1.4 + (1 - phase);
          ctx.stroke();
        }
        ctx.globalAlpha = 1;
        const pulse = 1 + 0.22 * Math.sin(t / 180);
        ctx.beginPath();
        ctx.arc(p.x, p.y, baseR * pulse, 0, Math.PI * 2);
        ctx.fillStyle = col.fill;
        ctx.fill();
        ctx.strokeStyle = col.stroke;
        ctx.lineWidth = 1.5;
        ctx.stroke();
        if (mag != null && scale >= 0.95) {
          ctx.fillStyle = 'rgba(255,255,255,0.92)';
          ctx.font = 'bold 7px monospace';
          ctx.fillText('M' + Number(mag).toFixed(1), p.x + baseR + 3, p.y - 2);
        }
      });
      const el = document.getElementById('gw-cnt-quakes');
      if (el) el.textContent = String(count || '—');
    }
  },
  volcanoes: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      const t = Date.now();
      const events = window.__gwVisibleEvents || activeFeedEvents || [];
      let count = 0;
      events.forEach((ev) => {
        if (!isVolcanoEvent(ev)) return;
        const lat = ev.lat != null ? ev.lat : ev.latitude;
        const lon = ev.lon != null ? ev.lon : ev.longitude;
        if (lat == null || lon == null) return;
        const p = projectLatLon(Number(lat), Number(lon), rotY, rotX, scale, cw, ch);
        if (!p.visible) return;
        count++;
        const erupting = isEruptingVolcano(ev);
        const col = volcanoMarkerColor(erupting);
        const s = 6.5 * (1 + 0.08 * Math.sin(t / 220));
        ctx.save();
        ctx.beginPath();
        ctx.moveTo(p.x, p.y - s);
        ctx.lineTo(p.x + s * 0.85, p.y + s * 0.7);
        ctx.lineTo(p.x - s * 0.85, p.y + s * 0.7);
        ctx.closePath();
        ctx.fillStyle = col.fill;
        ctx.fill();
        ctx.strokeStyle = col.stroke;
        ctx.lineWidth = 1.4;
        ctx.stroke();
        if (erupting) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, s * 1.6 + Math.sin(t / 200) * 2, 0, Math.PI * 2);
          ctx.strokeStyle = 'rgba(244,63,94,0.45)';
          ctx.lineWidth = 1.2;
          ctx.stroke();
        }
        ctx.restore();
      });
      const el = document.getElementById('gw-cnt-volcanoes');
      if (el) el.textContent = String(count || '—');
    }
  },
  news: {
    visible: true,
    draw(ctx, cw, ch, rotY, rotX, scale) {
      const quakesOn = !!(globeLayers.earthquakes && globeLayers.earthquakes.visible);
      const volcanoesOn = !!(globeLayers.volcanoes && globeLayers.volcanoes.visible);
      const newsEvents = (activeFeedEvents || []).map((ev, idx) => {
        if (quakesOn && isEarthquakeEvent(ev)) return null;
        if (volcanoesOn && isVolcanoEvent(ev)) return null;
        return { ...ev, __idx: idx };
      }).filter(Boolean);
      let pinsToDraw = buildGlobePins(activeFeedEvents, rotY, rotX, scale, cw, ch).filter((pin) => {
        const ev = activeFeedEvents[pin.idx] || {};
        if (quakesOn && isEarthquakeEvent(ev)) return false;
        if (volcanoesOn && isVolcanoEvent(ev)) return false;
        return true;
      });
      const clusterBase = globalWatchPreferences.clusterMode === 'compact'
        ? 28
        : globalWatchPreferences.clusterMode === 'precise' ? 9 : 18;
      const clusterR = scale < 1.2 ? clusterBase * 1.25 : clusterBase;
      const used = new Array(pinsToDraw.length).fill(false);
      const clusters = [];
      for (let i = 0; i < pinsToDraw.length; i++) {
        if (used[i]) continue;
        const root = pinsToDraw[i];
        const members = [root];
        used[i] = true;
        for (let j = i + 1; j < pinsToDraw.length; j++) {
          if (used[j]) continue;
          const dx = pinsToDraw[j].x - root.x;
          const dy = pinsToDraw[j].y - root.y;
          if (dx * dx + dy * dy < clusterR * clusterR) {
            used[j] = true;
            members.push(pinsToDraw[j]);
          }
        }
        clusters.push(members);
      }
      window.__globePins = buildGlobePins(activeFeedEvents, rotY, rotX, scale, cw, ch);
      window.__globeClusters = clusters;
      void newsEvents;
      clusters.forEach((members) => {
        const p = members[0];
        const n = members.length;
        const hasSel = members.some(m => m.idx === localGlobeSelectedIndex);
        const ev = activeFeedEvents[p.idx] || {};
        let color = "#00f0ff";
        if (ev.domain === "geo") color = "#39ff14";
        else if (ev.domain === "cyber") color = "#ff004f";
        else if (ev.domain === "economic") color = "#ff7b00";
        else if (ev.domain === "humanitarian") color = "#9d4edd";

        const isSel = hasSel;
        const size = isSel ? 8.5 : (n > 1 ? 6.5 : 3.8);
        const pulse = isSel ? 1.6 + Math.sin(Date.now() / 150) * 0.8 : 1.0;

        ctx.shadowColor = color;
        ctx.shadowBlur = isSel ? 10 : 3;
        ctx.fillStyle = color;
        ctx.beginPath();
        ctx.arc(p.x, p.y, size * pulse, 0, Math.PI * 2);
        ctx.fill();

        if (n > 1) {
          ctx.fillStyle = "rgba(5,10,20,0.92)";
          ctx.beginPath();
          ctx.arc(p.x, p.y, size * pulse * 0.72, 0, Math.PI * 2);
          ctx.fill();
          ctx.fillStyle = "#fff";
          ctx.font = "bold 8px monospace";
          ctx.textAlign = "center";
          ctx.textBaseline = "middle";
          ctx.fillText(String(n > 99 ? "99+" : n), p.x, p.y + 0.5);
          ctx.textAlign = "start";
          ctx.textBaseline = "alphabetic";
        }

        if (isSel) {
          ctx.strokeStyle = "rgba(255,255,255,0.85)";
          ctx.lineWidth = 1.2;
          ctx.beginPath();
          ctx.arc(p.x, p.y, size * pulse + 3.5, 0, Math.PI * 2);
          ctx.stroke();
        }
        ctx.shadowBlur = 0;
      });
    }
  }
};

export const visibleLayers = new Proxy({}, {
  get(target, prop) {
    return globeLayers[prop] ? globeLayers[prop].visible : false;
  },
  set(target, prop, value) {
    if (globeLayers[prop]) {
      globeLayers[prop].visible = !!value;
      return true;
    }
    return false;
  },
  has(target, prop) {
    return prop in globeLayers;
  },
  ownKeys(target) {
    return Object.keys(globeLayers);
  },
  getOwnPropertyDescriptor(target, prop) {
    return { enumerable: true, configurable: true };
  }
});

export function loadUserCameras() {
  try {
    const saved = localStorage.getItem('aethel_user_cameras');
    if (saved) {
      const parsed = JSON.parse(saved);
      if (Array.isArray(parsed)) {
        parsed.forEach(cam => {
          const exists = cameraData.some(c => Math.abs(c.lat - cam.lat) < 0.01 && Math.abs(c.lon - cam.lon) < 0.01);
          if (!exists && cam.lat != null && cam.lon != null) {
            const stream = safeExternalURL(cam.stream);
            cameraData.push({
              lat: cam.lat,
              lon: cam.lon,
              name: String(cam.name || 'User Cam').slice(0, 120) + ' (user)',
              ...(stream ? { stream: stream.toString() } : {})
            });
          }
        });
      }
    }
  } catch (_) {}
}

export function saveUserCameras() {
  try {
    const toSave = cameraData.filter((c, i) => i >= 10 || (c.name || '').includes('(user)'));
    localStorage.setItem('aethel_user_cameras', JSON.stringify(toSave.map(c => ({lat:c.lat, lon:c.lon, name: c.name, stream: c.stream}))));
  } catch (_) {}
}

loadUserCameras();
