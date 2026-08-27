// STATUS: DIAMANT VGT SUPREME

export function el(tag, className = '', text = '') {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== '') node.textContent = String(text);
    return node;
}

export function byID(id) {
    return document.getElementById(id);
}

export function button(text, handler, className = '') {
    const node = el('button', `ow-button${className ? ` ${className}` : ''}`, text);
    node.type = 'button';
    node.addEventListener('click', handler);
    return node;
}

export function input(id, placeholder, type = 'text') {
    const node = el('input', 'ow-input');
    node.id = id;
    node.type = type;
    node.placeholder = placeholder;
    return node;
}

export function panel(id, title, className = '') {
    const root = el('section', `ow-panel${className ? ` ${className}` : ''}`);
    const head = el('div', 'ow-panel-head');
    const count = el('span', 'ow-count', '0');
    count.id = `${id}-count`;
    head.append(el('h2', '', title), count);
    const body = el('div');
    body.id = id;
    root.append(head, body);
    return root;
}

export function record(title, metadata = [], actions = [], detail = '') {
    const root = el('article', 'ow-record');
    root.appendChild(el('div', 'ow-record-title', title || 'Untitled record'));
    const meta = el('div', 'ow-record-meta');
    for (const item of metadata.filter(Boolean)) meta.appendChild(el('span', '', item));
    root.appendChild(meta);
    if (detail) root.appendChild(el('p', 'ow-record-detail', detail));
    if (actions.length) {
        const row = el('div', 'ow-record-actions');
        for (const action of actions) row.appendChild(action);
        root.appendChild(row);
    }
    return root;
}

export function renderList(id, values, renderer) {
    const root = byID(id);
    const count = byID(`${id}-count`);
    if (!root) return;
    const items = Array.isArray(values) ? values : [];
    if (count) count.textContent = String(items.length);
    const list = el('div', 'ow-list');
    if (!items.length) list.appendChild(el('div', 'ow-empty', 'NO RECORDS'));
    else for (const item of items) list.appendChild(renderer(item));
    root.replaceChildren(list);
}

export function activateWorkspace(name) {
    for (const tab of document.querySelectorAll('.ow-tab')) {
        const active = tab.dataset.workspace === name;
        tab.classList.toggle('active', active);
        tab.setAttribute('aria-selected', String(active));
    }
    for (const workspace of document.querySelectorAll('.ow-workspace')) {
        workspace.classList.toggle('hidden', workspace.dataset.workspace !== name);
    }
}
