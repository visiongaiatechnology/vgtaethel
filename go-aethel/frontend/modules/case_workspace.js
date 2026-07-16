// STATUS: DIAMANT VGT SUPREME
// Case Workspace: evidence-first, case-scoped, pure DOM rendering.

import { state } from './state.js';

const API = '/v1/intelligence/';
const REFRESH_INTERVAL_MS = 25_000;
const SAFE_ENTITY_KINDS = Object.freeze(['person', 'organisation', 'location', 'asset', 'event', 'geo_point']);

let currentCase = null;
let casesCache = [];
let initialized = false;
let detailGeneration = 0;

function byID(id) { return document.getElementById(id); }

function element(tag, className = '', text = '') {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== '') node.textContent = String(text);
    return node;
}

function toast(message, type = 'info') {
    if (typeof window.showAethelToast === 'function') {
        window.showAethelToast(message, type);
        return;
    }
    const region = byID('case-status-region');
    if (region) {
        region.textContent = message;
        region.dataset.status = type;
    }
}

async function api(path, options = {}) {
    const response = await fetch(API + path, {
        headers: { 'Content-Type': 'application/json' },
        ...options,
    });
    if (!response.ok) {
        const detail = (await response.text().catch(() => '')).slice(0, 500);
        throw new Error(`HTTP ${response.status}${detail ? ` · ${detail}` : ''}`);
    }
    return response.json();
}

function safeHTTPSURL(value) {
    try {
        const parsed = new URL(String(value || ''));
        return parsed.protocol === 'https:' && !parsed.username && !parsed.password ? parsed : null;
    } catch (_) {
        return null;
    }
}

function clampConfidence(value, fallback = 65) {
    const parsed = Number.parseInt(String(value), 10);
    return Number.isFinite(parsed) ? Math.max(0, Math.min(100, parsed)) : fallback;
}

