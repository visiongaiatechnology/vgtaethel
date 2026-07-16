const VIEW_META = Object.freeze({
    chat: ['CONVERSATION LAYER', 'Chat Terminal', 'Dialog, Tools und laufende Arbeit in einem fokussierten Arbeitsraum.'],
    agent: ['ORCHESTRATION LAYER', 'Agent Tracker', 'Planung, Ausführung, Validierung und Wiederaufnahme transparent steuern.'],
    security: ['TRUST CONTROL', 'Security & Audit', 'Berechtigungen, Risiken und kryptografische Nachweise zentral prüfen.'],
    memory: ['KNOWLEDGE LAYER', 'Nexus Memory', 'Persönliches Wissen, Projektkontext und geschützte Secrets verwalten.'],
    personal: ['IDENTITY LAYER', 'Personal Core', 'Aethels Persönlichkeit, Routinen und proaktive Zusammenarbeit konfigurieren.'],
    tasks: ['EXECUTION LAYER', 'Run Center', 'Persistente Runs starten, freigeben, pausieren und nachvollziehen.'],
    personas: ['BEHAVIOR LAYER', 'Persona Registry', 'Spezialisierte Aethel-Rollen mit klaren Verhaltensgrenzen erstellen.'],
    archive: ['HISTORY LAYER', 'Chat Archive', 'Vergangene Gespräche finden, prüfen und sicher fortsetzen.'],
});

function buildViewHeader(view, [kickerText, titleText, descriptionText]) {
    if (!view || view.querySelector(':scope > .vgt-view-header')) return;
    const header = document.createElement('header');
    header.className = 'vgt-view-header';

    const copy = document.createElement('div');
    copy.className = 'vgt-view-header-copy';
    const kicker = document.createElement('span');
    kicker.className = 'vgt-view-kicker';
    kicker.textContent = kickerText;
    const title = document.createElement('h2');
    title.className = 'vgt-view-title';
    title.textContent = titleText;
    const description = document.createElement('p');
    description.className = 'vgt-view-description';
    description.textContent = descriptionText;
    copy.append(kicker, title, description);

    const state = document.createElement('div');
    state.className = 'vgt-view-state';
    const dot = document.createElement('span');
    dot.className = 'vgt-view-state-dot';
    const label = document.createElement('span');
    label.textContent = 'LOCAL CORE';
    state.append(dot, label);
    header.append(copy, state);
    view.prepend(header);
    view.setAttribute('aria-labelledby', `${view.id}-title`);
    title.id = `${view.id}-title`;
}

function setupResponsiveNavigation() {
    const header = document.querySelector('.header-bar');
    const sidebar = document.querySelector('.sidebar');
    if (!header || !sidebar || document.getElementById('vgt-mobile-nav-toggle')) return;

    const toggle = document.createElement('button');
    toggle.id = 'vgt-mobile-nav-toggle';
    toggle.className = 'vgt-mobile-nav-toggle';
    toggle.type = 'button';
    toggle.setAttribute('aria-label', 'Navigation öffnen');
    toggle.setAttribute('aria-expanded', 'false');
    toggle.textContent = 'MENU';

    const backdrop = document.createElement('button');
    backdrop.type = 'button';
    backdrop.className = 'vgt-sidebar-backdrop';
    backdrop.setAttribute('aria-label', 'Navigation schließen');

    const close = () => {
        document.body.classList.remove('vgt-sidebar-open');
        toggle.setAttribute('aria-expanded', 'false');
        toggle.setAttribute('aria-label', 'Navigation öffnen');
    };
    const open = () => {
        document.body.classList.add('vgt-sidebar-open');
        toggle.setAttribute('aria-expanded', 'true');
        toggle.setAttribute('aria-label', 'Navigation schließen');
    };
    toggle.addEventListener('click', () => document.body.classList.contains('vgt-sidebar-open') ? close() : open());
    backdrop.addEventListener('click', close);
    sidebar.addEventListener('click', event => {
        if (event.target.closest('.nav-menu-btn') && window.matchMedia('(max-width: 760px)').matches) close();
    });
    window.addEventListener('keydown', event => { if (event.key === 'Escape') close(); });
    window.addEventListener('resize', () => { if (!window.matchMedia('(max-width: 760px)').matches) close(); });

    header.prepend(toggle);
    document.body.appendChild(backdrop);
}

export function setupUIModernization() {
    Object.entries(VIEW_META).forEach(([key, meta]) => buildViewHeader(document.getElementById(`view-${key}`), meta));
    setupResponsiveNavigation();
}
