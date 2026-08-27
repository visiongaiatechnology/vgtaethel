// STATUS: DIAMANT VGT SUPREME
// Local-only WebGL command globe with evidence-bound directional overlays.

const DEG = Math.PI / 180;
const MAX_LINKS = 80;
const MAX_REGIONS = 160;

export class ShadowCommandGlobe {
    constructor(container, onSelect) {
        this.container = container;
        this.onSelect = typeof onSelect === 'function' ? onSelect : () => {};
        this.canvas = container.querySelector('#shadow-conflict-globe');
        this.overlay = container.querySelector('#shadow-conflict-overlay');
        this.gl = this.canvas?.getContext('webgl2', { antialias: true, alpha: true, powerPreference: 'high-performance' }) || this.canvas?.getContext('webgl', { antialias: true, alpha: true });
        this.ctx = this.overlay?.getContext('2d');
        this.yaw = -18 * DEG;
        this.pitch = -10 * DEG;
        this.zoom = 1;
        this.regions = [];
        this.links = [];
        this.frame = 0;
        this.lastFrame = 0;
        this.dragging = false;
        this.pointer = { x: -1, y: -1 };
        this.hitTargets = [];
        this.ready = this.gl ? this.initializeGL() : false;
        this.bindControls();
    }

    setData(regions, links) {
        this.regions = Array.isArray(regions) ? regions.filter(validRegion).slice(0, MAX_REGIONS) : [];
        this.links = Array.isArray(links) ? links.filter(validLink).slice(0, MAX_LINKS) : [];
    }

    start() {
        if (this.frame) return;
        const tick = now => {
            this.frame = window.requestAnimationFrame(tick);
            if (!this.container.isConnected || this.container.closest('.hidden')) return;
            if (now - this.lastFrame < 32) return;
            const delta = Math.min(50, now - this.lastFrame || 16);
            this.lastFrame = now;
            if (!this.dragging) this.yaw += delta * 0.000025;
            this.render(now);
        };
        this.frame = window.requestAnimationFrame(tick);
    }

    stop() {
        if (this.frame) window.cancelAnimationFrame(this.frame);
        this.frame = 0;
    }

