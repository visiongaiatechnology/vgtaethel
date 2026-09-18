/**
 * High-precision aircraft, vessel, and sensor SVG glyphs for Aethel GEOINT.
 * Based on type-aware planforms (airliner, fastjet, quadjet, turboprop, helicopter, drone, ship, satellite).
 */

const VIEW = 96;
const C = VIEW / 2; // 48
const STROKE = 'stroke="rgba(0,0,0,0.35)" stroke-width="1.6" stroke-linejoin="round"';
const STROKE_BOLD = 'stroke="rgba(0,0,0,0.45)" stroke-width="2.2" stroke-linejoin="round"';

export const GLYPH_PATHS = {
  alert: `
    <path d="M0,-38 L38,0 L0,38 L-38,0 Z" fill="currentColor" fill-opacity="0.24" ${STROKE_BOLD}/>
    <circle cx="0" cy="0" r="11" fill="currentColor"/>
    <circle cx="0" cy="0" r="4" fill="rgba(0,0,0,0.65)"/>`,
  airliner: `
    <path d="M0,-42 C 3.8,-40 4.6,-34 4.6,-26 L 4.6,-14
             L 32,4 L 34,6 L 34,10 L 31.4,9.2 L 4.6,2.4
             L 4.2,20
             L 14,28 L 14,32 L 0,28.6 L -14,32 L -14,28 L -4.2,20
             L -4.6,2.4 L -31.4,9.2 L -34,10 L -34,6 L -32,4 L -4.6,-14
             L -4.6,-26 C -4.6,-34 -3.8,-40 0,-42 Z" fill="currentColor" ${STROKE}/>
    <path d="M-15.5,-1.5 l3,7.6 4,-1.4 -1.5,-8.4 Z" fill="currentColor"/>
    <path d="M15.5,-1.5 l-3,7.6 -4,-1.4 1.5,-8.4 Z" fill="currentColor"/>
    <path d="M-1.6,33.5 L 1.6,33.5 L 1.6,40 L -1.6,40 Z" fill="currentColor"/>`,

  fastjet: `
    <path d="M0,-43
             L 3.5,-30
             C 4,-24 4.6,-16 5,-8
             L 27,20 L 27,26 L 6,16
             L 8,30 L 8,34 L 3,31
             L 3,38 L 6.5,42 L 6.5,44 L 0,41.5
             L -6.5,44 L -6.5,42 L -3,38
             L -3,31 L -8,34 L -8,30 L -6,16
             L -27,26 L -27,20 L -5,-8
             C -4.6,-16 -4,-24 -3.5,-30 Z" fill="currentColor" ${STROKE}/>`,

  quadjet: `
    <path d="M0,-45 C 7,-42 9,-34 9,-25 L 9,-11
             L 46,12 L 46,21 L 9,11.5
             L 8.4,21 L 20,31 L 20,37.5 L 0,32 L -20,37.5 L -20,31 L -8.4,21
             L -9,11.5 L -46,21 L -46,12 L -9,-11
             L -9,-25 C -9,-34 -7,-42 0,-45 Z" fill="currentColor" ${STROKE_BOLD}/>
    <rect x="-31" y="9" width="7" height="12" rx="2" fill="currentColor"/>
    <rect x="-17" y="4.5" width="7" height="12" rx="2" fill="currentColor"/>
    <rect x="10" y="4.5" width="7" height="12" rx="2" fill="currentColor"/>
    <rect x="24" y="9" width="7" height="12" rx="2" fill="currentColor"/>`,

  turboprop: `
    <path d="M0,-40 C 3.4,-38.5 4.2,-33 4.2,-26 L 4.2,-18
             L 36,-15.5 L 36,-7.5 L 4.2,-8
             L 3.8,22
             L 13,27.5 L 13,31.5 L 0,28.6 L -13,31.5 L -13,27.5 L -3.8,22
             L -4.2,-8 L -36,-7.5 L -36,-15.5 L -4.2,-18
             L -4.2,-26 C -4.2,-33 -3.4,-38.5 0,-40 Z" fill="currentColor" ${STROKE}/>
    <circle cx="-17.5" cy="-16.5" r="7.5" fill="currentColor" fill-opacity="0.5"/>
    <circle cx="17.5" cy="-16.5" r="7.5" fill="currentColor" fill-opacity="0.5"/>`,

  helicopter: `
    <circle cx="0" cy="-6" r="31" fill="currentColor" fill-opacity="0.22"/>
    <g transform="rotate(45 0 -6)">
      <rect x="-30.5" y="-8.2" width="61" height="4.4" rx="2.2" fill="currentColor" fill-opacity="0.9"/>
      <rect x="-30.5" y="-8.2" width="61" height="4.4" rx="2.2" fill="currentColor" fill-opacity="0.9" transform="rotate(90 0 -6)"/>
    </g>
    <path d="M0,-22 C 8,-20 10.5,-13 10.5,-6 C 10.5,2 7.5,7 0,8.5
             C -7.5,7 -10.5,2 -10.5,-6 C -10.5,-13 -8,-20 0,-22 Z" fill="currentColor" ${STROKE}/>
    <path d="M-2.6,8 L 2.6,8 L 1.8,32 L -1.8,32 Z" fill="currentColor" ${STROKE}/>
    <circle cx="5.6" cy="35" r="6" fill="currentColor" fill-opacity="0.6"/>`,

  drone: `
    <path d="M0,-40
             C 3.6,-40 4.6,-35 4.4,-30
             L 2.4,-12
             L 43,-7 L 43,-2.5 L 2.3,0
             L 2.1,24
             L 13,32 L 13,36 L 1.6,30
             L 0,38
             L -1.6,30 L -13,36 L -13,32 L -2.1,24
             L -2.3,0 L -43,-2.5 L -43,-7 L -2.4,-12
             L -4.4,-30 C -4.6,-35 -3.6,-40 0,-40 Z" fill="currentColor" ${STROKE}/>`,

  ship: `
    <path d="M0,-42 C 6,-38 12,-20 12,10 L 12,32 L 10,38 L -10,38 L -12,32 L -12,10 C -12,-20 -6,-38 0,-42 Z" fill="currentColor" ${STROKE}/>
    <rect x="-6" y="-8" width="12" height="24" rx="2" fill="rgba(0,0,0,0.3)"/>
    <rect x="-4" y="20" width="8" height="12" rx="1" fill="rgba(0,0,0,0.4)"/>`,

  camera: `
    <rect x="-24" y="-16" width="36" height="32" rx="4" fill="currentColor" ${STROKE}/>
    <polygon points="12,-6 28,-18 28,18 12,6" fill="currentColor" ${STROKE}/>
    <circle cx="-6" cy="0" r="8" fill="rgba(0,0,0,0.4)"/>
    <circle cx="-6" cy="0" r="4" fill="rgba(255,255,255,0.7)"/>`,

  satellite: `
    <rect x="-8" y="-12" width="16" height="24" rx="2" fill="currentColor" ${STROKE}/>
    <rect x="-40" y="-8" width="28" height="16" rx="1" fill="rgba(0,255,100,0.8)" ${STROKE}/>
    <line x1="-30" y1="-8" x2="-30" y2="8" stroke="rgba(0,0,0,0.4)" stroke-width="1.5"/>
    <line x1="-20" y1="-8" x2="-20" y2="8" stroke="rgba(0,0,0,0.4)" stroke-width="1.5"/>
    <rect x="12" y="-8" width="28" height="16" rx="1" fill="rgba(0,255,100,0.8)" ${STROKE}/>
    <line x1="22" y1="-8" x2="22" y2="8" stroke="rgba(0,0,0,0.4)" stroke-width="1.5"/>
    <line x1="32" y1="-8" x2="32" y2="8" stroke="rgba(0,0,0,0.4)" stroke-width="1.5"/>
    <circle cx="0" cy="0" r="4" fill="rgba(0,0,0,0.5)"/>`,
};

