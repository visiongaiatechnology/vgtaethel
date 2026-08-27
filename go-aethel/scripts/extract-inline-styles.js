// STATUS: DIAMANT VGT SUPREME

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const targets = [
    'frontend/index.html',
    'frontend/app.js',
    'frontend/modules/alarm_engine.js',
    'frontend/modules/chat.js',
    'frontend/modules/chat_addmessage.js',
    'frontend/modules/emergency_overlay.js',
    'frontend/modules/sphere.js',
    'frontend/modules/osint/selection_and_chat.js',
    'frontend/modules/osint/ui_controls.js',
];

const styles = new Map();

function hashStyle(value) {
    let hash = 0x811c9dc5;
    for (let index = 0; index < value.length; index += 1) {
        hash ^= value.charCodeAt(index);
        hash = Math.imul(hash, 0x01000193) >>> 0;
    }
    return hash.toString(16).padStart(8, '0');
}

function classFor(style) {
    const normalized = style.trim().replace(/\s+/g, ' ');
    let suffix = hashStyle(normalized);
    let className = `vgt-inline-${suffix}`;
    let collision = 0;
    while (styles.has(className) && styles.get(className) !== normalized) {
        collision += 1;
        className = `vgt-inline-${suffix}-${collision}`;
    }
    styles.set(className, normalized);
    return className;
}

function rewriteStartTag(tag) {
    const stylePattern = /\sstyle=(['"])(.*?)\1/s;
    const match = stylePattern.exec(tag);
    if (!match || match[2].includes('${')) return tag;
    const className = classFor(match[2]);
    let rewritten = tag.slice(0, match.index) + tag.slice(match.index + match[0].length);
    const classPattern = /\sclass=(['"])(.*?)\1/s;
    const classMatch = classPattern.exec(rewritten);
    if (classMatch) {
        const replacement = ` class=${classMatch[1]}${classMatch[2]} ${className}${classMatch[1]}`;
        rewritten = rewritten.slice(0, classMatch.index) + replacement + rewritten.slice(classMatch.index + classMatch[0].length);
    } else {
        rewritten = rewritten.replace(/\s*\/>$|>$/, ending => ` class="${className}"${ending}`);
    }
    return rewritten;
}

for (const relative of targets) {
    const absolute = path.join(root, relative);
    const source = fs.readFileSync(absolute, 'utf8');
    const rewritten = source.replace(/<[A-Za-z][^<>]*\sstyle=(['"])(.*?)\1[^<>]*>/gs, rewriteStartTag);
    fs.writeFileSync(absolute, rewritten, 'utf8');
}

const css = [
    '/* STATUS: DIAMANT VGT SUPREME */',
    '/* Deterministically extracted from static inline declarations. */',
    ...[...styles.entries()].sort(([left], [right]) => left.localeCompare(right)).map(([className, style]) => `.${className}{${style}}`),
    '',
].join('\n');
fs.writeFileSync(path.join(root, 'frontend/inline-extracted.css'), css, 'utf8');