function downloadMarkdown(filename, content) {
    const blob = new Blob([String(content || '')], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
    setTimeout(() => URL.revokeObjectURL(url), 1500);
}

function closeCaseDialog(overlay, result, resolve) {
    overlay.remove();
    resolve(result);
}

function showCaseDialog({ title, description = '', fields = [], confirmLabel = 'BESTÄTIGEN', danger = false }) {
    return new Promise(resolve => {
        const overlay = element('div', 'case-dialog-overlay');
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        const panel = element('form', `case-dialog${danger ? ' case-dialog-danger' : ''}`);
        const heading = element('header', 'case-dialog-header');
        const identity = element('div', 'case-dialog-identity');
        identity.append(element('span', 'case-dialog-kicker', 'CASE WORKSPACE // OPERATOR INPUT'), element('h3', '', title));
        const close = element('button', 'case-dialog-close', '×');
        close.type = 'button';
        close.setAttribute('aria-label', 'Dialog schließen');
        heading.append(identity, close);
        panel.appendChild(heading);
        if (description) panel.appendChild(element('p', 'case-dialog-description', description));

        const controls = new Map();
        const fieldGrid = element('div', 'case-dialog-fields');
        fields.forEach((field, index) => {
            const wrapper = element('label', `case-dialog-field${field.wide ? ' case-dialog-field-wide' : ''}`);
            wrapper.appendChild(element('span', '', field.label));
            let control;
            if (field.type === 'textarea') {
                control = document.createElement('textarea');
                control.rows = field.rows || 5;
            } else if (field.type === 'select') {
                control = document.createElement('select');
                (field.options || []).forEach(optionValue => {
                    const option = document.createElement('option');
                    const optionObject = typeof optionValue === 'object' ? optionValue : { value: optionValue, label: optionValue };
                    option.value = String(optionObject.value);
                    option.textContent = String(optionObject.label);
                    control.appendChild(option);
                });
            } else {
                control = document.createElement('input');
                control.type = field.type || 'text';
            }
            control.name = field.name;
            control.value = String(field.value ?? '');
            control.placeholder = field.placeholder || '';
            control.required = field.required !== false;
            if (field.min != null) control.min = String(field.min);
            if (field.max != null) control.max = String(field.max);
            if (field.minLength) control.minLength = field.minLength;
            if (field.maxLength) control.maxLength = field.maxLength;
            wrapper.appendChild(control);
            fieldGrid.appendChild(wrapper);
            controls.set(field.name, control);
            if (index === 0) setTimeout(() => control.focus(), 0);
        });
        if (fields.length) panel.appendChild(fieldGrid);

        const error = element('div', 'case-dialog-error');
        error.hidden = true;
        panel.appendChild(error);
        const actions = element('footer', 'case-dialog-actions');
        const cancel = element('button', 'cyber-button case-dialog-cancel', 'ABBRECHEN');
        cancel.type = 'button';
        const submit = element('button', `cyber-button case-dialog-confirm${danger ? ' danger' : ''}`, confirmLabel);
        submit.type = 'submit';
        actions.append(cancel, submit);
        panel.appendChild(actions);
        overlay.appendChild(panel);
        document.body.appendChild(overlay);

        let escape;
        const dismiss = () => {
            if (escape) document.removeEventListener('keydown', escape);
            closeCaseDialog(overlay, null, resolve);
        };
        close.addEventListener('click', dismiss);
        cancel.addEventListener('click', dismiss);
        overlay.addEventListener('mousedown', event => { if (event.target === overlay) dismiss(); });
        escape = event => {
            if (event.key === 'Escape') {
                document.removeEventListener('keydown', escape);
                dismiss();
            }
        };
        document.addEventListener('keydown', escape);
        panel.addEventListener('submit', event => {
            event.preventDefault();
            const values = {};
            for (const [name, control] of controls) {
                const value = control.value.trim();
                if (control.required && !value) {
                    error.textContent = `${control.closest('label')?.firstElementChild?.textContent || name} ist erforderlich.`;
                    error.hidden = false;
                    control.focus();
                    return;
                }
                values[name] = value;
            }
            document.removeEventListener('keydown', escape);
            closeCaseDialog(overlay, values, resolve);
        });
    });
}

async function confirmDestructive(title, description) {
    return Boolean(await showCaseDialog({ title, description, fields: [], confirmLabel: 'DAUERHAFT LÖSCHEN', danger: true }));
}

export async function initCaseWorkspace() {
    if (initialized) return;
    initialized = true;
    byID('nav-btn-case')?.addEventListener('click', () => void refreshCaseWorkspace());
    byID('btn-create-case')?.addEventListener('click', () => void createCase());
    window.AETHEL_REFRESH_CASE_WORKSPACE = refreshCaseWorkspace;
    window.AETHEL_PROMOTE_TO_CASE = promoteToCase;
    window.AETHEL_IMPORT_FROM_WATCH = importFromGlobalWatch;
    setInterval(() => {
        const view = byID('view-case');
        if (view && !view.classList.contains('hidden')) void refreshCaseWorkspace(false);
    }, REFRESH_INTERVAL_MS);
}

async function createCase() {
    const values = await showCaseDialog({
        title: 'Neuen Intelligence Case eröffnen',
        description: 'Zweck und Umfang werden revisionssicher mit dem Fall verknüpft.',
        fields: [
            { name: 'title', label: 'Case-Titel', placeholder: 'Kurzer, präziser Untersuchungstitel', maxLength: 180 },
            { name: 'purpose', label: 'Legitimes Analyseziel', value: 'threat_intel', placeholder: 'threat_intel · supply_chain · due_diligence', maxLength: 240 },
        ],
        confirmLabel: 'CASE ERÖFFNEN',
    });
    if (!values) return;
    try {
        const created = await api('cases', { method: 'POST', body: JSON.stringify(values) });
        currentCase = created;
        await refreshCaseWorkspace();
        toast('Case wurde eröffnet.', 'success');
    } catch (error) {
        toast(`Case konnte nicht erstellt werden: ${error.message}`, 'error');
    }
}

export async function refreshCaseWorkspace(showLoading = true) {
    const list = byID('case-list');
    const details = byID('case-details-panel');
    const count = byID('case-count');
    if (!list || !details) return;
    if (showLoading) list.replaceChildren(element('div', 'case-empty', 'Lade Cases und Evidence Chain…'));
    try {
        const response = await api('cases');
        casesCache = Array.isArray(response.cases) ? response.cases : [];
        if (count) count.textContent = `${casesCache.length} CASES`;
        renderCaseList(list, casesCache);
        const selectedID = currentCase?.id;
        currentCase = casesCache.find(item => item.id === selectedID) || casesCache[0] || null;
        if (currentCase) renderCaseDetail(details, currentCase);
        else renderEmptyDetails(details);
    } catch (error) {
        list.replaceChildren(element('div', 'case-empty case-error', `Cases nicht erreichbar: ${error.message}`));
        toast('Case Workspace konnte nicht synchronisiert werden.', 'error');
    }
}

function renderEmptyDetails(container) {
    const empty = element('div', 'case-details-empty');
    empty.append(element('span', 'case-empty-symbol', '◇'), element('h3', '', 'Kein aktiver Case'), element('p', '', 'Eröffne einen Fall oder übernimm eine belegte Beobachtung aus Global Watch.'));
    const importButton = element('button', 'cyber-button case-import-button', 'AUS GLOBAL WATCH IMPORTIEREN');
    importButton.type = 'button';
    importButton.addEventListener('click', () => void importFromGlobalWatch());
    empty.appendChild(importButton);
    container.replaceChildren(empty);
}

function renderCaseList(container, cases) {
    container.replaceChildren();
    if (!cases.length) {
        container.appendChild(element('div', 'case-empty', 'Keine aktiven Ermittlungsverfahren. Global-Watch-Beobachtungen können hierher übernommen werden.'));
        return;
    }
    cases.forEach(caseItem => {
        const card = element('article', `case-list-card${currentCase?.id === caseItem.id ? ' active' : ''}`);
        card.tabIndex = 0;
        const header = element('div', 'case-list-card-head');
        header.append(element('strong', '', caseItem.title || 'Unbenannter Case'), element('span', 'case-status-badge', String(caseItem.status || 'open').toUpperCase()));
        const meta = element('div', 'case-list-card-meta');
        meta.append(element('span', '', caseItem.purpose || '—'), element('span', '', `${(caseItem.evidence || []).length} EV`), element('span', '', `${(caseItem.entities || []).length} ENT`));
        const timestamp = element('time', 'case-list-card-time', caseItem.created_at ? new Date(caseItem.created_at).toLocaleString() : '—');
        const remove = element('button', 'case-list-delete', 'LÖSCHEN');
        remove.type = 'button';
        remove.addEventListener('click', event => {
            event.stopPropagation();
            void deleteCase(caseItem);
        });
        card.append(header, meta, timestamp, remove);
        const select = () => {
            currentCase = caseItem;
            renderCaseDetail(byID('case-details-panel'), caseItem);
            container.querySelectorAll('.case-list-card').forEach(node => node.classList.toggle('active', node === card));
        };
        card.addEventListener('click', select);
        card.addEventListener('keydown', event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); select(); } });
        container.appendChild(card);
    });
}