const _imageMap = new Map();

/**
 * Pre-renders an SVG glyph into an HTMLImageElement with color tint and size.
 */
export function getGlyphImage(kind, colorHex = '#00d2ff', px = 48) {
  const k = GLYPH_PATHS[kind] ? kind : 'airliner';
  const key = `${k}:${colorHex}:${px}`;
  if (_imageMap.has(key)) {
    return _imageMap.get(key);
  }

  const svgStr = `<svg xmlns="http://www.w3.org/2000/svg" width="${px}" height="${px}" viewBox="0 0 ${VIEW} ${VIEW}" style="color:${colorHex};"><g transform="translate(${C},${C})">${GLYPH_PATHS[k]}</g></svg>`;
  const img = new Image();
  img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svgStr);
  _imageMap.set(key, img);
  return img;
}

export function classifyEntityGlyph(entity) {
  if (entity.type === 'MIL_AIRCRAFT') {
    const cls = (entity.classification || '').toUpperCase();
    if (cls.includes('FIGHTER') || cls.includes('FASTJET') || cls.includes('VIPER')) return 'fastjet';
    if (cls.includes('DRONE') || cls.includes('REAPER') || cls.includes('UAV') || cls.includes('ISR')) return 'drone';
    if (cls.includes('ROTOR') || cls.includes('HELICOPTER')) return 'helicopter';
    if (cls.includes('TANKER') || cls.includes('TRANSPORT') || cls.includes('AIRLIFT') || cls.includes('BOMBER')) return 'quadjet';
    return 'fastjet';
  }
  if (entity.type === 'AIRCRAFT') {
    const cls = (entity.classification || '').toUpperCase();
    if (cls.includes('HELICOPTER')) return 'helicopter';
    if (cls.includes('PROP')) return 'turboprop';
    return 'airliner';
  }
  if (entity.type === 'VESSEL') return 'ship';
  if (entity.type === 'SATELLITE') return 'satellite';
  if (entity.type === 'CAMERA') return 'camera';
  return 'airliner';
}