    initializeGL() {
        const gl = this.gl;
        const vertexSource = `
            attribute vec3 a_position;
            attribute vec2 a_uv;
            uniform float u_yaw;
            uniform float u_pitch;
            uniform float u_aspect;
            uniform float u_zoom;
            varying vec2 v_uv;
            varying vec3 v_normal;
            void main() {
                float cy=cos(u_yaw), sy=sin(u_yaw), cp=cos(u_pitch), sp=sin(u_pitch);
                vec3 p=vec3(cy*a_position.x+sy*a_position.z,a_position.y,-sy*a_position.x+cy*a_position.z);
                p=vec3(p.x,cp*p.y-sp*p.z,sp*p.y+cp*p.z);
                float depth=3.25-p.z;
                float focal=2.05*u_zoom;
                gl_Position=vec4(p.x*focal/u_aspect,p.y*focal,depth-0.12,depth);
                v_uv=a_uv;
                v_normal=p;
            }`;
        const fragmentSource = `
            precision highp float;
            uniform sampler2D u_texture;
            uniform float u_time;
            varying vec2 v_uv;
            varying vec3 v_normal;
            void main() {
                vec3 source=texture2D(u_texture,v_uv).rgb;
                float land=max(source.g-source.b*.42,0.0);
                float luminance=dot(source,vec3(.22,.68,.10));
                float gridLon=smoothstep(.96,1.0,abs(sin(v_uv.x*3.14159265*36.0)));
                float gridLat=smoothstep(.97,1.0,abs(sin(v_uv.y*3.14159265*18.0)));
                float limb=pow(max(v_normal.z,0.0),.52);
                float light=max(dot(normalize(v_normal),normalize(vec3(-.35,.55,1.0))),0.0);
                vec3 ocean=vec3(.006,.012,.016)+vec3(.014,.024,.028)*luminance;
                vec3 terrain=vec3(.18,.135,.045)*(.35+land*1.8)+vec3(.42,.33,.11)*light*.42;
                vec3 color=mix(ocean,terrain,smoothstep(.02,.16,land));
                color+=vec3(.42,.31,.09)*(gridLon+gridLat)*.055;
                color*=.22+.78*light;
                color+=vec3(.31,.23,.07)*pow(1.0-limb,4.0)*.55;
                gl_FragColor=vec4(color,clamp(limb*1.22,0.0,1.0));
            }`;
        const program = createProgram(gl, vertexSource, fragmentSource);
        if (!program) return false;
        const geometry = sphereGeometry(96, 64);
        const buffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
        gl.bufferData(gl.ARRAY_BUFFER, geometry.vertices, gl.STATIC_DRAW);
        const indexBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, indexBuffer);
        gl.bufferData(gl.ELEMENT_ARRAY_BUFFER, geometry.indices, gl.STATIC_DRAW);
        this.program = program;
        this.indexCount = geometry.indices.length;
        this.locations = {
            position: gl.getAttribLocation(program, 'a_position'), uv: gl.getAttribLocation(program, 'a_uv'),
            yaw: gl.getUniformLocation(program, 'u_yaw'), pitch: gl.getUniformLocation(program, 'u_pitch'),
            aspect: gl.getUniformLocation(program, 'u_aspect'), zoom: gl.getUniformLocation(program, 'u_zoom'),
            time: gl.getUniformLocation(program, 'u_time'), texture: gl.getUniformLocation(program, 'u_texture')
        };
        gl.enableVertexAttribArray(this.locations.position);
        gl.vertexAttribPointer(this.locations.position, 3, gl.FLOAT, false, 20, 0);
        gl.enableVertexAttribArray(this.locations.uv);
        gl.vertexAttribPointer(this.locations.uv, 2, gl.FLOAT, false, 20, 12);
        this.texture = gl.createTexture();
        gl.bindTexture(gl.TEXTURE_2D, this.texture);
        gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, new Uint8Array([8, 10, 8, 255]));
        const image = new Image();
        image.decoding = 'async';
        image.onload = () => {
            gl.bindTexture(gl.TEXTURE_2D, this.texture);
            gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, 1);
            gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, image);
            gl.generateMipmap(gl.TEXTURE_2D);
            gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR);
            gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
        };
        image.src = 'assets/earth_day.jpg';
        gl.enable(gl.DEPTH_TEST);
        gl.enable(gl.BLEND);
        gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
        return true;
    }

    bindControls() {
        if (!this.overlay) return;
        this.overlay.addEventListener('pointerdown', event => {
            this.dragging = true;
            this.dragStart = { x: event.clientX, y: event.clientY, yaw: this.yaw, pitch: this.pitch };
            this.overlay.setPointerCapture(event.pointerId);
        });
        this.overlay.addEventListener('pointermove', event => {
            const rect = this.overlay.getBoundingClientRect();
            const scaleX = this.overlay.width / Math.max(1, rect.width);
            const scaleY = this.overlay.height / Math.max(1, rect.height);
            this.pointer = { x: (event.clientX - rect.left) * scaleX, y: (event.clientY - rect.top) * scaleY };
            if (!this.dragging) return;
            this.yaw = this.dragStart.yaw + (event.clientX - this.dragStart.x) * 0.006;
            this.pitch = Math.max(-1.15, Math.min(1.15, this.dragStart.pitch + (event.clientY - this.dragStart.y) * 0.005));
        });
        const release = () => { this.dragging = false; };
        this.overlay.addEventListener('pointerup', release);
        this.overlay.addEventListener('pointercancel', release);
        this.overlay.addEventListener('pointerleave', () => { this.pointer = { x: -1, y: -1 }; });
        this.overlay.addEventListener('wheel', event => {
            event.preventDefault();
            this.zoom = Math.max(0.72, Math.min(1.42, this.zoom - event.deltaY * 0.0008));
        }, { passive: false });
        this.overlay.addEventListener('click', () => {
            const hit = this.hitTargets.find(target => Math.hypot(target.x - this.pointer.x, target.y - this.pointer.y) < target.radius);
            if (hit) this.onSelect(hit.data);
        });
    }

    resize() {
        const rect = this.container.getBoundingClientRect();
        const dpr = Math.min(window.devicePixelRatio || 1, 2);
        const width = Math.max(420, Math.round(rect.width * dpr));
        const height = Math.max(360, Math.round(rect.height * dpr));
        for (const canvas of [this.canvas, this.overlay]) {
            if (canvas.width !== width || canvas.height !== height) {
                canvas.width = width;
                canvas.height = height;
            }
        }
        return { width, height, dpr };
    }

    render(now) {
        const size = this.resize();
        if (this.ready) this.renderSphere(size, now);
        this.renderOverlay(size, now);
    }

    renderSphere({ width, height }, now) {
        const gl = this.gl;
        gl.viewport(0, 0, width, height);
        gl.clearColor(0, 0, 0, 0);
        gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
        gl.useProgram(this.program);
        gl.uniform1f(this.locations.yaw, this.yaw);
        gl.uniform1f(this.locations.pitch, this.pitch);
        gl.uniform1f(this.locations.aspect, width / height);
        gl.uniform1f(this.locations.zoom, this.zoom);
        gl.uniform1f(this.locations.time, now * 0.001);
        gl.activeTexture(gl.TEXTURE0);
        gl.bindTexture(gl.TEXTURE_2D, this.texture);
        gl.uniform1i(this.locations.texture, 0);
        gl.drawElements(gl.TRIANGLES, this.indexCount, gl.UNSIGNED_SHORT, 0);
    }

    renderOverlay({ width, height, dpr }, now) {
        const ctx = this.ctx;
        ctx.clearRect(0, 0, width, height);
        this.hitTargets = [];
        const view = { width, height, aspect: width / height };
        this.drawOrbitalRings(ctx, view, now);
        for (const link of this.links) this.drawConflictLink(ctx, view, link, now);
        for (const region of this.regions) this.drawRegion(ctx, view, region, now, dpr);
        if (!this.ready) {
            ctx.fillStyle = '#e7c866'; ctx.font = `${12 * dpr}px ui-monospace, monospace`; ctx.textAlign = 'center';
            ctx.fillText('WEBGL COMMAND GLOBE UNAVAILABLE', width / 2, height / 2);
        }
    }

    project(latitude, longitude, altitude, view) {
        const lat = Number(latitude) * DEG;
        const lon = Number(longitude) * DEG;
        const radius = 1 + altitude;
        let x = radius * Math.cos(lat) * Math.sin(lon);
        let y = radius * Math.sin(lat);
        let z = radius * Math.cos(lat) * Math.cos(lon);
        [x, z] = [Math.cos(this.yaw) * x + Math.sin(this.yaw) * z, -Math.sin(this.yaw) * x + Math.cos(this.yaw) * z];
        [y, z] = [Math.cos(this.pitch) * y - Math.sin(this.pitch) * z, Math.sin(this.pitch) * y + Math.cos(this.pitch) * z];
        const depth = 3.25 - z;
        const focal = 2.05 * this.zoom;
        return { x: (x * focal / view.aspect / depth * 0.5 + 0.5) * view.width, y: (-y * focal / depth * 0.5 + 0.5) * view.height, visible: z > -0.04, z };
    }

    drawOrbitalRings(ctx, view, now) {
        const center = this.project(0, 0, 0, view);
        const radius = Math.min(view.width, view.height) * 0.31 * this.zoom;
        ctx.save();
        ctx.translate(center.x, center.y);
        ctx.rotate(-0.18);
        ctx.strokeStyle = 'rgba(214,178,72,.12)';
        ctx.lineWidth = 1;
        ctx.setLineDash([3, 9]);
        ctx.beginPath(); ctx.ellipse(0, 0, radius * 1.24, radius * .42, 0, 0, Math.PI * 2); ctx.stroke();
        const pulse = (now * .018) % (Math.PI * 2);
        ctx.fillStyle = 'rgba(247,211,104,.8)';
        ctx.beginPath(); ctx.arc(Math.cos(pulse) * radius * 1.24, Math.sin(pulse) * radius * .42, 2.2, 0, Math.PI * 2); ctx.fill();
        ctx.restore();
    }

    drawConflictLink(ctx, view, link, now) {
        const start = vectorFromGeo(link.attacker_latitude, link.attacker_longitude);
        const end = vectorFromGeo(link.target_latitude, link.target_longitude);
        const angle = Math.acos(Math.max(-1, Math.min(1, dot(start, end))));
        if (!Number.isFinite(angle) || angle < 0.005) return;
        const sinAngle = Math.sin(angle);
        const points = [];
        for (let index = 0; index <= 48; index++) {
            const t = index / 48;
            const a = Math.sin((1 - t) * angle) / sinAngle;
            const b = Math.sin(t * angle) / sinAngle;
            const vector = normalizeVector({ x: a * start.x + b * end.x, y: a * start.y + b * end.y, z: a * start.z + b * end.z });
            const geo = geoFromVector(vector);
            points.push(this.project(geo.lat, geo.lon, Math.sin(Math.PI * t) * .28, view));
        }
        const hostile = link.action !== 'MILITARY_SUPPORT';
        const color = hostile ? '255,69,87' : '85,190,255';
        ctx.save();
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
        ctx.shadowColor = `rgba(${color},.55)`;
        ctx.shadowBlur = 10;
        ctx.strokeStyle = `rgba(${color},.2)`;
        ctx.lineWidth = 5;
        strokeVisiblePath(ctx, points);
        ctx.shadowBlur = 0;
        ctx.strokeStyle = `rgba(${color},.9)`;
        ctx.lineWidth = 1.4;
        strokeVisiblePath(ctx, points);
        const progress = (now * .00012 + stablePhase(link.attacker_name + link.target_name)) % 1;
        const particle = points[Math.min(points.length - 1, Math.floor(progress * points.length))];
        if (particle?.visible) {
            ctx.fillStyle = `rgba(${color},1)`;
            ctx.beginPath(); ctx.arc(particle.x, particle.y, 2.8, 0, Math.PI * 2); ctx.fill();
        }
        const visible = points.filter(point => point.visible);
        if (visible.length > 1) {
            const tip = visible[visible.length - 1];
            const before = visible[visible.length - 3] || visible[0];
            drawArrowhead(ctx, before, tip, `rgba(${color},1)`);
            this.hitTargets.push({ x: (before.x + tip.x) / 2, y: (before.y + tip.y) / 2, radius: 20, data: { type: 'link', value: link } });
        }
        ctx.restore();
    }

    drawRegion(ctx, view, region, now, dpr) {
        const point = this.project(region.latitude, region.longitude, .018, view);
        if (!point.visible) return;
        const color = conflictColor(region.conflict_level);
        const risk = Math.max(0, Math.min(100, 100 - Number(region.security_score || 0)));
        const radius = (4 + risk * .045) * dpr;
        const pulse = radius + (4 + Math.sin(now * .004 + stablePhase(region.region_id)) * 2) * dpr;
        ctx.save();
        ctx.strokeStyle = color;
        ctx.fillStyle = color;
        ctx.shadowColor = color;
        ctx.shadowBlur = 14;
        ctx.beginPath(); ctx.arc(point.x, point.y, radius, 0, Math.PI * 2); ctx.fill();
        ctx.shadowBlur = 0;
        ctx.globalAlpha = .55;
        ctx.beginPath(); ctx.arc(point.x, point.y, pulse, 0, Math.PI * 2); ctx.stroke();
        ctx.globalAlpha = 1;
        ctx.font = `${9 * dpr}px ui-monospace, monospace`;
        ctx.fillStyle = '#f4e9c0';
        ctx.fillText(`${String(region.region_name).slice(0, 28).toUpperCase()}  ${region.security_score}`, point.x + radius + 6, point.y - 5);
        ctx.restore();
        this.hitTargets.push({ x: point.x, y: point.y, radius: Math.max(14, pulse), data: { type: 'region', value: region } });
    }
}

