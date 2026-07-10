import { state } from './state.js';
import * as api from './api.js';
import { requestRunApproval } from './approval_dialog.js';

let monitorTimer = null;
let monitorInFlight = false;

// One signed approval flow is shared by every surface. A pending run therefore
// cannot become invisible merely because its originating view is no longer open.
export async function requestApprovalForRun(run) {
    if (!run?.id || state.approvalPromptedRuns.has(run.id)) return false;
    state.approvalPromptedRuns.add(run.id);
    try {
        const challenge = await api.runAction(run.id, 'approval', {});
        const approved = await requestRunApproval(challenge);
        if (!approved) {
            await api.runAction(run.id, 'cancel');
            return false;
        }
        await api.runAction(run.id, 'approval', { approval_token: challenge.approval_token });
        return true;
    } catch (error) {
        console.error('Run approval could not be completed', error);
        return false;
    } finally {
        state.approvalPromptedRuns.delete(run.id);
    }
}

export async function pollPendingRunApprovals() {
    if (monitorInFlight || document.hidden) return;
    monitorInFlight = true;
    try {
        const payload = await api.getRuns();
        const pending = (payload.runs || []).find(run => run.status === 'waiting_approval' && !state.approvalPromptedRuns.has(run.id));
        if (pending) await requestApprovalForRun(pending);
    } catch (error) {
        // Core availability is already represented in the HUD; never interrupt
        // the operator with polling failures.
        console.debug('Global run approval monitor unavailable', error);
    } finally {
        monitorInFlight = false;
    }
}

export function startGlobalRunApprovalMonitor() {
    if (monitorTimer !== null) return;
    const poll = () => { void pollPendingRunApprovals(); };
    poll();
    monitorTimer = window.setInterval(poll, 900);
    document.addEventListener('visibilitychange', () => { if (!document.hidden) poll(); });
}
