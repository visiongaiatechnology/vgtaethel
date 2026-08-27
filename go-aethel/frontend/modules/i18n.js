// STATUS: DIAMANT VGT SUPREME
// Local-only UI language registry. No remote translation service or dynamic code execution.

const STORAGE_KEY = 'aethel_ui_language';
const SUPPORTED = Object.freeze(['de', 'en', 'ru', 'es', 'fr']);

const messages = Object.freeze({
    de: Object.freeze({
        'language.label': 'Sprache',
        'splash.start': 'STARTEN',
        'disclosure.title': 'BETA V3 LAUFZEIT-HINWEIS',
        'disclosure.body': 'Dies ist VGT AETHEL Beta V3 — ein souveränes strategisches Intelligence OS, entwickelt von VisionGaiaTechnology.\n\nDie Nutzung der Software erfolgt vollständig auf eigene Verantwortung. Aethel besitzt weitreichende Berechtigungen (Shell-Befehle, Dateisystem-Zugriff, Tastatur- & Maus-Simulation). Unerwartete Aktionen oder Fehlfunktionen können Datenverlust oder Systeminstabilitäten verursachen.',
        'disclosure.accept': 'VERSTANDEN & BESTÄTIGEN',
        'setup.title': 'VGT GENESIS', 'setup.subtitle': 'Erste Systemkonfiguration',
        'setup.initialize': 'SYSTEM INITIALISIEREN', 'setup.local': 'LOKALE KI STARTEN (OLLAMA)',
        'nav.assistant': 'Assistent', 'nav.core': 'Neural Core', 'nav.chat': 'Chat', 'nav.personas': 'Personas',
        'nav.agents': 'Agenten-Team', 'nav.operator': 'Live-Operator', 'nav.workspace': 'Arbeitsbereich',
        'nav.sphere': 'Sphere', 'nav.tasks': 'Automationen & Tasks', 'nav.memory': 'Nexus Memory', 'nav.personal': 'Personal Core',
        'nav.global': 'Global Watch', 'nav.globe': 'Live-Globus & Feeds', 'nav.runs': 'Run Center',
        'nav.casesSection': 'Fälle & Sicherheit', 'nav.cases': 'Cases & Evidenz', 'nav.security': 'Security & Audit',
        'nav.archive': 'Chat-Archiv', 'sidebar.model': 'KI-Modell', 'sidebar.voice': 'Stimme',
        'sidebar.persona': 'Persona', 'sidebar.newChat': '+ Neuer Chat', 'settings.title': 'Einstellungen',
        'settings.subtitle': 'API-Zugänge, Provider-Status und Systemaktionen — alles bleibt auf deinem Gerät.',
        'gw.title': 'GLOBAL WATCH', 'gw.briefing': 'Lagebriefing', 'gw.refresh': 'Feed aktualisieren',
        'gw.settings': 'Einstellungen', 'gw.feed': 'Nachrichtenfeed', 'gw.layers': 'KARTENEBENEN',
        'gw.translate': 'ÜBERSETZEN', 'gw.compose': 'BEITRAG ERSTELLEN', 'gw.discuss': 'MIT AETHEL BESPRECHEN',
    }),
    en: Object.freeze({
        'language.label': 'Language',
        'splash.start': 'START',
        'disclosure.title': 'BETA V3 RUNTIME DISCLOSURE',
        'disclosure.body': 'This is VGT AETHEL Beta V3 — a sovereign strategic Intelligence OS developed by VisionGaiaTechnology.\n\nUse of this software is entirely at your own risk. Aethel holds low-level host capabilities (terminal execution, filesystem write, GUI simulation). Unintended model behaviors or bugs may result in data loss or host instability.',
        'disclosure.accept': 'I UNDERSTAND & AGREE',
        'setup.title': 'VGT GENESIS', 'setup.subtitle': 'Initial System Configuration',
        'setup.initialize': 'INITIALIZE SYSTEM', 'setup.local': 'START LOCAL AI (OLLAMA)',
        'nav.assistant': 'Assistant', 'nav.core': 'Neural Core', 'nav.chat': 'Chat', 'nav.personas': 'Personas',
        'nav.agents': 'Agent Team', 'nav.operator': 'Live Operator', 'nav.workspace': 'Workspace',
        'nav.sphere': 'Sphere', 'nav.tasks': 'Automations & Tasks', 'nav.memory': 'Nexus Memory', 'nav.personal': 'Personal Core',
        'nav.global': 'Global Watch', 'nav.globe': 'Live Globe & Feeds', 'nav.runs': 'Run Center',
        'nav.casesSection': 'Cases & Security', 'nav.cases': 'Cases & Evidence', 'nav.security': 'Security & Audit',
        'nav.archive': 'Chat Archive', 'sidebar.model': 'AI Model', 'sidebar.voice': 'Voice',
        'sidebar.persona': 'Persona', 'sidebar.newChat': '+ New Chat', 'settings.title': 'Settings',
        'settings.subtitle': 'API access, provider health and system actions — all data stays on your device.',
        'gw.title': 'GLOBAL WATCH', 'gw.briefing': 'Situation Briefing', 'gw.refresh': 'Refresh Feed',
        'gw.settings': 'Settings', 'gw.feed': 'News Feed', 'gw.layers': 'MAP LAYERS',
        'gw.translate': 'TRANSLATE', 'gw.compose': 'CREATE ARTICLE', 'gw.discuss': 'DISCUSS WITH AETHEL',
    }),
    es: Object.freeze({
        'language.label': 'Idioma',
        'splash.start': 'INICIAR',
        'disclosure.title': 'AVISO DE EJECUCIÓN BETA V3',
        'disclosure.body': 'Este es VGT AETHEL Beta V3 — un sistema operativo soberano de inteligencia estratégica desarrollado por VisionGaiaTechnology.\n\nEl uso del software es totalmente bajo su propio riesgo. Aethel posee capacidades avanzadas en el sistema host (ejecución en terminal, escritura en disco, simulación de teclado y ratón). Comportamientos imprevistos pueden causar pérdida de datos o inestabilidad.',
        'disclosure.accept': 'ENTENDIDO Y ACEPTAR',
        'setup.title': 'VGT GENESIS', 'setup.subtitle': 'Configuración inicial del sistema',
        'setup.initialize': 'INICIALIZAR SISTEMA', 'setup.local': 'INICIAR IA LOCAL (OLLAMA)',
        'nav.assistant': 'Asistente', 'nav.core': 'Núcleo neuronal', 'nav.chat': 'Chat', 'nav.personas': 'Personas',
        'nav.agents': 'Equipo de agentes', 'nav.operator': 'Operador en vivo', 'nav.workspace': 'Espacio de trabajo',
        'nav.sphere': 'Sphere', 'nav.tasks': 'Automatizaciones y Tareas', 'nav.memory': 'Memoria Nexus', 'nav.personal': 'Núcleo personal',
        'nav.global': 'Vigilancia global', 'nav.globe': 'Globo y fuentes en vivo', 'nav.runs': 'Centro de ejecuciones',
        'nav.casesSection': 'Casos y seguridad', 'nav.cases': 'Casos y evidencias', 'nav.security': 'Seguridad y auditoría',
        'nav.archive': 'Archivo de chats', 'sidebar.model': 'Modelo de IA', 'sidebar.voice': 'Voz',
        'sidebar.persona': 'Persona', 'sidebar.newChat': '+ Nuevo chat', 'settings.title': 'Ajustes',
        'settings.subtitle': 'Acceso API, salud de proveedores y acciones del sistema — todo permanece en tu dispositivo.',
        'gw.title': 'VIGILANCIA GLOBAL', 'gw.briefing': 'Informe de situación', 'gw.refresh': 'Actualizar fuentes',
        'gw.settings': 'Ajustes', 'gw.feed': 'Noticias', 'gw.layers': 'CAPAS DEL MAPA',
        'gw.translate': 'TRADUCIR', 'gw.compose': 'CREAR ARTÍCULO', 'gw.discuss': 'HABLAR CON AETHEL',
    }),
    fr: Object.freeze({
        'language.label': 'Langue',
        'splash.start': 'DÉMARRER',
        'disclosure.title': 'AVERTISSEMENT D\'EXÉCUTION BETA V3',
        'disclosure.body': 'Ceci est VGT AETHEL Beta V3 — un système souverain d\'intelligence stratégique développé par VisionGaiaTechnology.\n\nL\'utilisation de ce logiciel se fait entièrement à vos propres risques. Aethel possède des autorisations système étendues (commandes terminales, écriture sur disque, simulation clavier/souris). Des comportements inattendus peuvent entraîner des pertes de données.',
        'disclosure.accept': 'COMPRIS ET ACCEPTER',
        'setup.title': 'VGT GENESIS', 'setup.subtitle': 'Configuration initiale du système',
        'setup.initialize': 'INITIALISER LE SYSTÈME', 'setup.local': 'DÉMARRER IA LOCALE (OLLAMA)',
        'nav.assistant': 'Assistant', 'nav.core': 'Noyau Neuronal', 'nav.chat': 'Chat', 'nav.personas': 'Personas',
        'nav.agents': 'Équipe d\'Agents', 'nav.operator': 'Opérateur en Direct', 'nav.workspace': 'Espace de Travail',
        'nav.sphere': 'Sphere', 'nav.tasks': 'Automatisations & Tâches', 'nav.memory': 'Mémoire Nexus', 'nav.personal': 'Noyau Personnel',
        'nav.global': 'Surveillance Globale', 'nav.globe': 'Globe & Flux en Direct', 'nav.runs': 'Centre d\'Exécution',
        'nav.casesSection': 'Dossiers & Sécurité', 'nav.cases': 'Dossiers & Preuves', 'nav.security': 'Sécurité & Audit',
        'nav.archive': 'Archives Chat', 'sidebar.model': 'Modèle IA', 'sidebar.voice': 'Voix',
        'sidebar.persona': 'Persona', 'sidebar.newChat': '+ Nouveau Chat', 'settings.title': 'Paramètres',
        'settings.subtitle': 'Clés API, état des fournisseurs et actions système — toutes vos données restent locales.',
        'gw.title': 'SURVEILLANCE GLOBALE', 'gw.briefing': 'Rapport de Situation', 'gw.refresh': 'Actualiser le Flux',
        'gw.settings': 'Paramètres', 'gw.feed': 'Fil d\'actualités', 'gw.layers': 'COUCHES DE CARTE',
        'gw.translate': 'TRADUIRE', 'gw.compose': 'CRÉER UN ARTICLE', 'gw.discuss': 'DISCUTER AVEC AETHEL',
    }),
    ru: Object.freeze({
        'language.label': 'Язык',
        'splash.start': 'НАЧАТЬ',
        'disclosure.title': 'УВЕДОМЛЕНИЕ О BETA V3',
        'disclosure.body': 'Это VGT AETHEL Beta V3 — суверенная система стратегической разведки, разработанная VisionGaiaTechnology.\n\nИспользование программного обеспечения осуществляется на ваш собственный риск. Aethel обладает широкими системными полномочиями (терминал, запись на диск, эмуляция ввода). Неожиданные действия модели могут привести к потере данных.',
        'disclosure.accept': 'ПОНЯТНО И ПОДТВЕРЖДАЮ',
        'setup.title': 'VGT GENESIS', 'setup.subtitle': 'Начальная конфигурация системы',
        'setup.initialize': 'ИНИЦИАЛИЗИРОВАТЬ СИСТЕМУ', 'setup.local': 'ЗАПУСТИТЬ ЛОКАЛЬНЫЙ ИИ (OLLAMA)',
        'nav.assistant': 'Ассистент', 'nav.core': 'Нейронное ядро', 'nav.chat': 'Чат', 'nav.personas': 'Персоны',
        'nav.agents': 'Команда агентов', 'nav.operator': 'Живой оператор', 'nav.workspace': 'Рабочая среда',
        'nav.sphere': 'Сфера', 'nav.tasks': 'Автоматизация и Задачи', 'nav.memory': 'Память Nexus', 'nav.personal': 'Личное ядро',
        'nav.global': 'Глобальный мониторинг', 'nav.globe': 'Глобус и ленты', 'nav.runs': 'Центр задач',
        'nav.casesSection': 'Дела и безопасность', 'nav.cases': 'Дела и доказательства', 'nav.security': 'Безопасность и аудит',
        'nav.archive': 'Архив чатов', 'sidebar.model': 'Модель ИИ', 'sidebar.voice': 'Голос',
        'sidebar.persona': 'Персона', 'sidebar.newChat': '+ Новый чат', 'settings.title': 'Настройки',
        'settings.subtitle': 'API, состояние провайдеров и системные действия — данные остаются на устройстве.',
        'gw.title': 'ГЛОБАЛЬНЫЙ МОНИТОРИНГ', 'gw.briefing': 'Ситуационная сводка', 'gw.refresh': 'Обновить ленту',
        'gw.settings': 'Настройки', 'gw.feed': 'Лента новостей', 'gw.layers': 'СЛОИ КАРТЫ',
        'gw.translate': 'ПЕРЕВЕСТИ', 'gw.compose': 'СОЗДАТЬ МАТЕРИАЛ', 'gw.discuss': 'ОБСУДИТЬ С AETHEL',
    }),
});

