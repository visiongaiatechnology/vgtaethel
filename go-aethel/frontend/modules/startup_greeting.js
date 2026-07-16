// STATUS: DIAMANT VGT SUPREME

import * as api from './api.js';
import { speak } from './voice.js';

const GREETINGS = Object.freeze({
    de: Object.freeze({
        personalized: name => `Core initialisiert. Willkommen zurück, ${name}.`,
        fallback: 'Core initialisiert. Willkommen bei Aethel.'
    }),
    en: Object.freeze({
        personalized: name => `Core initialized. Welcome back, ${name}.`,
        fallback: 'Core initialized. Welcome to Aethel.'
    }),
    es: Object.freeze({
        personalized: name => `Núcleo inicializado. Te doy la bienvenida de nuevo, ${name}.`,
        fallback: 'Núcleo inicializado. Te doy la bienvenida a Aethel.'
    }),
    ru: Object.freeze({
        personalized: name => `Ядро инициализировано. С возвращением, ${name}.`,
        fallback: 'Ядро инициализировано. Добро пожаловать в Aethel.'
    })
});

let greetingSpoken = false;

function activeGreeting() {
    const language = String(localStorage.getItem('aethel_ui_language') || 'de').toLowerCase();
    return GREETINGS[language] || GREETINGS.de;
}

function normalizeDisplayName(value) {
    return String(value || '')
        .replace(/[\u0000-\u001F\u007F]/gu, ' ')
        .replace(/\s+/gu, ' ')
        .trim()
        .slice(0, 80);
}

export async function speakPersonalizedStartupGreeting() {
    if (greetingSpoken) return;
    greetingSpoken = true;

    const greeting = activeGreeting();
    let text = greeting.fallback;
    try {
        const status = await api.getPersonalStatus();
        const displayName = normalizeDisplayName(status?.profile?.display_name);
        if (displayName) text = greeting.personalized(displayName);
    } catch (error) {
        console.warn('Personalized startup greeting unavailable; using local fallback.', error);
    }

    await speak(text);
}