async function deleteCase(caseItem) {
    const confirmed = await confirmDestructive('Case dauerhaft löschen', `„${caseItem.title || caseItem.id}“ einschließlich Evidence, Entitäten und Relationen entfernen. Diese Aktion ist nicht rückgängig zu machen.`);
    if (!confirmed) return;
    try {
        await api(`cases/${encodeURIComponent(caseItem.id)}`, { method: 'DELETE' });
        if (currentCase?.id === caseItem.id) currentCase = null;
        await refreshCaseWorkspace();
        toast('Case wurde dauerhaft gelöscht.', 'success');
    } catch (error) {
        toast(`Löschen fehlgeschlagen: ${error.message}`, 'error');
    }
}

function section(title, count, className = '') {
    const wrapper = element('section', `case-detail-section ${className}`.trim());
    const heading = element('div', 'case-detail-section-head');
    heading.append(element('h4', '', title), element('span', '', String(count)));
    const body = element('div', 'case-detail-section-body');
    wrapper.append(heading, body);
    return { wrapper, body };
}

function actionButton(label, className, handler) {
    const button = element('button', `cyber-button case-action ${className}`.trim(), label);
    button.type = 'button';
    button.addEventListener('click', handler);
    return button;
}

function renderCaseDetail(container, caseItem) {
    if (!container || !caseItem) return;
    const generation = ++detailGeneration;
    const evidence = Array.isArray(caseItem.evidence) ? caseItem.evidence : [];
    const entities = Array.isArray(caseItem.entities) ? caseItem.entities : [];
    const relations = Array.isArray(caseItem.relations) ? caseItem.relations : [];
    const audit = Array.isArray(caseItem.audit) ? caseItem.audit : [];

    const shell = element('div', 'case-detail-shell');
    const header = element('header', 'case-detail-header');
    const identity = element('div', 'case-detail-identity');
    identity.append(element('span', 'case-detail-id', `CASE ${caseItem.id}`), element('h2', '', caseItem.title || 'Unbenannter Case'), element('p', '', `${caseItem.purpose || '—'} · ${caseItem.classification || '—'} · ${caseItem.created_at ? new Date(caseItem.created_at).toLocaleString() : '—'}`));
    const primaryActions = element('div', 'case-detail-actions');
    primaryActions.append(
        actionButton('+ EVIDENCE', 'primary', () => void sealEvidence(caseItem.id)),
        actionButton('GLOBAL WATCH IMPORT', '', () => void importIntoCase(caseItem.id)),
        actionButton('CASE LÖSCHEN', 'danger', () => void deleteCase(caseItem)),
    );
    header.append(identity, primaryActions);
    shell.append(header, element('div', 'case-privacy-gate', 'HMAC-PSEUDONYMISIERUNG AKTIV · PERSONEN FALLBEZOGEN ALIASIERT · RE-ID NUR MIT DUAL CONTROL'));

    const evidenceSection = section('SEALED EVIDENCE', evidence.length, 'evidence');
    if (!evidence.length) evidenceSection.body.appendChild(element('div', 'case-section-empty', 'Noch keine versiegelte Evidenz.'));
    evidence.slice().reverse().forEach(item => {
        const card = element('article', 'case-evidence-card');
        const cardHead = element('div', 'case-evidence-head');
        cardHead.append(element('strong', '', item.source || 'Unbekannte Quelle'), element('span', '', String(item.validation_status || 'PENDING').toUpperCase()));
        const excerpt = element('p', '', String(item.excerpt || '').slice(0, 500));
        const metadata = element('div', 'case-evidence-meta', `SHA-256 ${String(item.sha256 || '').slice(0, 24)}… · ${item.collected_at ? new Date(item.collected_at).toLocaleString() : '—'}`);
        card.append(cardHead, excerpt, metadata);
        const url = safeHTTPSURL(item.url);
        if (url) {
            const link = element('a', 'case-evidence-link', 'Originalquelle öffnen ↗');
            link.href = url.toString();
            link.target = '_blank';
            link.rel = 'noopener noreferrer';
            card.appendChild(link);
        }
        evidenceSection.body.appendChild(card);
    });
    shell.appendChild(evidenceSection.wrapper);

    const graph = element('div', 'case-graph-grid');
    const entitySection = section('ENTITÄTEN · PSEUDONYMISIERT', entities.length, 'entities');
    if (!entities.length) entitySection.body.appendChild(element('div', 'case-section-empty', 'Keine belegten Entitäten.'));
    entities.forEach(item => {
        const row = element('div', 'case-entity-row');
        row.append(element('span', '', String(item.kind || 'entity').toUpperCase()), element('strong', '', item.label || item.id || '—'), element('span', '', `${clampConfidence(item.confidence, 0)}%`));
        entitySection.body.appendChild(row);
    });
    entitySection.wrapper.appendChild(actionButton('+ ENTITÄT', '', () => void addEntity(caseItem.id)));

    const relationSection = section('EVIDENCE-TIED RELATIONEN', relations.length, 'relations');
    if (!relations.length) relationSection.body.appendChild(element('div', 'case-section-empty', 'Keine beleggebundenen Beziehungen.'));
    relations.forEach(item => {
        const row = element('div', 'case-relation-row');
        row.append(element('span', '', item.from || item.from_entity_id || '—'), element('strong', '', item.type || item.relation || 'RELATED_TO'), element('span', '', item.to || item.to_entity_id || '—'), element('small', '', `${clampConfidence(item.confidence, 0)}% · EV ${String(item.evidenceId || item.evidence_id || '').slice(0, 10)}`));
        relationSection.body.appendChild(row);
    });
    relationSection.wrapper.appendChild(actionButton('+ RELATION', '', () => void addRelation(caseItem.id, entities, evidence)));
    graph.append(entitySection.wrapper, relationSection.wrapper);
    shell.appendChild(graph);

    const timeline = section('TIMELINE & AUDIT', evidence.length + audit.length + 1, 'timeline');
    const items = [{ at: caseItem.created_at, text: `CASE OPENED · ${caseItem.purpose || ''}` }];
    evidence.forEach(item => items.push({ at: item.collected_at, text: `EVIDENCE SEALED · ${item.source || ''}` }));
    audit.forEach(item => items.push({ at: item.at, text: `${item.action || 'AUDIT'} · ${item.detail || ''}` }));
    items.sort((left, right) => new Date(right.at) - new Date(left.at)).slice(0, 20).forEach(item => {
        const row = element('div', 'case-timeline-row');
        row.append(element('time', '', item.at ? new Date(item.at).toLocaleString() : '—'), element('span', '', item.text));
        timeline.body.appendChild(row);
    });
    shell.appendChild(timeline.wrapper);

    const footer = element('footer', 'case-detail-footer');
    footer.append(
        actionButton('RE-ID ANFORDERN', 'danger', () => void requestReID(caseItem.id)),
        actionButton('RE-ID FREIGEBEN', 'danger', () => void approveReID(caseItem.id)),
        actionButton('TIMELINE EXPORT', '', () => void exportCaseArtifact(caseItem.id, 'timeline')),
        actionButton('REPORT EXPORT', '', () => void exportCaseArtifact(caseItem.id, 'report')),
    );
    const reIDStatus = element('div', 'case-reid-status', 'RE-ID STATUS WIRD GELADEN…');
    footer.appendChild(reIDStatus);
    shell.appendChild(footer);
    container.replaceChildren(shell);
    void loadReIDStatus(caseItem.id, reIDStatus, generation);
}

