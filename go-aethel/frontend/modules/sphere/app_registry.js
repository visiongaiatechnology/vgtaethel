// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/app_registry.js
// Purpose: Declarative Application Registry, Manifests, Accent Colors, and Capabilities

export const sphereAppDefinitions = {
    dashboard: {
        id: 'dashboard',
        title: 'PERSONAL DASHBOARD',
        category: 'operations',
        accent: 'cyan',
        icon: '◫',
        left: 48,
        top: 40,
        width: 720,
        height: 520,
        desktop: 'PERSONAL',
        capabilities: ['personal.read', 'weather.read', 'market.read']
    },
    writer: {
        id: 'writer',
        title: 'AETHEL WRITER // CANVAS',
        category: 'productivity',
        accent: 'cyan',
        icon: '✦',
        left: 64,
        top: 50,
        width: 680,
        height: 540,
        desktop: 'WORK',
        capabilities: ['document.read', 'document.write']
    },
    browser: {
        id: 'browser',
        title: 'AETHEL BROWSER // MULTI-TAB',
        category: 'intelligence',
        accent: 'purple',
        icon: '◌',
        left: 420,
        top: 60,
        width: 660,
        height: 480,
        desktop: 'RESEARCH',
        capabilities: ['network.public.read', 'browser.navigate', 'browser.interact']
    },
    travel: {
        id: 'travel',
        title: 'TRIP PLANNER // GLOBAL WATCH',
        category: 'operations',
        accent: 'orange',
        icon: '🗺',
        left: 100,
        top: 60,
        width: 760,
        height: 550,
        desktop: 'TRAVEL',
        capabilities: ['globalwatch.read', 'weather.read', 'planner.write']
    },
    planner: {
        id: 'planner',
        title: 'MASTER PLANNER // GOALS & RUNS',
        category: 'productivity',
        accent: 'cyan',
        icon: '⏰',
        left: 140,
        top: 70,
        width: 720,
        height: 520,
        desktop: 'WORK',
        capabilities: ['tasks.read', 'tasks.write', 'runs.read']
    },
    research: {
        id: 'research',
        title: 'RESEARCH DESK // EVIDENCE',
        category: 'intelligence',
        accent: 'purple',
        icon: '🔬',
        left: 200,
        top: 80,
        width: 700,
        height: 500,
        desktop: 'RESEARCH',
        capabilities: ['intelligence.read', 'evidence.write']
    },
    files: {
        id: 'files',
        title: 'VAULT EXPLORER // WORKSPACE',
        category: 'system',
        accent: 'cyan',
        icon: '▤',
        left: 96,
        top: 100,
        width: 540,
        height: 420,
        desktop: 'WORK',
        capabilities: ['filesystem.read']
    },
    mail: {
        id: 'mail',
        title: 'SPHERE MAIL // COMMAND',
        category: 'communication',
        accent: 'orange',
        icon: '✉',
        left: 260,
        top: 90,
        width: 680,
        height: 480,
        desktop: 'PERSONAL',
        capabilities: ['mail.read']
    },
    weather: {
        id: 'weather',
        title: 'WEATHER PULSE // SATELLITE',
        category: 'intelligence',
        accent: 'cyan',
        icon: '☼',
        left: 650,
        top: 74,
        width: 380,
        height: 340,
        desktop: 'PERSONAL',
        capabilities: ['weather.read']
    },
    market: {
        id: 'market',
        title: 'LIVE MARKET ENGINE // MULTI-ASSET',
        category: 'intelligence',
        accent: 'orange',
        icon: '◇',
        left: 580,
        top: 80,
        width: 600,
        height: 460,
        desktop: 'PERSONAL',
        capabilities: ['market.read']
    },
    runs: {
        id: 'runs',
        title: 'RUN DESK // AGENT ENGINE',
        category: 'system',
        accent: 'purple',
        icon: '◈',
        left: 540,
        top: 138,
        width: 440,
        height: 360,
        desktop: 'PROJECT',
        capabilities: ['runs.read']
    },
    live: {
        id: 'live',
        title: 'AETHEL LIVE FLOW',
        category: 'system',
        accent: 'purple',
        icon: '⌁',
        left: 470,
        top: 160,
        width: 480,
        height: 420,
        desktop: 'PROJECT',
        capabilities: ['runs.read']
    },
    terminal: {
        id: 'terminal',
        title: 'AETHEL TERMINAL // SHELL',
        category: 'system',
        accent: 'orange',
        icon: '$',
        left: 300,
        top: 120,
        width: 560,
        height: 380,
        desktop: 'PROJECT',
        capabilities: ['command.exec']
    },
    aethel: {
        id: 'aethel',
        title: 'AETHEL CONSOLE // LIVE AGENT',
        category: 'core',
        accent: 'cyan',
        icon: '◈',
        left: 180,
        top: 74,
        width: 540,
        height: 410,
        desktop: 'PERSONAL',
        capabilities: ['chat.write']
    },
    notes: {
        id: 'notes',
        title: 'SCRATCH NOTES',
        category: 'productivity',
        accent: 'orange',
        icon: '✎',
        left: 220,
        top: 100,
        width: 400,
        height: 340,
        desktop: 'PERSONAL',
        capabilities: ['local.storage']
    }
};

export function getAppDefinition(appID) {
    return sphereAppDefinitions[appID] || null;
}
