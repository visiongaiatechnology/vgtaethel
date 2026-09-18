// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere.js
// Purpose: Facade & Export Hub for Aethel Sphere 2.0 Personal Operations Desktop

import { state } from './state.js';
import { setupSphereWorkspace as setupSphere2, openSphereApp as openSphereApp2 } from './sphere/index.js';
import { focusWindow as focusWin, makeDraggable as makeDrag, makeResizable as makeResize } from './sphere/window_manager.js';

// Strict HTML Sanitizer to prevent XSS (Cross-Site Scripting) - required by Goja tests
export function sanitizeRichText(html) {
    if (!html) return "";
    
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    
    const allowedTags = new Set([
        "h1", "h2", "h3", "p", "b", "i", "u", "ul", "ol", "li", "br", "div", "span", "strong", "em", "table", "thead", "tbody", "tr", "th", "td", "blockquote", "code", "pre"
    ]);
    
    const cleanNode = (node) => {
        if (node.nodeType === Node.TEXT_NODE) {
            return document.createTextNode(node.textContent);
        }
        if (node.nodeType !== Node.ELEMENT_NODE) {
            return null;
        }
        
        const tagName = node.tagName.toLowerCase();
        if (!allowedTags.has(tagName)) {
            return document.createTextNode(node.textContent);
        }
        
        const newEl = document.createElement(tagName);
        if (node.hasAttribute("style")) {
            const styleValue = node.getAttribute("style").toLowerCase().trim();
            const allowedStyles = [];
            if (styleValue.includes("text-align: center") || styleValue.includes("text-align:center")) {
                allowedStyles.push("text-align: center;");
            } else if (styleValue.includes("text-align: right") || styleValue.includes("text-align:right")) {
                allowedStyles.push("text-align: right;");
            } else if (styleValue.includes("text-align: left") || styleValue.includes("text-align:left")) {
                allowedStyles.push("text-align: left;");
            } else if (styleValue.includes("text-align: justify") || styleValue.includes("text-align:justify")) {
                allowedStyles.push("text-align: justify;");
            }
            if (allowedStyles.length > 0) {
                newEl.setAttribute("style", allowedStyles.join(" "));
            }
        }
        
        node.childNodes.forEach(child => {
            const cleanChild = cleanNode(child);
            if (cleanChild) {
                newEl.appendChild(cleanChild);
            }
        });
        
        return newEl;
    };
    
    const fragment = document.createDocumentFragment();
    doc.body.childNodes.forEach(child => {
        const cleanChild = cleanNode(child);
        if (cleanChild) {
            fragment.appendChild(cleanChild);
        }
    });
    
    const container = document.createElement("div");
    container.appendChild(fragment);
    return container.innerHTML;
}

export function makeDraggable(windowEl, handleEl) {
    return makeDrag(windowEl, handleEl);
}

export function makeResizable(windowEl) {
    return makeResize(windowEl);
}

export function focusWindow(windowEl) {
    return focusWin(windowEl);
}

/** Pure helpers for export (unit-tested via goja/static). */
export function htmlToMarkdownRough(html) {
    const tmp = document.createElement('div');
    tmp.innerHTML = sanitizeRichText(html || '');
    const walk = (node) => {
        if (node.nodeType === Node.TEXT_NODE) return node.textContent || '';
        if (node.nodeType !== Node.ELEMENT_NODE) return '';
        const tag = node.tagName.toLowerCase();
        const inner = Array.from(node.childNodes).map(walk).join('');
        if (tag === 'h1') return `# ${inner.trim()}\n\n`;
        if (tag === 'h2') return `## ${inner.trim()}\n\n`;
        if (tag === 'h3') return `### ${inner.trim()}\n\n`;
        if (tag === 'p' || tag === 'div') return `${inner.trim()}\n\n`;
        if (tag === 'br') return '\n';
        if (tag === 'li') return `- ${inner.trim()}\n`;
        if (tag === 'strong' || tag === 'b') return `**${inner}**`;
        if (tag === 'em' || tag === 'i') return `*${inner}*`;
        return inner;
    };
    return Array.from(tmp.childNodes).map(walk).join('').trim() + '\n';
}

export function buildSphereDocumentExport(html, format) {
    const clean = sanitizeRichText(html || '');
    if (format === 'md' || format === 'markdown') {
        return { filename: 'aethel-sphere-document.md', mime: 'text/markdown;charset=utf-8', body: htmlToMarkdownRough(clean) };
    }
    const wrapped = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Aethel Document</title></head><body>${clean}</body></html>`;
    return { filename: 'aethel-sphere-document.html', mime: 'text/html;charset=utf-8', body: wrapped };
}

export function exportSphereDocument(format) {
    const editor = document.getElementById('sphere-editor-area');
    if (!editor) return null;
    const pack = buildSphereDocumentExport(editor.innerHTML, format);
    const title = document.getElementById('sphere-writer-title')?.value || 'aethel-sphere-document';
    const base = String(title).normalize('NFKC').replace(/[^\p{L}\p{N}._-]+/gu, '-').replace(/^-+|-+$/g, '').slice(0, 80) || 'aethel-sphere-document';
    pack.filename = `${base}.${format === 'md' ? 'md' : 'html'}`;
    
    const blob = new Blob([pack.body], { type: pack.mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = pack.filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 2000);
    return pack;
}

export function populateNotesApp(body) {
    if (!body) return;
    const ta = document.createElement('textarea');
    ta.className = 'sphere-notes-area';
    ta.placeholder = 'Scratch notes — lokal im Browser (nicht Nexus).';
    ta.value = localStorage.getItem('aethel_sphere_notes') || '';
    ta.style.cssText = 'width:100%;height:100%;min-height:200px;resize:none;background:rgba(0,0,0,0.35);border:1px solid rgba(255,255,255,0.08);color:#fff;padding:12px;font:12px/1.5 var(--font-mono);border-radius:8px;';
    ta.addEventListener('input', () => localStorage.setItem('aethel_sphere_notes', ta.value));
    body.appendChild(ta);
}

export function setupSphereWorkspace() {
    setupSphere2();
}

export function openSphereApp(appID) {
    openSphereApp2(appID);
}

// Global Exports
window.sanitizeRichText = sanitizeRichText;
window.htmlToMarkdownRough = htmlToMarkdownRough;
window.buildSphereDocumentExport = buildSphereDocumentExport;
window.exportSphereDocument = exportSphereDocument;
window.populateNotesApp = populateNotesApp;
window.makeDraggable = makeDraggable;
window.makeResizable = makeResizable;
window.focusWindow = focusWindow;
window.setupSphereWorkspace = setupSphereWorkspace;
window.openSphereApp = openSphereApp;