async function loadReIDStatus(caseID, target, generation) {
    try {
        const status = await api(`cases/${encodeURIComponent(caseID)}/reid`);
        if (generation !== detailGeneration || !target.isConnected) return;
        const unlock = status.alias_unlock ? ` · ALIAS UNLOCK BIS ${status.unlock_expires_at || '—'}` : '';
        target.textContent = `RE-ID: ${String(status.reidentification || 'not_eligible').toUpperCase()} · REQUESTS ${status.request_count || 0}${unlock} · ${status.reason || ''}`;
    } catch (_) {
        if (generation === detailGeneration && target.isConnected) target.textContent = 'RE-ID STATUS NICHT VERFÜGBAR';
    }
}

async function sealEvidence(caseID) {
    const values = await showCaseDialog({
        title: 'Evidence versiegeln',
        description: 'Der Auszug wird gehasht und mit Provenienz in der fallbezogenen Custody Chain gespeichert.',
        fields: [
            { name: 'source', label: 'Quelle / Collector', maxLength: 160 },
            { name: 'url', label: 'HTTPS-URL oder Referenz', required: false, type: 'url', maxLength: 1200 },
            { name: 'excerpt', label: 'Beleg / Auszug', type: 'textarea', wide: true, rows: 6, maxLength: 6000 },
        ],
        confirmLabel: 'EVIDENCE SEALEN',
    });
    if (!values) return;
    try {
        await api(`cases/${encodeURIComponent(caseID)}/evidence`, { method: 'POST', body: JSON.stringify(values) });
        await refreshCaseWorkspace(false);
        toast('Evidence wurde mit SHA-256 und Custody-Provenienz versiegelt.', 'success');
    } catch (error) {
        toast(`Evidence konnte nicht versiegelt werden: ${error.message}`, 'error');
    }
}

