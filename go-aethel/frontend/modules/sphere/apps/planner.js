// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/planner.js
// Purpose: Master Planner, Goals, Milestones, Task Dependencies, and Agent Run Integration

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';

let plansList = [];
let activePlan = null;

/**
 * Render Master Planner inside a window body
 * @param {HTMLElement} body
 */
export async function renderPlannerApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-planner-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:10px; color:#fff;';

    // Topbar
    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(0,240,255,0.2); padding-bottom:8px;';

    const planSelect = document.createElement('select');
    planSelect.id = 'sphere-planner-select';
    planSelect.style.cssText = 'background:rgba(0,0,0,0.4); border:1px solid rgba(0,240,255,0.3); color:#fff; padding:4px 8px; border-radius:4px; font-size:11px; font-family:var(--font-mono); outline:none;';

    const rightActions = document.createElement('div');
    rightActions.style.cssText = 'display:flex; gap:6px;';

    const btnNewPlan = document.createElement('button');
    btnNewPlan.type = 'button';
    btnNewPlan.className = 'cyber-button';
    btnNewPlan.style.cssText = 'font-size:10px; padding:4px 10px; background:rgba(0,240,255,0.15); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnNewPlan.textContent = '+ NEUER PLAN';
    btnNewPlan.addEventListener('click', () => openNewPlanModal());

    rightActions.appendChild(btnNewPlan);
    header.append(planSelect, rightActions);

    const content = document.createElement('div');
    content.className = 'sphere-planner-content';
    content.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:12px; padding-right:4px;';

    container.append(header, content);
    body.appendChild(container);

    async function loadPlans() {
        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/plans`);
            if (res.ok) {
                const data = await res.json();
                plansList = data.plans || [];
                if (plansList.length === 0) {
                    await createDefaultSamplePlan();
                    return;
                }
                activePlan = plansList[0];
                updatePlanSelect();
                renderPlanDetails();
            }
        } catch (err) {
            console.error('Failed to load plans', err);
        }
    }

    function updatePlanSelect() {
        planSelect.replaceChildren();
        plansList.forEach(p => {
            const opt = document.createElement('option');
            opt.value = p.id;
            opt.textContent = `${p.title} (${p.progress_pct || 0}%)`;
            if (activePlan && p.id === activePlan.id) opt.selected = true;
            planSelect.appendChild(opt);
        });
    }

    planSelect.addEventListener('change', () => {
        activePlan = plansList.find(p => p.id === planSelect.value);
        renderPlanDetails();
    });

    function renderPlanDetails() {
        content.replaceChildren();
        if (!activePlan) return;

        // 1. Plan Overview Card
        const overview = document.createElement('article');
        overview.className = 'glass-card';
        overview.style.cssText = 'background:linear-gradient(135deg, rgba(0,240,255,0.08), rgba(0,0,0,0.4)); border:1px solid rgba(0,240,255,0.25); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:8px;';

        const topRow = document.createElement('div');
        topRow.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

        const pTitle = document.createElement('strong');
        pTitle.style.cssText = 'font-size:14px; color:#fff;';
        pTitle.textContent = activePlan.title;

        const pStatus = document.createElement('span');
        pStatus.style.cssText = 'font-size:9px; background:rgba(0,240,255,0.15); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); padding:2px 6px; border-radius:4px; text-transform:uppercase;';
        pStatus.textContent = activePlan.status || 'ACTIVE';

        topRow.append(pTitle, pStatus);

        const pObj = document.createElement('p');
        pObj.style.cssText = 'margin:0; font-size:11px; color:rgba(255,255,255,0.8); line-height:1.4;';
        pObj.textContent = activePlan.objective;

        // Progress Bar
        const progBox = document.createElement('div');
        progBox.style.cssText = 'display:flex; flex-direction:column; gap:4px;';
        const progLabel = document.createElement('small');
        progLabel.style.cssText = 'font-size:9px; color:var(--vgt-text-dim); display:flex; justify-content:space-between;';
        progLabel.innerHTML = `<span>FORTSCHRITT</span><span>${activePlan.progress_pct || 0}%</span>`;

        const progTrack = document.createElement('div');
        progTrack.style.cssText = 'height:6px; background:rgba(255,255,255,0.1); border-radius:3px; overflow:hidden;';
        const progFill = document.createElement('div');
        progFill.style.cssText = `height:100%; width:${activePlan.progress_pct || 0}%; background:var(--vgt-cyan);`;
        progTrack.appendChild(progFill);
        progBox.append(progLabel, progTrack);

        overview.append(topRow, pObj, progBox);
        content.appendChild(overview);

        // 2. Milestones & Tasks Section
        const mSection = document.createElement('section');
        mSection.style.cssText = 'display:flex; flex-direction:column; gap:8px;';
        mSection.innerHTML = `<strong style="font-size:11px; color:var(--vgt-cyan);">🎯 MEILENSTEINE & AUFGABEN</strong>`;

        const milestones = activePlan.milestones || [];
        milestones.forEach((m, idx) => {
            const mCard = document.createElement('div');
            mCard.style.cssText = 'background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.06); border-radius:6px; padding:10px; display:flex; flex-direction:column; gap:6px;';

            const mTop = document.createElement('div');
            mTop.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

            const mName = document.createElement('span');
            mName.style.cssText = 'font-size:11px; font-weight:bold; color:#fff; display:flex; align-items:center; gap:6px;';
            mName.innerHTML = `<span style="color:${m.completed ? 'var(--vgt-green)' : 'var(--vgt-orange)'};">${m.completed ? '✓' : '○'}</span> Meilenstein ${idx+1}: ${m.title}`;

            mTop.appendChild(mName);
            mCard.appendChild(mTop);
            mSection.appendChild(mCard);
        });

        content.appendChild(mSection);
    }

    async function createDefaultSamplePlan() {
        const defaultPlan = {
            title: 'Aethel Beta 4 Release',
            objective: 'Sphere 2.0 Personal Operations Workspace fertigstellen, Security-Audit bestehen & Release',
            status: 'active',
            progress_pct: 60,
            milestones: [
                { id: 'M1', title: 'Sphere 2.0 Backend & Object Store', completed: true, order: 1 },
                { id: 'M2', title: 'Modular Frontend Window Manager & Apps', completed: true, order: 2 },
                { id: 'M3', title: 'Sicherheits-Audit & Verifikation (Diamant)', completed: false, order: 3 }
            ]
        };
        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/plans`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(defaultPlan)
            });
            if (res.ok) {
                const created = await res.json();
                plansList = [created];
                activePlan = created;
                updatePlanSelect();
                renderPlanDetails();
            }
        } catch (e) {
            console.error('Failed to create sample plan', e);
        }
    }

    function openNewPlanModal() {
        let dialog = document.getElementById('sphere-new-plan-dialog');
        if (dialog) dialog.remove();

        dialog = document.createElement('div');
        dialog.id = 'sphere-new-plan-dialog';
        dialog.style.cssText = 'position:fixed; inset:0; z-index:10080; background:rgba(3,6,15,0.85); backdrop-filter:blur(10px); display:flex; justify-content:center; align-items:center; font-family:var(--font-mono);';

        const modal = document.createElement('div');
        modal.className = 'glass-card';
        modal.style.cssText = 'width:90%; max-width:480px; background:linear-gradient(145deg, rgba(8,16,32,0.98), rgba(4,6,14,0.99)); border:1px solid var(--vgt-cyan); border-radius:12px; padding:20px; box-shadow:0 0 40px rgba(0,240,255,0.25); display:flex; flex-direction:column; gap:12px; color:#fff;';

        modal.innerHTML = `
            <div style="display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(0,240,255,0.2); padding-bottom:10px;">
                <strong style="color:var(--vgt-cyan); font-size:13px;">⏰ NEUEN PLAN ERSTELLEN</strong>
                <button id="plan-modal-close" style="background:none; border:none; color:var(--vgt-text-dim); font-size:16px; cursor:pointer;">✕</button>
            </div>
            <div style="display:flex; flex-direction:column; gap:10px; font-size:11px;">
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">PLAN-TITEL</label>
                    <input id="plan-input-title" type="text" placeholder="z.B. GaiaCom Markteinführung" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                </div>
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">HAUPTZIEL / OBJECTIVE</label>
                    <textarea id="plan-input-obj" rows="3" placeholder="z.B. Launch der sicheren Kommunikationsplattform im Oktober..." style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;"></textarea>
                </div>
            </div>
            <div style="display:flex; justify-content:flex-end; gap:8px; margin-top:8px;">
                <button id="plan-modal-cancel" class="cyber-button" style="font-size:10px; padding:6px 12px; background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.1); color:#fff; cursor:pointer; width:auto;">ABBRECHEN</button>
                <button id="plan-modal-save" class="cyber-button" style="font-size:10px; padding:6px 14px; background:rgba(0,240,255,0.2); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;">PLAN SPEICHERN</button>
            </div>
        `;

        dialog.appendChild(modal);
        document.body.appendChild(dialog);

        dialog.querySelector('#plan-modal-close').addEventListener('click', () => dialog.remove());
        dialog.querySelector('#plan-modal-cancel').addEventListener('click', () => dialog.remove());

        dialog.querySelector('#plan-modal-save').addEventListener('click', async () => {
            const title = dialog.querySelector('#plan-input-title').value.trim();
            const obj = dialog.querySelector('#plan-input-obj').value.trim();

            if (!title || !obj) {
                alert('Bitte Titel und Ziel eingeben.');
                return;
            }

            try {
                const res = await fetch(`${state.API_BASE}/v1/sphere/plans`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title: title, objective: obj, status: 'active' })
                });
                if (res.ok) {
                    dialog.remove();
                    await loadPlans();
                }
            } catch (e) {
                alert('Fehler: ' + e.message);
            }
        });
    }

    void loadPlans();
}
