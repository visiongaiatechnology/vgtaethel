// STATUS: DIAMANT VGT SUPREME
// Single-source orthographic projection for every Global Watch geo layer.

const DEG_TO_RAD = Math.PI / 180;
const RAD_TO_DEG = 180 / Math.PI;
const BASE_RADIUS_RATIO = 0.42;
const MIN_PITCH = -Math.PI / 2;
const MAX_PITCH = Math.PI / 2;

export function normalizeLongitude(longitude) {
    const value = Number(longitude);
    if (!Number.isFinite(value)) return 0;
    return ((value + 540) % 360) - 180;
}

export function clampLatitude(latitude) {
    const value = Number(latitude);
    if (!Number.isFinite(value)) return 0;
    return Math.max(-90, Math.min(90, value));
}

export function globeRadius(scale, width, height) {
    const safeScale = Math.max(0.01, Number(scale) || 1);
    const safeWidth = Math.max(1, Number(width) || 1);
    const safeHeight = Math.max(1, Number(height) || 1);
    return Math.min(safeWidth, safeHeight) * BASE_RADIUS_RATIO * safeScale;
}

export function validateEquirectangularDimensions(width, height, tolerance = 0.015) {
    const imageWidth = Number(width);
    const imageHeight = Number(height);
    const allowedError = Math.max(0, Math.min(0.1, Number(tolerance) || 0));
    if (!Number.isFinite(imageWidth) || !Number.isFinite(imageHeight) || imageWidth < 512 || imageHeight < 256) {
        return Object.freeze({ valid: false, aspectRatio: 0, error: 1, reason: 'dimensions' });
    }
    const aspectRatio = imageWidth / imageHeight;
    const error = Math.abs(aspectRatio - 2) / 2;
    return Object.freeze({
        valid: error <= allowedError,
        aspectRatio,
        error,
        reason: error <= allowedError ? '' : 'projection',
    });
}

export function createProjection(rotationY, rotationX, scale, width, height) {
    const yaw = Number(rotationY) || 0;
    const pitch = Math.max(MIN_PITCH, Math.min(MAX_PITCH, Number(rotationX) || 0));
    const canvasWidth = Math.max(1, Number(width) || 1);
    const canvasHeight = Math.max(1, Number(height) || 1);
    const radius = globeRadius(scale, canvasWidth, canvasHeight);
    return Object.freeze({
        rotationY: yaw,
        rotationX: pitch,
        scale: Math.max(0.01, Number(scale) || 1),
        width: canvasWidth,
        height: canvasHeight,
        centerX: canvasWidth / 2,
        centerY: canvasHeight / 2,
        radius,
        cosPitch: Math.cos(pitch),
        sinPitch: Math.sin(pitch),
    });
}

export function project(projection, latitude, longitude) {
    const phi = clampLatitude(latitude) * DEG_TO_RAD;
    const lambda = normalizeLongitude(longitude) * DEG_TO_RAD + projection.rotationY;
    const cosPhi = Math.cos(phi);
    const sphereX = cosPhi * Math.sin(lambda);
    const sphereY = Math.sin(phi);
    const sphereZ = cosPhi * Math.cos(lambda);
    const cameraY = sphereY * projection.cosPitch - sphereZ * projection.sinPitch;
    const depth = sphereY * projection.sinPitch + sphereZ * projection.cosPitch;
    return {
        x: projection.centerX + sphereX * projection.radius,
        y: projection.centerY - cameraY * projection.radius,
        visible: depth > 0,
        depth,
    };
}

export function unproject(projection, screenX, screenY, pixelCenter = true) {
    const offset = pixelCenter ? 0.5 : 0;
    const normalizedX = (Number(screenX) + offset - projection.centerX) / projection.radius;
    const normalizedY = (projection.centerY - (Number(screenY) + offset)) / projection.radius;
    const distanceSquared = normalizedX * normalizedX + normalizedY * normalizedY;
    // Floating-point noise at the exact limb must not tear coastlines or make
    // a visible projected point impossible to unproject.
    if (distanceSquared > 1 + 1e-10) return null;
    const depth = Math.sqrt(Math.max(0, 1 - Math.min(1, distanceSquared)));
    const sphereY = normalizedY * projection.cosPitch + depth * projection.sinPitch;
    const sphereZ = -normalizedY * projection.sinPitch + depth * projection.cosPitch;
    const latitude = Math.asin(Math.max(-1, Math.min(1, sphereY))) * RAD_TO_DEG;
    const longitude = normalizeLongitude(Math.atan2(normalizedX, sphereZ) * RAD_TO_DEG - projection.rotationY * RAD_TO_DEG);
    return {
        lat: latitude,
        lon: longitude,
        depth,
        u: (longitude + 180) / 360,
        v: (90 - latitude) / 180,
    };
}

export function focusRotation(longitude, latitude) {
    return {
        rotY: -normalizeLongitude(longitude) * DEG_TO_RAD,
        rotX: clampLatitude(latitude) * DEG_TO_RAD,
    };
}

export function applyDrag(rotationY, rotationX, deltaX, deltaY) {
    return {
        rotY: (Number(rotationY) || 0) + (Number(deltaX) || 0) * 0.0055,
        rotX: Math.max(MIN_PITCH, Math.min(MAX_PITCH, (Number(rotationX) || 0) - (Number(deltaY) || 0) * 0.006)),
    };
}

export function applyZoom(scale, deltaY) {
    return Math.max(0.55, Math.min(2.6, (Number(scale) || 1) - (Number(deltaY) || 0) * 0.0014));
}

export function viewCenter(rotationY, rotationX) {
    return {
        lon: normalizeLongitude(-(Number(rotationY) || 0) * RAD_TO_DEG),
        lat: clampLatitude((Number(rotationX) || 0) * RAD_TO_DEG),
    };
}