async function addEntity(caseID) {
    const values = await showCaseDialog({
        title: 'Belegte Entität hinzufügen',
        description: 'Personenbezüge werden automatisch fallbezogen pseudonymisiert.',
        fields: [
            { name: 'label', label: 'Entitätsbezeichnung', maxLength: 240 },
            { name: 'kind', label: 'Entitätstyp', type: 'select', options: SAFE_ENTITY_KINDS },
            { name: 'confidence', label: 'Konfidenz', type: 'number', value: '65', min: 0, max: 100 },
        ],
        confirmLabel: 'ENTITÄT SPEICHERN',
    });
    if (!values) return;
    try {
        await api(`cases/${encodeURIComponent(caseID)}/entities`, { method: 'POST', body: JSON.stringify({ label: values.label, kind: values.kind, confidence: clampConfidence(values.confidence) }) });
        await refreshCaseWorkspace(false);
        toast('Entität wurde fallbezogen gespeichert.', 'success');
    } catch (error) {
        toast(`Entität konnte nicht gespeichert werden: ${error.message}`, 'error');
    }
}

async function addRelation(caseID, entities, evidence) {
    if (entities.length < 2 || !evidence.length) {
        toast('Eine Relation benötigt mindestens zwei Entitäten und eine Evidence-Referenz.', 'error');
        return;
    }
    const entityOptions = entities.map(item => ({ value: item.id || item.label, label: `${item.label || item.id} · ${item.kind || 'entity'}` }));
    const evidenceOptions = evidence.map(item => ({ value: item.id, label: `${item.source || 'Evidence'} · ${String(item.id || '').slice(0, 12)}` }));
    const values = await showCaseDialog({
        title: 'Evidence-tied Relation anlegen',
        fields: [
            { name: 'from_entity_id', label: 'Ausgangsentität', type: 'select', options: entityOptions },
            { name: 'to_entity_id', label: 'Zielentität', type: 'select', options: entityOptions },
            { name: 'relation', label: 'Relationstyp', value: 'related_to', maxLength: 100 },
            { name: 'evidence_id', label: 'Evidence-Referenz', type: 'select', options: evidenceOptions },
            { name: 'confidence', label: 'Konfidenz', type: 'number', value: '70', min: 0, max: 100 },
        ],
        confirmLabel: 'RELATION VERKNÜPFEN',
    });
    if (!values) return;
    if (values.from_entity_id === values.to_entity_id) {
        toast('Ausgangs- und Zielentität müssen verschieden sein.', 'error');
        return;
    }
    try {
        await api(`cases/${encodeURIComponent(caseID)}/relations`, { method: 'POST', body: JSON.stringify({ ...values, confidence: clampConfidence(values.confidence, 70) }) });
        await refreshCaseWorkspace(false);
        toast('Relation wurde evidence-gebunden gespeichert.', 'success');
    } catch (error) {
        toast(`Relation konnte nicht gespeichert werden: ${error.message}`, 'error');
    }
}