function createProgram(gl, vertexSource, fragmentSource) {
    const compile = (type, source) => {
        const shader = gl.createShader(type); gl.shaderSource(shader, source); gl.compileShader(shader);
        if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) { gl.deleteShader(shader); return null; }
        return shader;
    };
    const vertex = compile(gl.VERTEX_SHADER, vertexSource);
    const fragment = compile(gl.FRAGMENT_SHADER, fragmentSource);
    if (!vertex || !fragment) return null;
    const program = gl.createProgram(); gl.attachShader(program, vertex); gl.attachShader(program, fragment); gl.linkProgram(program);
    gl.deleteShader(vertex); gl.deleteShader(fragment);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) { gl.deleteProgram(program); return null; }
    return program;
}

function sphereGeometry(longitudes, latitudes) {
    const vertices = [];
    const indices = [];
    for (let latIndex = 0; latIndex <= latitudes; latIndex++) {
        const v = latIndex / latitudes;
        const lat = (v - .5) * Math.PI;
        for (let lonIndex = 0; lonIndex <= longitudes; lonIndex++) {
            const u = lonIndex / longitudes;
            const lon = (u * 2 - 1) * Math.PI;
            vertices.push(Math.cos(lat) * Math.sin(lon), Math.sin(lat), Math.cos(lat) * Math.cos(lon), u, v);
        }
    }
    for (let latIndex = 0; latIndex < latitudes; latIndex++) {
        for (let lonIndex = 0; lonIndex < longitudes; lonIndex++) {
            const first = latIndex * (longitudes + 1) + lonIndex;
            const second = first + longitudes + 1;
            indices.push(first, second, first + 1, second, second + 1, first + 1);
        }
    }
    return { vertices: new Float32Array(vertices), indices: new Uint16Array(indices) };
}

