// STATUS: DIAMANT VGT SUPREME
// One-shot, deterministic migration of legacy dynamic HTML sinks to DOM construction.

const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..');

function rewrite(relativePath, transforms) {
    const filename = path.join(root, relativePath);
    let source = fs.readFileSync(filename, 'utf8');
    for (const [pattern, replacement] of transforms) {
        if (!pattern.test(source)) throw new Error(`Expected sink not found in ${relativePath}: ${pattern}`);
        source = source.replace(pattern, replacement);
    }
    fs.writeFileSync(filename, source, 'utf8');
}

rewrite('frontend/modules/sphere.js', [
    [/\s*empty\.innerHTML = 'Keine aktiven Automationen[^\n]+\n/, `
                const message = document.createElement('span');
                message.textContent = 'Keine aktiven Automationen vorhanden.';
                const create = document.createElement('button');
                create.type = 'button';
                create.className = 'vgt-inline-0d188e2f';
                create.textContent = '+ Ersten Task anlegen.';
                create.addEventListener('click', () => window.openTaskCreateModal?.());
                empty.append(message, document.createElement('br'), document.createElement('br'), create);
`],
    [/\s*timingRow\.innerHTML = `[^`]+`;\r?\n/, `
                const next = document.createElement('span');
                next.append('N\u00e4chster Lauf: ');
                const nextValue = document.createElement('b');
                nextValue.className = 'vgt-inline-52da9198';
                nextValue.textContent = nextStr;
                next.appendChild(nextValue);
                const last = document.createElement('span');
                last.textContent = \`Zuletzt: \${lastStr}\`;
                timingRow.append(next, last);
`],
    [/\s*taskListDiv\.innerHTML = `<div class="vgt-inline-ac122a37">[^`]+`;\r?\n/, `
            const failure = document.createElement('div');
            failure.className = 'vgt-inline-ac122a37';
            failure.textContent = \`Fehler beim Laden der Tasks: \${String(e.message || 'unknown error')}\`;
            taskListDiv.replaceChildren(failure);
`],
    [/\s*title\.innerHTML = `[^`]*\$\{escapeHtml\(task\.text \|\| task\.objective\)\}`;\r?\n/, `
    title.textContent = \`AUTOMATISCHER LAGEBERICHT // \${String(task.text || task.objective || '')}\`;
`],
    [/\s*footer\.innerHTML = `<span>[^`]+`;\r?\n/, `
    const executed = document.createElement('span');
    executed.textContent = \`Ausgef\u00fchrt: \${task.last_run_at ? new Date(task.last_run_at).toLocaleString('de-DE') : 'K\u00fcrzlich'}\`;
    footer.appendChild(executed);
`],
]);

rewrite('frontend/modules/osint/briefing_and_reader.js', [
    [/\s*firstTd\.innerHTML = `<span class="vgt-p-badge \$\{pClass\}">P\$\{num\}<\/span>`;\r?\n/, `
                        const badge = document.createElement('span');
                        badge.className = \`vgt-p-badge \${pClass}\`;
                        badge.textContent = \`P\${num}\`;
                        firstTd.replaceChildren(badge);
`],
]);

rewrite('frontend/modules/osint/selection_and_chat.js', [
    [/\s*svgPreview\.innerHTML = `<svg[^`]+`;\r?\n/, `
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('width', '100%'); svg.setAttribute('height', '48'); svg.classList.add('vgt-inline-5a5a123c');
    const rect = document.createElementNS(svg.namespaceURI, 'rect'); rect.setAttribute('width', '100%'); rect.setAttribute('height', '100%'); rect.setAttribute('fill', '#112');
    const marker = document.createElementNS(svg.namespaceURI, 'circle'); marker.setAttribute('cx', '50%'); marker.setAttribute('cy', '50%'); marker.setAttribute('r', '6'); marker.setAttribute('fill', '#334'); marker.setAttribute('stroke', '#ff0');
    const label = document.createElementNS(svg.namespaceURI, 'text'); label.setAttribute('x', '50%'); label.setAttribute('y', '38'); label.setAttribute('font-size', '8'); label.setAttribute('fill', '#ff0'); label.setAttribute('text-anchor', 'middle'); label.textContent = 'PUBLIC CAM';
    const coordinates = document.createElement('div'); coordinates.className = 'vgt-inline-d4a17506'; coordinates.textContent = \`\${Number(cam.lat).toFixed(1)}, \${Number(cam.lon).toFixed(1)}\`;
    const indicator = document.createElement('div'); indicator.className = 'vgt-inline-f4bd64bf';
    svg.append(rect, marker, label); svgPreview.append(svg, coordinates, indicator);
`],
]);
