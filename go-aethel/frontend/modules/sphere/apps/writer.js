// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/writer.js
// Purpose: VGT Writer 2.0 with Rich Text Canvas, Track Changes Diff Review, Version Snapshots & Export

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';
import { sanitizeRichText, htmlToMarkdownRough, buildSphereDocumentExport } from '../../sphere.js';

let activeDocumentID = 'DOC-DEFAULT';

/**
 * Render VGT Writer 2.0 inside a window body
 * @param {HTMLElement} body
 */
export async function renderWriterApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-writer-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:8px; color:#fff;';

    // 1. Toolbar (WYSIWYG formatting, templates, export, Send to)
    const toolbar = document.createElement('div');
    toolbar.className = 'sphere-writer-toolbar';
    toolbar.style.cssText = 'display:flex; flex-wrap:wrap; gap:4px; align-items:center; background:rgba(0,0,0,0.3); padding:6px 8px; border-radius:6px; border:1px solid rgba(0,240,255,0.15);';

    const tools = [
        { label: 'B', title: 'Fett', cmd: 'bold', isBold: true },
        { label: 'I', title: 'Kursiv', cmd: 'italic', isItalic: true },
        { label: 'U', title: 'Unterstreichen', cmd: 'underline', isUnderline: true },
        { sep: true },
        { label: 'H1', title: 'Überschrift 1', cmd: 'formatBlock', val: 'h1' },
        { label: 'H2', title: 'Überschrift 2', cmd: 'formatBlock', val: 'h2' },
        { label: 'H3', title: 'Überschrift 3', cmd: 'formatBlock', val: 'h3' },
        { label: 'P', title: 'Absatz', cmd: 'formatBlock', val: 'p' },
        { sep: true },
        { label: '• Liste', title: 'Aufzählung', cmd: 'insertUnorderedList' },
        { label: '1. Liste', title: 'Nummerierung', cmd: 'insertOrderedList' },
        { sep: true },
        { label: 'Align L', title: 'Linksbündig', cmd: 'justifyLeft' },
        { label: 'Align C', title: 'Zentriert', cmd: 'justifyCenter' },
        { label: 'Align R', title: 'Rechtsbündig', cmd: 'justifyRight' }
    ];

    tools.forEach(t => {
        if (t.sep) {
            const sepEl = document.createElement('span');
            sepEl.style.cssText = 'width:1px; height:14px; background:rgba(255,255,255,0.1); margin:0 2px;';
            toolbar.appendChild(sepEl);
            return;
        }
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'toolbar-btn';
        btn.title = t.title;
        btn.style.cssText = 'background:none; border:1px solid transparent; color:#fff; font-size:11px; padding:2px 6px; border-radius:4px; cursor:pointer;';
        if (t.isBold) btn.style.fontWeight = 'bold';
        if (t.isItalic) btn.style.fontStyle = 'italic';
        if (t.isUnderline) btn.style.textDecoration = 'underline';
        btn.textContent = t.label;

        btn.addEventListener('click', (e) => {
            e.preventDefault();
            const editor = document.getElementById('sphere-editor-area');
            if (editor) {
                editor.focus();
                document.execCommand(t.cmd, false, t.val || null);
            }
        });
        toolbar.appendChild(btn);
    });

    // Send To Button & Export Dropdown
    const rightActions = document.createElement('div');
    rightActions.style.cssText = 'margin-left:auto; display:flex; gap:4px; align-items:center;';

    const btnSendTo = document.createElement('button');
    btnSendTo.type = 'button';
    btnSendTo.className = 'cyber-button';
    btnSendTo.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(0,240,255,0.12); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnSendTo.textContent = '➔ SEND TO...';
    btnSendTo.addEventListener('click', () => {
        const titleVal = document.getElementById('sphere-writer-title')?.value || 'Dokument';
        const editor = document.getElementById('sphere-editor-area');
        contextBus.openSendToDialog({
            sourceApp: 'writer',
            title: titleVal,
            content: editor ? editor.innerHTML : ''
        });
    });

    const btnExportMD = document.createElement('button');
    btnExportMD.type = 'button';
    btnExportMD.className = 'toolbar-btn';
    btnExportMD.style.cssText = 'font-size:10px; padding:2px 6px; background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.1); color:#fff; border-radius:4px; cursor:pointer;';
    btnExportMD.textContent = '↓ MD';
    btnExportMD.title = 'Als Markdown exportieren';
    btnExportMD.addEventListener('click', () => exportDocument('md'));

    const btnExportHTML = document.createElement('button');
    btnExportHTML.type = 'button';
    btnExportHTML.className = 'toolbar-btn';
    btnExportHTML.style.cssText = 'font-size:10px; padding:2px 6px; background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.1); color:#fff; border-radius:4px; cursor:pointer;';
    btnExportHTML.textContent = '↓ HTML';
    btnExportHTML.title = 'Als HTML exportieren';
    btnExportHTML.addEventListener('click', () => exportDocument('html'));

    rightActions.append(btnSendTo, btnExportMD, btnExportHTML);
    toolbar.appendChild(rightActions);

    // 2. Document Title & Save State Bar
    const titleBar = document.createElement('div');
    titleBar.style.cssText = 'display:flex; justify-content:space-between; align-items:center; gap:8px;';

    const titleInput = document.createElement('input');
    titleInput.id = 'sphere-writer-title';
    titleInput.type = 'text';
    titleInput.value = localStorage.getItem('aethel_sphere_document_title') || 'Aethel Document';
    titleInput.maxLength = 120;
    titleInput.style.cssText = 'flex:1; background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.1); border-radius:6px; color:#fff; padding:5px 10px; font-size:12px; font-weight:bold; font-family:var(--font-mono); outline:none;';
    titleInput.addEventListener('input', () => {
        localStorage.setItem('aethel_sphere_document_title', titleInput.value.trim());
    });

    const saveStatus = document.createElement('span');
    saveStatus.id = 'sphere-writer-save-state';
    saveStatus.style.cssText = 'font-size:10px; color:var(--vgt-cyan); letter-spacing:0.05em;';
    saveStatus.textContent = 'BEREIT';

    titleBar.append(titleInput, saveStatus);

    // 3. Track Changes Diff Notification Banner (renders if pending proposals exist)
    const diffsContainer = document.createElement('div');
    diffsContainer.id = 'sphere-writer-diffs-container';
    diffsContainer.style.cssText = 'display:none; flex-direction:column; gap:6px;';

    // 4. Main Editable Document Body
    const editorArea = document.createElement('div');
    editorArea.id = 'sphere-editor-area';
    editorArea.className = 'sphere-window-body sphere-editor-body';
    editorArea.contentEditable = 'true';
    editorArea.setAttribute('placeholder', 'Beginne mit dem Schreiben oder weise Aethel an, den Writer zu verwenden...');
    editorArea.style.cssText = 'flex:1; overflow-y:auto; padding:16px 20px; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.08); border-radius:8px; font-size:13px; line-height:1.6; color:rgba(255,255,255,0.9); outline:none;';

    // 5. Status Bar Metrics
    const statusbar = document.createElement('footer');
    statusbar.className = 'sphere-writer-statusbar font-mono';
    statusbar.style.cssText = 'display:flex; justify-content:space-between; align-items:center; padding:4px 8px; font-size:10px; color:var(--vgt-text-dim); border-top:1px solid rgba(255,255,255,0.05);';

    const wordCount = document.createElement('span');
    wordCount.id = 'sphere-writer-word-count';
    wordCount.textContent = '0 WÖRTER';

    const charCount = document.createElement('span');
    charCount.id = 'sphere-writer-char-count';
    charCount.textContent = '0 ZEICHEN';

    const readTime = document.createElement('span');
    readTime.id = 'sphere-writer-read-time';
    readTime.textContent = '0 MIN LESEZEIT';

    const syncLabel = document.createElement('span');
    syncLabel.style.color = 'var(--vgt-cyan)';
    syncLabel.textContent = 'AETHEL LIVE-SYNC ●';

    statusbar.append(wordCount, charCount, readTime, syncLabel);

    container.append(toolbar, titleBar, diffsContainer, editorArea, statusbar);
    body.appendChild(container);

    // Auto-save logic
    let saveTimeout = null;
    const saveDocument = async () => {
        const clean = sanitizeRichText(editorArea.innerHTML);
        saveStatus.textContent = 'SPEICHERT …';
        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/document`, {
                method: 'POST',
                headers: { 'Content-Type': 'text/html' },
                body: clean
            });
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            saveStatus.textContent = 'GESPEICHERT';
        } catch (err) {
            saveStatus.textContent = 'SPEICHERFEHLER';
        }
    };

    const triggerDebouncedSave = () => {
        if (saveTimeout) clearTimeout(saveTimeout);
        saveTimeout = setTimeout(saveDocument, 800);
    };

    editorArea.addEventListener('input', () => {
        saveStatus.textContent = 'UNGESPEICHERT';
        updateMetrics();
        triggerDebouncedSave();
    });

    // Sanitized paste handler
    editorArea.addEventListener('paste', (e) => {
        e.preventDefault();
        const text = (e.originalEvent || e).clipboardData.getData('text/plain');
        const cleanText = text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
        document.execCommand('insertHTML', false, cleanText);
        triggerDebouncedSave();
    });

    // Load initial document
    try {
        const docRes = await fetch(`${state.API_BASE}/v1/sphere/document`);
        if (docRes.ok) {
            editorArea.innerHTML = sanitizeRichText(await docRes.text());
            updateMetrics();
        }
    } catch (err) {
        console.warn('Initial document load failed', err);
    }

    function updateMetrics() {
        const text = (editorArea.textContent || '').trim();
        const words = text ? text.split(/\s+/u).filter(Boolean).length : 0;
        const chars = Array.from(text).length;
        const minutes = words ? Math.max(1, Math.ceil(words / 220)) : 0;
        wordCount.textContent = `${words.toLocaleString()} WÖRTER`;
        charCount.textContent = `${chars.toLocaleString()} ZEICHEN`;
        readTime.textContent = `${minutes} MIN LESEZEIT`;
    }

    function exportDocument(format) {
        const pack = buildSphereDocumentExport(editorArea.innerHTML, format);
        const title = titleInput.value || 'aethel-document';
        const filename = `${title.replace(/[^a-zA-Z0-9._-]+/g, '-')}.${format === 'md' ? 'md' : 'html'}`;
        const blob = new Blob([pack.body], { type: pack.mime });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        setTimeout(() => URL.revokeObjectURL(url), 2000);
    }
}
