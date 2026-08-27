// STATUS: DIAMANT VGT SUPREME

import { byID, button, record, renderList } from './workbench_dom.js';

function dateLabel(value) {
    if (!value) return '';
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleString();
}

export function renderWorkspace(data, setFlash) {
    renderList('ow-collection-queue', data.collection_plans, item => record(
        (item.queries || []).join(' · ') || 'Collection plan',
        [item.status, item.owner_profile, `CASE ${item.case_id}`, `GAP ${item.information_gap_id}`],
        [], (item.constraints || []).join(' · '),
    ));
    renderList('ow-source-health', data.sources, item => record(
        item.name || item.id,
        [item.source_type, `TRUST T${item.trust_tier}`, item.availability_status || 'UNKNOWN', dateLabel(item.fetched_at)],
        [], item.publisher || item.final_url || item.url || '',
    ));
    renderList('ow-evidence-viewer', data.evidence, item => record(
        item.excerpt || item.id,
        [item.case_id, item.source_id, item.capture_scope, item.validation_status, item.sealed ? 'SEALED' : 'OPEN'],
        [button('WHY', () => setFlash(`SHA256 ${item.sha256} · RAW ${item.raw_sha256 || 'n/a'} · SNAPSHOT ${item.snapshot_id || 'n/a'}`))],
    ));
    renderList('ow-claim-matrix', data.claims, item => record(
        item.statement,
        [`CONF ${item.confidence}%`, item.status, item.source_nature, `SOURCES ${item.independent_source_count}`, `${(item.contradicting_evidence_ids || []).length} CONTRA`],
        [button('WHY', () => setFlash(item.calibration_basis || 'No calibration basis recorded.'))],
        `${item.subject} → ${item.predicate} → ${item.object}`,
    ));
    renderList('ow-link-graph', data.lineage, item => record(
        `${item.upstream_source_id} → ${item.downstream_source_id}`,
        [item.relationship, `CONF ${item.confidence}%`, item.reviewed ? 'REVIEWED' : 'PENDING'],
        [], item.detected_by,
    ));
    renderList('ow-timeline', [...(data.events || [])].reverse(), item => record(
        item.title,
        [item.domain, item.severity, `CONF ${item.confidence}%`, dateLabel(item.observed_at)],
        [], item.summary,
    ));
    renderList('ow-map-events', (data.events || []).filter(item => Number(item.latitude) || Number(item.longitude)), item => record(
        item.title,
        [`LAT ${item.latitude}`, `LON ${item.longitude}`, item.domain, item.severity],
    ));
    renderList('ow-alert-center', [...(data.alerts || [])].reverse(), item => record(
        item.reason,
        [item.severity, `CONF ${item.confidence}%`, item.acknowledged ? 'ACK' : 'OPEN', item.escalation_state],
        [button('WHY', () => setFlash(`Evidence: ${(item.evidence_ids || []).join(', ') || 'none recorded'}`))],
    ));
    renderList('ow-report-cases', data.cases, item => record(
        item.title,
        [item.classification, item.status, `${item.evidence_count} EVIDENCE`, `${item.entity_count} ENTITIES`],
        [button('SIGNED EXPORT', () => exportCase(item.id, item.title, setFlash))],
        item.purpose,
    ));
}

async function exportCase(caseID, title, setFlash) {
    const response = await fetch(`/v1/intelligence/analysis/case-exports/${encodeURIComponent(caseID)}`);
    if (!response.ok) {
        setFlash(`Case export failed: HTTP ${response.status}`, 'error');
        return;
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${String(title || caseID).replace(/[^a-z0-9_-]+/gi, '-')}.evidence.zip`;
    link.click();
    URL.revokeObjectURL(url);
    setFlash('Signed evidence package exported.', 'success');
}

export function renderCustody(data, recordFactory = record) {
    const events = Array.isArray(data.events) ? data.events.slice(-100).reverse() : [];
    const count = byID('ow-custody-count');
    if (count) {
        count.textContent = data.valid ? 'CHAIN VALID' : 'CHAIN INVALID';
        count.className = `ow-count ${data.valid ? 'ow-custody-valid' : 'ow-custody-invalid'}`;
    }
    const root = byID('ow-custody');
    if (!root) return;
    const list = document.createElement('div');
    list.className = 'ow-list';
    if (!events.length) list.appendChild(recordFactory('NO CUSTODY EVENTS'));
    for (const item of events) list.appendChild(recordFactory(item.action, [item.evidence_id, item.actor, dateLabel(item.at), String(item.event_hash || '').slice(0, 16)]));
    root.replaceChildren(list);
}
