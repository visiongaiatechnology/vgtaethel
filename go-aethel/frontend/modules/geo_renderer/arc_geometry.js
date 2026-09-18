// STATUS: DIAMANT VGT SUPREME

export function generateArcPositions(Cesium, origin, target, pointCount = 30, maximumAltitudeM = 450000) {
  const safePointCount = Math.max(4, Math.min(128, Math.round(Number(pointCount) || 30)));
  const altitude = Math.max(0, Math.min(5000000, Number(maximumAltitudeM) || 0));
  const positions = [];
  for (let index = 0; index <= safePointCount; index += 1) {
    const fraction = index / safePointCount;
    const latitude = origin.lat + (target.lat - origin.lat) * fraction;
    const longitude = origin.lon + (target.lon - origin.lon) * fraction;
    const curveAltitude = Math.sin(fraction * Math.PI) * altitude;
    positions.push(Cesium.Cartesian3.fromDegrees(longitude, latitude, curveAltitude));
  }
  return positions;
}

