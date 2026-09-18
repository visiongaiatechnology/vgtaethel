// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/terminal.js
// Purpose: Terminal Execution & Background Task Inspection in Sphere

import { state } from '../../state.js';

export function renderTerminalApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:8px; height:100%; color:#fff;';

    const controls = document.createElement('div');
    controls.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

    const btnGroup = document.createElement('div');
    btnGroup.style.cssText = 'display:flex; gap:6px;';

    const btnTasks = document.createElement('button');
    btnTasks.type = 'button';
    btnTasks.className = 'cyber-button';
    btnTasks.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnTasks.textContent = 'TASKS';

    const btnRuns = document.createElement('button');
    btnRuns.type = 'button';
    btnRuns.className = 'cyber-button';
    btnRuns.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(138,43,226,0.1); border:1px solid var(--vgt-purple); color:var(--vgt-purple); cursor:pointer; width:auto;';
    btnRuns.textContent = 'RUNS';

    btnGroup.append(btnTasks, btnRuns);

    const termStatus = document.createElement('span');
    termStatus.style.cssText = 'font-size:9px; color:var(--vgt-text-dim);';
    termStatus.textContent = 'READY // LOCAL HOST UPLINK';

    controls.append(btnGroup, termStatus);

    const termOutput = document.createElement('div');
    termOutput.style.cssText = 'flex:1; overflow-y:auto; padding:10px; background:rgba(0,0,0,0.6); border:1px solid rgba(255,255,255,0.08); border-radius:6px; font-size:11px; color:#39ff14; line-height:1.4; white-space:pre-wrap;';
    termOutput.textContent = 'Aethel Autonomous Exec Engine. Type command or choose query...\n';

    const inputRow = document.createElement('div');
    inputRow.style.cssText = 'display:flex; align-items:center; gap:6px; background:rgba(0,0,0,0.4); padding:4px 8px; border-radius:4px; border:1px solid rgba(255,255,255,0.1);';

    const promptSymbol = document.createElement('span');
    promptSymbol.style.color = 'var(--vgt-cyan)';
    promptSymbol.textContent = '$';

    const termInput = document.createElement('input');
    termInput.type = 'text';
    termInput.placeholder = 'Execute command in shell...';
    termInput.style.cssText = 'flex:1; background:none; border:none; color:#fff; font-family:var(--font-mono); font-size:11px; outline:none;';

    const runBtn = document.createElement('button');
    runBtn.type = 'button';
    runBtn.className = 'cyber-button';
    runBtn.style.cssText = 'font-size:9px; padding:2px 8px; background:rgba(0,240,255,0.2); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    runBtn.textContent = 'RUN';

    inputRow.append(promptSymbol, termInput, runBtn);
    container.append(controls, termOutput, inputRow);
    body.appendChild(container);

    function print(text, color = '#39ff14') {
        const line = document.createElement('div');
        line.style.color = color;
        line.style.marginBottom = '2px';
        line.textContent = text;
        termOutput.appendChild(line);
        termOutput.scrollTop = termOutput.scrollHeight;
    }

    async function execute() {
        const cmd = termInput.value.trim();
        if (!cmd) return;
        termInput.value = '';
        print(`$ ${cmd}`, 'var(--vgt-cyan)');

        if (cmd.toLowerCase() === 'clear' || cmd.toLowerCase() === 'cls') {
            termOutput.textContent = '';
            return;
        }

        termStatus.textContent = 'EXECUTING...';
        try {
            const res = await fetch(`${state.API_BASE}/v1/tasks/`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ text: cmd, objective: cmd })
            });
            if (!res.ok) throw new Error(await res.text());
            const task = await res.json();
            print(`Task registriert: ${task.id}`, 'var(--vgt-green)');
        } catch (err) {
            print(`Ausführungsfehler: ${err.message}`, 'var(--vgt-red)');
        } finally {
            termStatus.textContent = 'READY // LOCAL HOST UPLINK';
        }
    }

    runBtn.addEventListener('click', () => void execute());
    termInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            void execute();
        }
    });

    btnTasks.addEventListener('click', async () => {
        try {
            const res = await fetch(`${state.API_BASE}/v1/tasks/`);
            if (res.ok) {
                const list = await res.json();
                print('=== RUNNING SYSTEM TASKS ===', 'var(--vgt-cyan)');
                if (list.length === 0) print('Keine aktiven Hintergrund-Tasks.', '#fff');
                else list.forEach(t => print(`[${t.id || 'TASK'}] ${t.text} (${t.status || 'running'})`, '#fff'));
            }
        } catch (e) {
            print(`Fehler: ${e.message}`, 'var(--vgt-red)');
        }
    });

    btnRuns.addEventListener('click', async () => {
        try {
            const res = await fetch(`${state.API_BASE}/v1/runs`);
            if (res.ok) {
                const data = await res.json();
                const runs = data.runs || [];
                print('=== ACTIVE AGENT RUNS ===', 'var(--vgt-purple)');
                if (runs.length === 0) print('Keine aktiven Agenten-Runs.', '#fff');
                else runs.forEach(r => print(`[RUN_${r.id.slice(0,6)}] ${r.objective} - ${r.status}`, '#fff'));
            }
        } catch (e) {
            print(`Fehler: ${e.message}`, 'var(--vgt-red)');
        }
    });
}
