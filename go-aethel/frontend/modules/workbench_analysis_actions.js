// STATUS: DIAMANT VGT SUPREME

import { el, input, panel, record, renderList } from './workbench_dom.js';

function textarea(id, placeholder) {
    const node = el('textarea', 'ow-input ow-textarea');
    node.id = id;
    node.placeholder = placeholder;
    node.rows = 2;
    return node;
}

function select(id, values) {
    const node = el('select', 'ow-select');
    node.id = id;
    for (const value of values) {
        const option = el('option', '', value.toUpperCase());
        option.value = value;
        node.appendChild(option);
    }
    return node;
}

function ids(value) {
    return [...new Set(String(value || '').split(',').map(item => item.trim()).filter(Boolean))];
}

function formSection(title, fields, submitLabel, handler) {
    const details = el('details', 'ow-action-section');
    details.appendChild(el('summary', '', title));
    const form = el('form', 'ow-form-grid ow-action-form');
    for (const field of fields) form.appendChild(field);
    const submit = el('button', 'ow-button ow-span-2', submitLabel);
    submit.type = 'submit';
    form.appendChild(submit);
    form.addEventListener('submit', event => { event.preventDefault(); void handler(submit); });
    details.appendChild(form);
    return details;
}

export function buildAnalysisCommandPanel({ post, get, refresh, flash }) {
    const root = panel('ow-analysis-actions', 'ANALYTIC AUTHORING', 'ow-span-2');
    const body = root.lastChild;

    const claim = {
        caseID: input('ow-claim-case', 'Case ID'), subject: input('ow-claim-subject', 'Subject'), predicate: input('ow-claim-predicate', 'Predicate'), object: input('ow-claim-object', 'Object'),
        statement: textarea('ow-claim-statement', 'Atomic claim statement'), source: input('ow-claim-source', 'Asserting source ID'), nature: select('ow-claim-nature', ['primary', 'secondary', 'unknown']),
        passages: input('ow-claim-passages', 'Passage IDs, comma-separated'), supporting: input('ow-claim-support', 'Supporting evidence IDs'), contradicting: input('ow-claim-contra', 'Contradicting evidence IDs'),
        confidence: input('ow-claim-confidence', 'Confidence 0–100', 'number'), calibration: textarea('ow-claim-calibration', 'Calibration basis'),
    };
    claim.confidence.min = '0'; claim.confidence.max = '100'; claim.confidence.value = '50';
    body.appendChild(formSection('CREATE TRACEABLE CLAIM', Object.values(claim), 'CREATE CLAIM', async submit => {
        submit.disabled = true;
        try {
            await post('/claims', { case_id: claim.caseID.value.trim(), subject: claim.subject.value.trim(), predicate: claim.predicate.value.trim(), object: claim.object.value.trim(), statement: claim.statement.value.trim(), asserting_source_id: claim.source.value.trim(), source_nature: claim.nature.value, passage_ids: ids(claim.passages.value), supporting_evidence_ids: ids(claim.supporting.value), contradicting_evidence_ids: ids(claim.contradicting.value), confidence: Number.parseInt(claim.confidence.value, 10), calibration_basis: claim.calibration.value.trim(), status: 'unverified' });
            await refresh(); flash('Traceable claim created.', 'success');
        } catch (error) { flash(`Claim creation failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const review = { claimID: input('ow-review-claim', 'Claim ID'), status: select('ow-review-status', ['corroborated', 'verified', 'disputed', 'rejected']), reason: textarea('ow-review-reason', 'Review rationale') };
    body.appendChild(formSection('REVIEW CLAIM', Object.values(review), 'RECORD REVIEW', async submit => {
        submit.disabled = true;
        try { await post(`/claims/${encodeURIComponent(review.claimID.value.trim())}/review`, { status: review.status.value, actor: 'operator-workbench', reason: review.reason.value.trim() }); await refresh(); flash('Claim review recorded.', 'success'); }
        catch (error) { flash(`Claim review failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const hypothesis = { caseID: input('ow-hyp-case', 'Case ID'), statement: textarea('ow-hyp-statement', 'Hypothesis statement'), confidence: input('ow-hyp-confidence', 'Confidence 0–100', 'number'), alternatives: input('ow-hyp-alternatives', 'Alternative hypothesis IDs'), conditions: textarea('ow-hyp-conditions', 'Conditions that would change the assessment') };
    hypothesis.confidence.min = '0'; hypothesis.confidence.max = '100'; hypothesis.confidence.value = '50';
    body.appendChild(formSection('CREATE COMPETING HYPOTHESIS', Object.values(hypothesis), 'CREATE HYPOTHESIS', async submit => {
        submit.disabled = true;
        try {
            await post('/hypotheses', { case_id: hypothesis.caseID.value.trim(), statement: hypothesis.statement.value.trim(), confidence: Number.parseInt(hypothesis.confidence.value, 10), alternative_hypothesis_ids: ids(hypothesis.alternatives.value), change_conditions: ids(hypothesis.conditions.value) });
            await refresh(); flash('Hypothesis added to the board.', 'success');
        } catch (error) { flash(`Hypothesis creation failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const reassessment = { hypothesisID: input('ow-reassess-hypothesis', 'Hypothesis ID'), confidence: input('ow-reassess-confidence', 'New confidence 0–100', 'number'), reason: textarea('ow-reassess-reason', 'Reason for confidence change') };
    reassessment.confidence.min = '0'; reassessment.confidence.max = '100';
    body.appendChild(formSection('REASSESS HYPOTHESIS', Object.values(reassessment), 'UPDATE CONFIDENCE', async submit => {
        submit.disabled = true;
        try { await post(`/hypotheses/${encodeURIComponent(reassessment.hypothesisID.value.trim())}/confidence`, { confidence: Number.parseInt(reassessment.confidence.value, 10), reason: reassessment.reason.value.trim(), actor: 'operator-workbench' }); await refresh(); flash('Confidence history updated.', 'success'); }
        catch (error) { flash(`Reassessment failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const assessment = { hypothesisID: input('ow-assess-hypothesis', 'Hypothesis ID'), evidenceID: input('ow-assess-evidence', 'Case evidence ID'), compatibility: select('ow-assess-compatibility', ['2', '1', '0', '-1', '-2']), diagnosticity: input('ow-assess-diagnosticity', 'Diagnosticity 0–100', 'number'), reason: textarea('ow-assess-reason', 'Why this evidence supports or contradicts') };
    assessment.diagnosticity.min = '0'; assessment.diagnosticity.max = '100'; assessment.diagnosticity.value = '50';
    body.appendChild(formSection('ASSESS HYPOTHESIS EVIDENCE', Object.values(assessment), 'RECORD ACH CELL', async submit => {
        submit.disabled = true;
        try { await post(`/hypotheses/${encodeURIComponent(assessment.hypothesisID.value.trim())}/evidence`, { evidence_id: assessment.evidenceID.value.trim(), compatibility: Number.parseInt(assessment.compatibility.value, 10), diagnosticity: Number.parseInt(assessment.diagnosticity.value, 10), reason: assessment.reason.value.trim(), actor: 'operator-workbench' }); await refresh(); flash('Evidence assessment recorded.', 'success'); }
        catch (error) { flash(`Evidence assessment failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const gap = { caseID: input('ow-gap-case', 'Case ID'), question: textarea('ow-gap-question', 'Unanswered intelligence question'), priority: select('ow-gap-priority', ['medium', 'high', 'critical', 'low']), rationale: textarea('ow-gap-rationale', 'Why this gap matters') };
    body.appendChild(formSection('CREATE INFORMATION GAP', Object.values(gap), 'CREATE GAP', async submit => {
        submit.disabled = true;
        try { await post('/gaps', { case_id: gap.caseID.value.trim(), question: gap.question.value.trim(), priority: gap.priority.value, rationale: gap.rationale.value.trim() }); await refresh(); flash('Information gap created.', 'success'); }
        catch (error) { flash(`Gap creation failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const plan = { caseID: input('ow-plan-case', 'Case ID'), gapID: input('ow-plan-gap', 'Information gap ID'), sources: input('ow-plan-sources', 'Source types, comma-separated'), queries: textarea('ow-plan-queries', 'Collection queries, comma-separated'), constraints: input('ow-plan-constraints', 'Constraints, comma-separated'), owner: select('ow-plan-owner', ['collector', 'case_worker', 'operator']) };
    body.appendChild(formSection('CREATE COLLECTION PLAN', Object.values(plan), 'CREATE PLAN', async submit => {
        submit.disabled = true;
        try { await post('/collection-plans', { case_id: plan.caseID.value.trim(), information_gap_id: plan.gapID.value.trim(), source_types: ids(plan.sources.value), queries: ids(plan.queries.value), constraints: ids(plan.constraints.value), owner_profile: plan.owner.value }); await refresh(); flash('Collection plan proposed.', 'success'); }
        catch (error) { flash(`Collection plan failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const lineage = { upstream: input('ow-lineage-upstream', 'Upstream source ID'), downstream: input('ow-lineage-downstream', 'Downstream source ID'), relationship: select('ow-lineage-relation', ['republication', 'quotation', 'syndication', 'common_origin', 'independent', 'primary']), evidence: input('ow-lineage-evidence', 'Evidence IDs, comma-separated'), confidence: input('ow-lineage-confidence', 'Confidence 0–100', 'number') };
    lineage.confidence.min = '0'; lineage.confidence.max = '100'; lineage.confidence.value = '70';
    body.appendChild(formSection('RECORD SOURCE LINEAGE', Object.values(lineage), 'CREATE LINEAGE EDGE', async submit => {
        submit.disabled = true;
        try { await post('/lineage', { upstream_source_id: lineage.upstream.value.trim(), downstream_source_id: lineage.downstream.value.trim(), relationship: lineage.relationship.value, evidence_ids: ids(lineage.evidence.value), confidence: Number.parseInt(lineage.confidence.value, 10), detected_by: 'operator-workbench' }); await refresh(); flash('Source lineage edge recorded.', 'success'); }
        catch (error) { flash(`Lineage creation failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));

    const achCase = input('ow-ach-case', 'Case ID for ACH matrix');
    body.appendChild(formSection('BUILD ACH MATRIX', [achCase], 'BUILD MATRIX', async submit => {
        submit.disabled = true;
        try {
            const matrix = await get(`/ach?case_id=${encodeURIComponent(achCase.value.trim())}`);
            renderACH(matrix); flash('ACH matrix recalculated from current evidence assessments.', 'success');
        } catch (error) { flash(`ACH calculation failed: ${error.message}`, 'error'); }
        finally { submit.disabled = false; }
    }));
    return root;
}

function renderACH(matrix) {
    renderList('ow-ach-matrix', matrix.rows, row => record(row.statement, [`RANK ${row.rank}`, `INCONSISTENCY ${row.inconsistency_score}`, `${row.missing_evidence} MISSING`, `${(row.assessments || []).length} ASSESSED`], [], (row.assessments || []).map(cell => `${cell.evidence_id}: C${cell.compatibility}/D${cell.diagnosticity}`).join(' · ')));
}
