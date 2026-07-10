// STATUS: DIAMANT VGT SUPREME

function detailRow(label, value) {
    const row = document.createElement('div');
    row.className = 'run-approval-detail';
    const key = document.createElement('span');
    key.textContent = label;
    const content = document.createElement('strong');
    content.textContent = String(value || '—');
    row.append(key, content);
    return row;
}

export function requestRunApproval(challenge) {
    return new Promise(resolve => {
        const overlay = document.createElement('div');
        overlay.className = 'run-approval-overlay';
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        overlay.setAttribute('aria-labelledby', 'run-approval-title');

        const panel = document.createElement('section');
        panel.className = 'run-approval-panel';
        const eyebrow = document.createElement('div');
        eyebrow.className = 'run-approval-eyebrow';
        eyebrow.textContent = 'ZERO-TRUST EXECUTION GATE';
        const title = document.createElement('h2');
        title.id = 'run-approval-title';
        title.textContent = 'Einmalfreigabe erforderlich';
        const copy = document.createElement('p');
        copy.textContent = 'Aethel pausiert den Run, bis du diese konkrete Aktion geprüft hast. Die Freigabe ist signiert, kurzlebig und nur für diesen Tool-Aufruf gültig.';

        const details = document.createElement('div');
        details.className = 'run-approval-details';
        details.append(detailRow('TOOL', challenge.tool_name), detailRow('CAPABILITY', challenge.capability), detailRow('RISIKO', challenge.risk_level));
        const argsLabel = document.createElement('span');
        argsLabel.className = 'run-approval-args-label';
        argsLabel.textContent = 'GEBUNDENE ARGUMENTE';
        const args = document.createElement('pre');
        args.className = 'run-approval-args';
        args.textContent = JSON.stringify(challenge.tool_args || {}, null, 2);

        const actions = document.createElement('div');
        actions.className = 'run-approval-actions';
        const reject = document.createElement('button');
        reject.type = 'button';
        reject.className = 'cyber-button run-approval-reject';
        reject.textContent = 'RUN ABBRECHEN';
        const approve = document.createElement('button');
        approve.type = 'button';
        approve.className = 'cyber-button run-approval-confirm';
        approve.textContent = 'EINMALIG FREIGEBEN';
        actions.append(reject, approve);
        panel.append(eyebrow, title, copy, details, argsLabel, args, actions);
        overlay.appendChild(panel);
        document.body.appendChild(overlay);

        const onKeyDown = event => { if (event.key === 'Escape') finish(false); };
        const finish = approved => {
            document.removeEventListener('keydown', onKeyDown);
            overlay.remove();
            resolve(approved);
        };
        reject.addEventListener('click', () => finish(false), { once: true });
        approve.addEventListener('click', () => finish(true), { once: true });
        document.addEventListener('keydown', onKeyDown);
        approve.focus();
    });
}
