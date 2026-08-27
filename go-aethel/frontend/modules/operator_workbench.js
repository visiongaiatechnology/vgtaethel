// STATUS: DIAMANT VGT SUPREME
// Operator orchestration only. Untrusted backend data is rendered through textContent.

import { activateWorkspace, byID, button, el, input, panel, record, renderList } from './workbench_dom.js';
import { renderCustody, renderWorkspace } from './workbench_views.js';
import { buildAnalysisCommandPanel } from './workbench_analysis_actions.js';

const API = '/v1/intelligence/analysis';
let initialized = false;
let refreshSequence = 0;

async function request(path = '', options = {}) {
    const response = await fetch(API + path, options);
    if (!response.ok) {
        let detail = '';
        try { detail = String((await response.json()).error || ''); } catch (_) { detail = ''; }
        throw new Error(detail || `HTTP ${response.status}`);
    }
    return response.json();
}

function jsonRequest(path, body) {
    return request(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
}

function setFlash(message, level = 'info') {
    const flash = byID('ow-flash');
    if (!flash) return;
    flash.textContent = message;
    flash.dataset.level = level;
}

function workspace(name, panels) {
    const root = el('div', 'ow-workspace');
    root.dataset.workspace = name;
    if (name !== 'overview') root.classList.add('hidden');
    for (const item of panels) root.appendChild(item);
    return root;
}

function buildSearchPanel() {
    const root = panel('ow-search-results', 'FEDERATED CANONICAL SEARCH', 'ow-span-2');
    const form = el('form', 'ow-form-row');
    const query = input('ow-search-query', 'Claim, entity, event, evidence…');
    const submit = el('button', 'ow-button', 'SEARCH');
    submit.type = 'submit';
    form.append(query, submit);
    form.addEventListener('submit', event => { event.preventDefault(); void runSearch(); });
    root.insertBefore(form, root.lastChild);
    return root;
}

function buildMonitorPanel() {
    const root = panel('ow-monitors', 'SAVED SEARCH MONITORS');
    const form = el('form', 'ow-form-grid');
    const name = input('ow-monitor-name', 'Monitor name');
    const query = input('ow-monitor-query', 'Search query');
    const score = input('ow-monitor-score', 'Minimum score', 'number');
    score.min = '1'; score.max = '100'; score.value = '80';
    const submit = el('button', 'ow-button', 'CREATE MONITOR'); submit.type = 'submit';
    form.append(name, query, score, submit);
    form.addEventListener('submit', event => { event.preventDefault(); void createMonitor(); });
    root.insertBefore(form, root.lastChild);
    return root;
}

function buildWorkbench() {
    const view = byID('view-workbench');
    if (!view) return;
    const shell = el('div', 'ow-shell');
    const header = el('header', 'ow-header');
    const title = el('div');
    title.append(el('div', 'ow-kicker', 'VGT AETHEL // ANALYTIC CONTROL PLANE'), el('h1', '', 'INTELLIGENCE OPERATOR WORKBENCH'), el('p', '', 'Collect · Verify · Analyze · Decide · Export'));
    const status = el('div', 'ow-status');
    const statusText = el('span', '', 'CANONICAL STORE LINKED'); statusText.id = 'ow-link-status';
    status.append(el('span', 'ow-status-dot'), statusText);
    header.append(title, status);

    const metrics = el('div', 'ow-metrics');
    for (const [id, label] of [['documents', 'Documents'], ['claims', 'Claims'], ['hypotheses', 'Hypotheses'], ['information_gaps', 'Information Gaps'], ['resolved_entities', 'Resolved Entities'], ['custody_events', 'Custody Events']]) {
        const metric = el('div', 'ow-metric');
        const value = el('strong', '', '0'); value.id = `ow-metric-${id}`;
        metric.append(value, el('span', '', label)); metrics.appendChild(metric);
    }

    const tabs = el('div', 'ow-tabs');
    tabs.setAttribute('role', 'tablist');
    for (const [name, label] of [['overview', 'OPERATIONS'], ['analysis', 'ANALYSIS'], ['evidence', 'EVIDENCE'], ['monitoring', 'MONITORING & REPORTS']]) {
        const tab = button(label, () => activateWorkspace(name), 'ow-tab');
        tab.dataset.workspace = name;
        tab.setAttribute('role', 'tab');
        tab.setAttribute('aria-selected', String(name === 'overview'));
        if (name === 'overview') tab.classList.add('active');
        tabs.appendChild(tab);
    }

    const workspaces = el('main', 'ow-workspaces');
    workspaces.append(
        workspace('overview', [buildSearchPanel(), buildDomainInvestigationPanel(), buildWebsiteMonitorPanel(), panel('ow-collection-queue', 'COLLECTION QUEUE'), panel('ow-source-health', 'SOURCE HEALTH CENTER'), panel('ow-alert-center', 'ALERT CENTER', 'ow-span-2')]),
        workspace('analysis', [buildAnalysisCommandPanel({ post: jsonRequest, get: request, refresh: refreshOperatorWorkbench, flash: setFlash }), panel('ow-ach-matrix', 'ANALYSIS OF COMPETING HYPOTHESES', 'ow-span-2'), panel('ow-claim-matrix', 'CLAIM MATRIX', 'ow-span-2'), panel('ow-hypotheses', 'COMPETING HYPOTHESES'), panel('ow-gaps', 'INFORMATION GAPS'), panel('ow-link-graph', 'SOURCE LINEAGE GRAPH'), panel('ow-entities', 'ENTITY RESOLUTION REVIEW'), panel('ow-entity-history', 'ENTITY VERSION HISTORY'), panel('ow-timeline', 'SYNCHRONIZED TIMELINE'), panel('ow-map-events', 'GEO EVENT LAYER')]),
        workspace('evidence', [panel('ow-evidence-viewer', 'EVIDENCE VIEWER', 'ow-span-2'), panel('ow-custody', 'CHAIN OF CUSTODY'), buildAcquisitionPanel()]),
        workspace('monitoring', [buildMonitorPanel(), panel('ow-report-cases', 'REPORT BUILDER & SIGNED EXPORTS'), buildAnalyticsPanel()]),
    );

    const flash = el('div', 'ow-flash');
    flash.id = 'ow-flash'; flash.setAttribute('role', 'status'); flash.setAttribute('aria-live', 'polite');
    shell.append(header, metrics, tabs, workspaces, flash);
    view.replaceChildren(shell);
}

function buildDomainInvestigationPanel() {
    const root = panel('ow-domain-results', 'DNS · RDAP · CERTIFICATE TRANSPARENCY', 'ow-span-2');
    const form = el('form', 'ow-form-row');
    const domain = input('ow-domain-query', 'example.org');
    const submit = el('button', 'ow-button', 'INVESTIGATE DOMAIN'); submit.type = 'submit';
    form.append(domain, submit);
    form.addEventListener('submit', event => { event.preventDefault(); void investigateDomain(domain, submit); });
    root.insertBefore(form, root.lastChild);
    return root;
}

function buildWebsiteMonitorPanel() {
    const root = panel('ow-website-monitors', 'WEBSITE CHANGE MONITORS', 'ow-span-2');
    const form = el('form', 'ow-form-grid ow-monitor-create');
    const name = input('ow-web-name', 'Monitor name');
    const url = input('ow-web-url', 'https://official.example/status', 'url');
    const source = input('ow-web-source', 'Source ID');
    const domain = input('ow-web-domain', 'Intelligence domain'); domain.value = 'general';
    const interval = input('ow-web-interval', 'Interval minutes', 'number'); interval.min = '5'; interval.max = '10080'; interval.value = '30';
    const license = input('ow-web-license', 'License / terms ID');
    const retention = input('ow-web-retention', 'Retention days', 'number'); retention.min = '1'; retention.max = '3650'; retention.value = '365';
    const submit = el('button', 'ow-button', 'CREATE CHANGE MONITOR'); submit.type = 'submit';
    form.append(name, url, source, domain, interval, license, retention, submit);
    form.addEventListener('submit', event => { event.preventDefault(); void createWebsiteMonitor({ name, url, source, domain, interval, license, retention }, submit); });
    root.insertBefore(form, root.lastChild);
    return root;
}

function buildAcquisitionPanel() {
    const root = panel('ow-acquisition', 'FORENSIC ACQUISITION', 'ow-span-2');
    root.lastChild.append(buildDocumentUpload(), buildImageUpload());
    return root;
}

function buildAnalyticsPanel() {
    const root = panel('ow-analytics', 'COLD ANALYTICS SNAPSHOT');
    root.lastChild.append(el('p', 'ow-panel-copy', 'Create an integrity-hashed Parquet/DuckDB snapshot without modifying the canonical operational store.'), button('EXPORT PARQUET SNAPSHOT', () => void exportAnalytics()));
    return root;
}

function buildDocumentUpload() {
    const form = el('form', 'ow-upload');
    form.append(el('label', '', 'DOCUMENT / OFFICIAL EXPORT'));
    const file = input('ow-document-file', '', 'file'); file.className = 'ow-file'; file.accept = '.json,.html,.htm,.txt,application/json,text/html,text/plain';
    const source = input('ow-document-source', 'Source ID');
    const domain = input('ow-document-domain', 'Domain'); domain.value = 'general';
    const format = el('select', 'ow-select'); format.id = 'ow-document-format';
    for (const value of ['json', 'html', 'text']) { const option = el('option', '', value.toUpperCase()); option.value = value; format.appendChild(option); }
    const submit = el('button', 'ow-button', 'IMPORT & QUARANTINE CHECK'); submit.type = 'submit';
    form.append(file, source, domain, format, submit);
    form.addEventListener('submit', event => { event.preventDefault(); void uploadDocument(file, source, domain, format, submit); });
    return form;
}

function buildImageUpload() {
    const form = el('form', 'ow-upload');
    form.append(el('label', '', 'LOCAL REVERSE IMAGE MATCH'));
    const file = input('ow-image-file', '', 'file'); file.className = 'ow-file'; file.accept = 'image/png,image/jpeg,image/gif';
    const label = input('ow-image-label', 'Evidence label');
    const caseID = input('ow-image-case', 'Case ID (optional)');
    const source = input('ow-image-source', 'Source ID (optional)');
    const submit = el('button', 'ow-button', 'MATCH & INDEX'); submit.type = 'submit';
    form.append(file, label, caseID, source, submit);
    form.addEventListener('submit', event => { event.preventDefault(); void uploadImage(file, label, caseID, source, submit); });
    return form;
}

export async function refreshOperatorWorkbench() {
    if (!byID('view-workbench')) return;
    const sequence = ++refreshSequence;
    setFlash('Synchronizing canonical intelligence state…');
    try {
        const [summary, workspaceData, hypotheses, gaps, entities, custody, monitors] = await Promise.all([
            request(), request('/workspace'), request('/hypotheses'), request('/gaps'), request('/entity-resolution'), request('/custody'), request('/search-monitors'),
        ]);
        if (sequence !== refreshSequence) return;
        for (const key of ['documents', 'claims', 'hypotheses', 'information_gaps', 'resolved_entities', 'custody_events']) {
            const node = byID(`ow-metric-${key}`); if (node) node.textContent = String(summary[key] ?? 0);
        }
        const linkStatus = byID('ow-link-status');
        if (linkStatus) linkStatus.textContent = summary.state_journal_valid ? 'STORE + JOURNAL VERIFIED' : 'JOURNAL NOT INITIALIZED';
        renderWorkspace(workspaceData, setFlash);
        renderWebsiteMonitors(workspaceData.website_monitors, workspaceData.website_changes);
        renderList('ow-hypotheses', hypotheses.hypotheses, item => record(item.statement, [`CONF ${item.confidence}%`, item.status, `CASE ${item.case_id}`, `${(item.contradicting_evidence_ids || []).length} CONTRA`], [button('WHY', () => setFlash((item.change_conditions || []).join(' · ') || 'No change conditions recorded.'))]));
        renderList('ow-gaps', gaps.information_gaps, item => record(item.question, [item.priority, item.status, `CASE ${item.case_id}`], [], item.rationale));
        renderEntities(entities.candidates, entities.versions);
        renderCustody(custody);
        renderMonitors(monitors);
        setFlash('Canonical state synchronized.', 'success');
    } catch (error) {
        setFlash(`Workbench unavailable: ${error.message}`, 'error');
    }
}

async function createWebsiteMonitor(fields, submit) {
    submit.disabled = true;
    try {
        await jsonRequest('/web-monitors', {
            name: fields.name.value.trim(), url: fields.url.value.trim(), source_id: fields.source.value.trim(), domain: fields.domain.value.trim(),
            interval_minutes: Number.parseInt(fields.interval.value, 10), license_id: fields.license.value.trim(), allowed_use: 'situational-awareness',
            retention_days: Number.parseInt(fields.retention.value, 10), classification: 'public', authentication_mode: 'none', geography: 'global', redistribution: 'metadata-only',
        });
        fields.name.value = ''; fields.url.value = ''; fields.source.value = ''; fields.license.value = '';
        await refreshOperatorWorkbench(); setFlash('Website change monitor created.', 'success');
    } catch (error) { setFlash(`Monitor creation failed: ${error.message}`, 'error'); }
    finally { submit.disabled = false; }
}

function renderWebsiteMonitors(monitors, changes) {
    const changeList = Array.isArray(changes) ? changes : [];
    renderList('ow-website-monitors', monitors, monitor => {
        const monitorChanges = changeList.filter(change => change.monitor_id === monitor.id);
        return record(monitor.name, [monitor.enabled ? 'ARMED' : 'DISABLED', `EVERY ${monitor.interval_minutes} MIN`, `${monitor.consecutive_failures} FAILURES`, `${monitorChanges.length} CHANGES`, monitor.last_http_status ? `HTTP ${monitor.last_http_status}` : 'NOT RUN'], [button('RUN NOW', async () => {
            try {
                const result = await jsonRequest(`/web-monitors/${encodeURIComponent(monitor.id)}/run`, {});
                await refreshOperatorWorkbench(); setFlash(result.change ? 'Website change captured and archived.' : 'Website baseline checked; no change.', 'success');
            } catch (error) { setFlash(`Website collection failed: ${error.message}`, 'error'); }
        })], monitor.last_error || `${monitor.url} · NEXT ${monitor.next_check_at}`);
    });
}

async function runSearch() {
    const query = byID('ow-search-query')?.value.trim() || '';
    if (query.length < 2) { setFlash('Search query requires at least two characters.', 'error'); return; }
    try {
        const data = await jsonRequest('/search', { query, limit: 100 });
        renderList('ow-search-results', data.hits, hit => record(hit.title, [hit.record_type, `SCORE ${hit.score}`, hit.source_id, hit.case_id, hit.timestamp], [button('WHY', () => setFlash(hit.snippet || hit.title))]));
        setFlash(`${data.count || 0} canonical records matched.`, 'success');
    } catch (error) { setFlash(`Search failed: ${error.message}`, 'error'); }
}

async function investigateDomain(domainInput, submit) {
    const domain = domainInput.value.trim();
    if (!domain) { setFlash('Enter a domain to investigate.', 'error'); return; }
    submit.disabled = true;
    try {
        const data = await jsonRequest('/domain-investigation', { domain });
        const records = [];
        records.push(record(`DNS ${data.domain}`, [
            `${(data.dns?.addresses || []).length} ADDRESSES`,
            `${(data.dns?.name_servers || []).length} NS`,
            `${(data.dns?.mail_servers || []).length} MX`,
        ], [], [...(data.dns?.addresses || []), ...(data.dns?.name_servers || []), ...(data.dns?.mail_servers || [])].join(' · ')));
        records.push(record(`RDAP ${data.rdap?.ldh_name || data.domain}`, data.rdap?.status || [], [], `${data.rdap?.handle || ''} · ${(data.rdap?.name_servers || []).join(' · ')}`));
        for (const certificate of data.certificate_transparency || []) {
            records.push(record((certificate.dns_names || []).join(' · '), [certificate.issuer_name, certificate.not_before, certificate.not_after, certificate.revoked ? 'REVOKED' : 'OBSERVED'], [], `CERT ${certificate.cert_sha256 || certificate.id}`));
        }
        renderList('ow-domain-results', records, item => item);
        const warning = (data.collection_warnings || []).join(' · ');
        setFlash(warning || `${data.certificate_transparency?.length || 0} CT certificates collected.`, warning ? 'error' : 'success');
    } catch (error) { setFlash(`Domain investigation failed: ${error.message}`, 'error'); }
    finally { submit.disabled = false; }
}

async function createMonitor() {
    const name = byID('ow-monitor-name')?.value.trim() || '';
    const query = byID('ow-monitor-query')?.value.trim() || '';
    const minimumScore = Number.parseInt(byID('ow-monitor-score')?.value || '80', 10);
    try {
        await jsonRequest('/search-monitors', { name, search: { query, limit: 100 }, minimum_score: minimumScore });
        byID('ow-monitor-name').value = ''; byID('ow-monitor-query').value = '';
        await refreshOperatorWorkbench(); setFlash('Saved-search monitor armed.', 'success');
    } catch (error) { setFlash(`Monitor creation failed: ${error.message}`, 'error'); }
}

function renderMonitors(data) {
    const alerts = Array.isArray(data.alerts) ? data.alerts : [];
    renderList('ow-monitors', data.monitors, monitor => record(monitor.name, [`MIN ${monitor.minimum_score}`, monitor.enabled ? 'ARMED' : 'DISABLED', `${(monitor.seen_hit_ids || []).length} SEEN`, `${alerts.filter(item => item.monitor_id === monitor.id).length} ALERTS`], [button('RUN NOW', async () => { try { const result = await jsonRequest(`/search-monitors/${encodeURIComponent(monitor.id)}/run`, {}); setFlash(`${result.count || 0} new alerts.`, 'success'); await refreshOperatorWorkbench(); } catch (error) { setFlash(error.message, 'error'); } })]));
}

function renderEntities(candidates, versions) {
    renderList('ow-entities', candidates, candidate => {
        const actions = candidate.status === 'pending' ? [button('MERGE', () => reviewEntity(candidate.id, 'merge')), button('SPLIT', () => reviewEntity(candidate.id, 'split')), button('REJECT', () => reviewEntity(candidate.id, 'reject'), 'negative')] : [];
        return record(`${candidate.left_entity_id} ↔ ${candidate.right_entity_id}`, [`SCORE ${candidate.score}`, candidate.status, `CASE ${candidate.case_id}`], actions, (candidate.reasons || []).join(' · '));
    });
    renderList('ow-entity-history', [...(versions || [])].reverse(), version => record(`${version.snapshot?.canonical_label || version.resolved_entity_id} · v${version.version}`, [version.action, version.actor, version.at], [], `${version.reason} · ${(version.snapshot?.source_entity_ids || []).join(' · ')}`));
}

async function reviewEntity(id, action) {
    try {
        await jsonRequest(`/entity-resolution/candidates/${encodeURIComponent(id)}/review`, { action, actor: 'operator-workbench', reason: `Operator ${action} decision` });
        await refreshOperatorWorkbench(); setFlash(`Entity candidate ${action} decision recorded.`, 'success');
    } catch (error) { setFlash(`Entity review failed: ${error.message}`, 'error'); }
}

async function uploadDocument(fileInput, source, domain, format, submit) {
    const file = fileInput.files?.[0];
    if (!file) { setFlash('Select a document first.', 'error'); return; }
    const form = new FormData(); form.append('document', file); form.append('format', format.value); form.append('source_id', source.value.trim()); form.append('domain', domain.value.trim());
    submit.disabled = true;
    try { const result = await request('/import', { method: 'POST', body: form }); setFlash(`Imported ${result.observation_ids?.length || 0} records; quarantined=${Boolean(result.quarantined)}.`, result.quarantined ? 'error' : 'success'); fileInput.value = ''; await refreshOperatorWorkbench(); }
    catch (error) { setFlash(`Import failed: ${error.message}`, 'error'); } finally { submit.disabled = false; }
}

async function uploadImage(fileInput, label, caseID, source, submit) {
    const file = fileInput.files?.[0];
    if (!file) { setFlash('Select an image first.', 'error'); return; }
    const form = new FormData(); form.append('image', file); form.append('label', label.value.trim()); form.append('case_id', caseID.value.trim()); form.append('source_id', source.value.trim()); form.append('index', 'true');
    submit.disabled = true;
    try { const result = await request('/images/match', { method: 'POST', body: form }); const best = result.matches?.[0]; setFlash(best ? `Best visual match: ${best.similarity_percent}% · ${best.fingerprint.label}` : 'Image fingerprint indexed; no prior matches.', 'success'); fileInput.value = ''; }
    catch (error) { setFlash(`Image match failed: ${error.message}`, 'error'); } finally { submit.disabled = false; }
}

async function exportAnalytics() {
    try {
        const result = await request('/analytics/export', { method: 'POST' });
        setFlash(`Cold snapshot ${result.artifact} · ${result.rows} rows · ${result.engine} · SHA256 ${String(result.sha256 || '').slice(0, 16)}…`, 'success');
    } catch (error) { setFlash(`Analytics export failed: ${error.message}`, 'error'); }
}

export function initOperatorWorkbench() {
    if (initialized) return;
    initialized = true;
    buildWorkbench();
    byID('nav-btn-workbench')?.addEventListener('click', () => void refreshOperatorWorkbench());
    window.setInterval(() => { const view = byID('view-workbench'); if (view && !view.classList.contains('hidden')) void refreshOperatorWorkbench(); }, 30_000);
}