function vectorFromGeo(latitude, longitude) { const lat = Number(latitude) * DEG, lon = Number(longitude) * DEG; return { x: Math.cos(lat) * Math.sin(lon), y: Math.sin(lat), z: Math.cos(lat) * Math.cos(lon) }; }
function geoFromVector(vector) { return { lat: Math.asin(vector.y) / DEG, lon: Math.atan2(vector.x, vector.z) / DEG }; }
function normalizeVector(vector) { const length = Math.hypot(vector.x, vector.y, vector.z) || 1; return { x: vector.x / length, y: vector.y / length, z: vector.z / length }; }
function dot(a, b) { return a.x * b.x + a.y * b.y + a.z * b.z; }
function stablePhase(value) { let hash = 0; for (const char of String(value)) hash = (hash * 31 + char.charCodeAt(0)) >>> 0; return (hash % 1000) / 1000; }
function conflictColor(level) { if (level === 'WAR') return '#ff4557'; if (level === 'ESCALATION') return '#ff963f'; if (level === 'TENSION') return '#f0cc62'; return '#54d7a0'; }
function validCoordinate(value, min, max) { const number = Number(value); return Number.isFinite(number) && number >= min && number <= max; }
function validRegion(region) { return region && validCoordinate(region.latitude, -90, 90) && validCoordinate(region.longitude, -180, 180) && validCoordinate(region.security_score, 0, 100); }
function validLink(link) { return link && validCoordinate(link.attacker_latitude, -90, 90) && validCoordinate(link.attacker_longitude, -180, 180) && validCoordinate(link.target_latitude, -90, 90) && validCoordinate(link.target_longitude, -180, 180) && typeof link.attacker_name === 'string' && typeof link.target_name === 'string'; }
function strokeVisiblePath(ctx, points) { let open = false; ctx.beginPath(); for (const point of points) { if (!point.visible) { open = false; continue; } if (!open) { ctx.moveTo(point.x, point.y); open = true; } else ctx.lineTo(point.x, point.y); } ctx.stroke(); }
function drawArrowhead(ctx, from, to, color) { const angle = Math.atan2(to.y - from.y, to.x - from.x); ctx.fillStyle = color; ctx.beginPath(); ctx.moveTo(to.x, to.y); ctx.lineTo(to.x - 11 * Math.cos(angle - .42), to.y - 11 * Math.sin(angle - .42)); ctx.lineTo(to.x - 11 * Math.cos(angle + .42), to.y - 11 * Math.sin(angle + .42)); ctx.closePath(); ctx.fill(); }