export function currentLanguage() {
    const stored = localStorage.getItem(STORAGE_KEY);
    return SUPPORTED.includes(stored) ? stored : 'de';
}

export function t(key, language = currentLanguage()) {
    return messages[language]?.[key] || messages.de[key] || key;
}

export function updateDisclosureModalText(language = currentLanguage()) {
    const titleEl = document.querySelector('#startup-warning-modal [data-i18n="disclosure.title"]');
    const bodyEl = document.getElementById('startup-disclosure-body');
    const btnEl = document.getElementById('startup-warning-btn-accept');

    if (titleEl) titleEl.textContent = t('disclosure.title', language);
    if (bodyEl) bodyEl.textContent = t('disclosure.body', language);
    if (btnEl) btnEl.textContent = t('disclosure.accept', language);
}

export function applyTranslations(root = document) {
    const language = currentLanguage();
    document.documentElement.lang = language;
    root.querySelectorAll('[data-i18n]').forEach(element => {
        element.textContent = t(element.dataset.i18n, language);
    });
    root.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
        element.setAttribute('placeholder', t(element.dataset.i18nPlaceholder, language));
    });
    root.querySelectorAll('[data-i18n-title]').forEach(element => {
        element.setAttribute('title', t(element.dataset.i18nTitle, language));
    });
    document.querySelectorAll('.aethel-language-select').forEach(select => { select.value = language; });
    updateDisclosureModalText(language);
}