async function requestReID(caseID) {
    const values = await showCaseDialog({
        title: 'Re-ID-Zugriff anfordern',
        description: 'Roh-PII bleibt ausgeschlossen. Der Antrag wird auditiert und benötigt zwei unterschiedliche Freigaben.',
        fields: [{ name: 'purpose', label: 'Konkrete Begründung', type: 'textarea', rows: 4, minLength: 10, maxLength: 1200, value: 'Notwendig zur Verifizierung einer konkreten Bedrohung' }],
        confirmLabel: 'ANTRAG STELLEN',
        danger: true,
    });
    if (!values) return;
    try {
        const result = await api(`cases/${encodeURIComponent(caseID)}/reid`, { method: 'POST', body: JSON.stringify({ action: 'request', purpose: values.purpose, approver: 'operator' }) });
        await refreshCaseWorkspace(false);
        toast(`Re-ID-Antrag ${result.id || ''} wurde auditiert.`, 'success');
    } catch (error) {
        toast(`Re-ID-Antrag fehlgeschlagen: ${error.message}`, 'error');
    }
}

async function approveReID(caseID) {
    try {
        const status = await api(`cases/${encodeURIComponent(caseID)}/reid`);
        const open = (status.requests || []).filter(item => item.status === 'requested' || item.status === 'approved_once');
        if (!open.length) {
            toast('Keine offene Re-ID-Anfrage vorhanden.', 'info');
            return;
        }
        const request = open[open.length - 1];
        const values = await showCaseDialog({
            title: 'Re-ID-Antrag freigeben',
            description: `Request ${request.id}. Der zweite Approver muss sich vom ersten unterscheiden.`,
            fields: [{ name: 'approver', label: 'Approver-Identität', value: 'operator-2', maxLength: 120 }],
            confirmLabel: 'DUAL-CONTROL FREIGABE',
            danger: true,
        });
        if (!values) return;
        const result = await api(`cases/${encodeURIComponent(caseID)}/reid`, { method: 'POST', body: JSON.stringify({ action: 'approve', request_id: request.id, approver: values.approver }) });
        await refreshCaseWorkspace(false);
        toast(`Re-ID-Status: ${result.status || 'aktualisiert'} · Unlock ${Boolean(result.unlocked)}`, 'success');
    } catch (error) {
        toast(`Re-ID-Freigabe fehlgeschlagen: ${error.message}`, 'error');
    }
}

