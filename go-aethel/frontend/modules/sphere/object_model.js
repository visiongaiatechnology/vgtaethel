// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/object_model.js
// Purpose: Universal Sphere Object Model definitions, client validation, and helpers

export const ObjectType = {
    DOCUMENT: 'document',
    TRIP: 'trip',
    PLAN: 'plan',
    TASK: 'task',
    PROJECT: 'project',
    RESEARCH_ITEM: 'research_item',
    PLACE: 'place',
    PERSON: 'person',
    SOURCE: 'source',
    EVENT: 'event',
    ALERT: 'alert',
    FILE: 'file',
    BOOKING: 'booking',
    CONVERSATION: 'conversation',
    RUN: 'run'
};

export const VirtualDesktop = {
    PERSONAL: 'PERSONAL',
    WORK: 'WORK',
    RESEARCH: 'RESEARCH',
    TRAVEL: 'TRAVEL',
    PROJECT: 'PROJECT',
    INCIDENT: 'INCIDENT'
};

/**
 * Generate a unique typed entity ID
 * @param {string} prefix
 * @returns {string}
 */
export function generateEntityID(prefix = 'OBJ') {
    const ts = new Date().toISOString().replace(/[-:T]/g, '').slice(0, 14);
    const rand = Math.floor(Math.random() * 1000000).toString().padStart(6, '0');
    return `${prefix.toUpperCase()}-${ts}-${rand}`;
}

/**
 * Clean text for safe DOM node insertion (preventing XSS)
 * @param {string} raw
 * @returns {string}
 */
export function escapeText(raw) {
    if (!raw) return '';
    return String(raw)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

/**
 * Build a Universal SendTo payload
 */
export function createSendToPayload(sourceApp, targetApp, title, content, objectType = ObjectType.DOCUMENT, metadata = {}) {
    return {
        source_app: sourceApp,
        target_app: targetApp,
        object_type: objectType,
        title: title || 'Neues Element',
        content: content || '',
        metadata: metadata
    };
}