function translateElement(element, language) {
    if (!(element instanceof Element)) return;
    if (element.dataset.i18n) element.textContent = t(element.dataset.i18n, language);
    if (element.dataset.i18nPlaceholder) element.setAttribute('placeholder', t(element.dataset.i18nPlaceholder, language));
    if (element.dataset.i18nTitle) element.setAttribute('title', t(element.dataset.i18nTitle, language));
    element.querySelectorAll('[data-i18n], [data-i18n-placeholder], [data-i18n-title]').forEach(child => {
        if (child.dataset.i18n) child.textContent = t(child.dataset.i18n, language);
        if (child.dataset.i18nPlaceholder) child.setAttribute('placeholder', t(child.dataset.i18nPlaceholder, language));
        if (child.dataset.i18nTitle) child.setAttribute('title', t(child.dataset.i18nTitle, language));
    });
}

export function setLanguage(language) {
    if (!SUPPORTED.includes(language)) return false;
    localStorage.setItem(STORAGE_KEY, language);
    applyTranslations();
    window.dispatchEvent(new CustomEvent('aethel:language-changed', { detail: { language } }));
    return true;
}

export function setupI18n() {
    document.querySelectorAll('.aethel-language-select').forEach(select => {
        if (select.dataset.i18nBound === 'true') return;
        select.dataset.i18nBound = 'true';
        select.addEventListener('change', () => setLanguage(select.value));
    });
    applyTranslations();
    if (!window.__aethelI18nObserver) {
        window.__aethelI18nObserver = new MutationObserver(records => {
            const language = currentLanguage();
            for (const record of records) {
                record.addedNodes.forEach(node => translateElement(node, language));
            }
        });
        window.__aethelI18nObserver.observe(document.body, { childList: true, subtree: true });
    }
}