async function exportCaseArtifact(caseID, kind) {
    try {
        const result = await api(`cases/${encodeURIComponent(caseID)}/${kind}`);
        const content = result[kind] || result.report || JSON.stringify(result, null, 2);
        downloadMarkdown(`aethel-case-${caseID}-${kind}.md`, content);
        toast(`${kind === 'report' ? 'Report' : 'Timeline'} exportiert.`, 'success');
    } catch (error) {
        toast(`Export fehlgeschlagen: ${error.message}`, 'error');
    }
}

async function promoteToCase(title, summary, source, lat, lon, sourceEventID) {
    try {
        const purpose = sourceEventID ? `osint_investigation source_event_id=${String(sourceEventID).slice(0, 80)}` : 'osint_investigation';
        const created = await api('cases', { method: 'POST', body: JSON.stringify({ title: String(title || 'Promoted Observation').slice(0, 180), purpose }) });
        if (summary || title) {
            const evidence = { source: source || 'GLOBAL_WATCH', url: '', excerpt: String(summary || title).slice(0, 1600) };
            if (sourceEventID) evidence.source_event_id = String(sourceEventID).slice(0, 160);
            await api(`cases/${encodeURIComponent(created.id)}/evidence`, { method: 'POST', body: JSON.stringify(evidence) });
        }
        const latitude = Number(lat);
        const longitude = Number(lon);
        if (Number.isFinite(latitude) && Number.isFinite(longitude) && latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180 && (Math.abs(latitude) > 0.001 || Math.abs(longitude) > 0.001)) {
            await api(`cases/${encodeURIComponent(created.id)}/entities`, { method: 'POST', body: JSON.stringify({ label: `${latitude.toFixed(4)},${longitude.toFixed(4)}`, kind: 'geo_point', confidence: 70 }) }).catch(() => {});
        }
        currentCase = created;
        await refreshCaseWorkspace();
        byID('nav-btn-case')?.click();
        toast('Case eröffnet und Ausgangsevidenz versiegelt.', 'success');
        return created;
    } catch (error) {
        toast(`Promotion fehlgeschlagen: ${error.message}`, 'error');
        throw error;
    }
}

async function importIntoCase(caseID) {
    try {
        const response = await fetch('/v1/osint/feeds?domain=all');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const data = await response.json();
        const events = Array.isArray(data.events) ? data.events.slice(0, 5) : [];
        if (!events.length) {
            toast('Keine aktuellen Global-Watch-Ereignisse vorhanden.', 'info');
            return 0;
        }
        let sealed = 0;
        for (const event of events) {
            try {
                await api(`cases/${encodeURIComponent(caseID)}/evidence`, { method: 'POST', body: JSON.stringify({ source: event.source || 'GLOBAL_WATCH', url: safeHTTPSURL(event.url)?.toString() || '', excerpt: `${event.title || ''} — ${String(event.summary || '').slice(0, 800)}`, source_event_id: event.id || '' }) });
                sealed++;
            } catch (_) { /* independent evidence items remain isolated */ }
        }
        await refreshCaseWorkspace(false);
        toast(`${sealed} Global-Watch-Evidenzen versiegelt.`, sealed ? 'success' : 'error');
        return sealed;
    } catch (error) {
        toast(`Global-Watch-Import fehlgeschlagen: ${error.message}`, 'error');
        return 0;
    }
}

async function importFromGlobalWatch() {
    const values = await showCaseDialog({
        title: 'Global Watch als neuen Case importieren',
        fields: [{ name: 'title', label: 'Case-Titel', value: `Import aus Global Watch ${new Date().toLocaleDateString()}`, maxLength: 180 }],
        confirmLabel: 'CASE ERSTELLEN & IMPORTIEREN',
    });
    if (!values) return;
    try {
        const created = await api('cases', { method: 'POST', body: JSON.stringify({ title: values.title, purpose: 'osint_watch_import' }) });
        currentCase = created;
        await importIntoCase(created.id);
    } catch (error) {
        toast(`Import fehlgeschlagen: ${error.message}`, 'error');
    }
}

export const refreshCasesList = refreshCaseWorkspace;
