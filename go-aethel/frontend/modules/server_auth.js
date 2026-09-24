// STATUS: DIAMANT VGT SUPREME
// VGT AETHEL // SERVER MODE AUTHENTICATION & CSRF GATE

let currentCSRFToken = '';
let fetchInterceptorInstalled = false;
let activeAuthResolve = null;

function isSameOriginAPI(input) {
    try {
        const raw = typeof input === 'string' ? input : (input?.url || '');
        if (!raw) return false;
        if (raw.startsWith('/v1/') || raw.startsWith('/browser/') || raw === '/health') {
            return true;
        }
        const parsed = new URL(raw, window.location.origin);
        return parsed.origin === window.location.origin &&
            (parsed.pathname.startsWith('/v1/') || parsed.pathname.startsWith('/browser/'));
    } catch (_) {
        return false;
    }
}

function isMutatingMethod(method) {
    const m = String(method || 'GET').toUpperCase();
    return m === 'POST' || m === 'PUT' || m === 'PATCH' || m === 'DELETE';
}

function installFetchCSRFInterceptor() {
    if (fetchInterceptorInstalled) return;
    fetchInterceptorInstalled = true;

    const nativeFetch = window.fetch.bind(window);
    window.fetch = async function aethelGuardedFetch(input, init = {}) {
        if (isSameOriginAPI(input)) {
            const nextInit = { ...init };
            if (!nextInit.credentials) {
                nextInit.credentials = 'same-origin';
            }
            const method = nextInit.method || (typeof input === 'object' && input?.method) || 'GET';
            if (isMutatingMethod(method) && currentCSRFToken) {
                const headers = new Headers(nextInit.headers || (typeof input === 'object' ? input.headers : undefined) || {});
                if (!headers.has('X-Aethel-CSRF')) {
                    headers.set('X-Aethel-CSRF', currentCSRFToken);
                }
                nextInit.headers = headers;
            }
            const response = await nativeFetch(input, nextInit);
            const urlStr = typeof input === 'string' ? input : (input?.url || '');
            if (response.status === 401 && !urlStr.includes('/v1/auth/')) {
                showAuthGateOverlay(true);
            }
            return response;
        }
        return nativeFetch(input, init);
    };
}

function applyServerChromeMode(isServerMode, isAuthenticated) {
    const winControls = document.querySelector('.aethel-window-controls');
    const winDivider = document.querySelector('.win-controls-divider');
    const logoutBtn = document.getElementById('btn-server-logout');
    const isDesktopWebView = Boolean(window.runtime || window.go?.main?.App);

    if (isServerMode || !isDesktopWebView) {
        document.body.classList.add('aethel-server-mode');
        if (winControls) winControls.classList.add('hidden');
        if (winDivider) winDivider.classList.add('hidden');
    }

    if (logoutBtn) {
        logoutBtn.classList.toggle('hidden', !(isServerMode && isAuthenticated));
    }
}

function configureGateFormMode(configured) {
    const badgeEl = document.getElementById('auth-gate-mode-badge');
    const titleEl = document.getElementById('auth-gate-title');
    const subtitleEl = document.getElementById('auth-gate-subtitle');
    const confirmGroup = document.getElementById('auth-gate-confirm-group');
    const submitLabel = document.getElementById('auth-gate-submit-label');
    const errorBox = document.getElementById('auth-gate-error');

    if (errorBox) errorBox.classList.add('hidden');

    if (configured) {
        if (badgeEl) badgeEl.textContent = 'SERVER ACCESS // ARGON2ID + AES-256-GCM';
        if (titleEl) titleEl.textContent = 'OPERATOR AUTHENTIFIZIERUNG';
        if (subtitleEl) subtitleEl.textContent = 'Dieser Aethel-Server ist geschützt. Bitte gib dein Operator-Passwort ein, um die Sitzung freizuschalten.';
        if (confirmGroup) confirmGroup.classList.add('hidden');
        if (submitLabel) submitLabel.textContent = 'SITZUNGENTSperren // LOGIN'.replace('SITZUNGENTSperren', 'SITZUNG ENTSPERREN');
    } else {
        if (badgeEl) badgeEl.textContent = 'INITIAL SERVER SETUP // ARGON2ID SEAL';
        if (titleEl) titleEl.textContent = 'SERVER-PASSWORT FESTLEGEN';
        if (subtitleEl) subtitleEl.textContent = 'Erster Server-Start erkannt. Lege jetzt dein Operator-Passwort fest (mindestens 10 Zeichen).';
        if (confirmGroup) confirmGroup.classList.remove('hidden');
        if (submitLabel) submitLabel.textContent = 'PASSWORT VERSIEGELN & STARTEN';
    }
}

function showAuthError(message) {
    const errorBox = document.getElementById('auth-gate-error');
    const errorText = document.getElementById('auth-gate-error-text');
    if (errorText) errorText.textContent = message || 'Authentifizierung fehlgeschlagen.';
    if (errorBox) errorBox.classList.remove('hidden');
}

function showAuthGateOverlay(configured = true) {
    const gate = document.getElementById('aethel-auth-gate');
    if (!gate) return;
    gate.dataset.configured = configured ? 'true' : 'false';
    configureGateFormMode(configured);
    gate.classList.remove('hidden');
    const pwdInput = document.getElementById('auth-gate-password');
    if (pwdInput) {
        pwdInput.value = '';
        setTimeout(() => {
            try { pwdInput.focus(); } catch (_) {}
        }, 60);
    }
    const confirmInput = document.getElementById('auth-gate-password-confirm');
    if (confirmInput) confirmInput.value = '';
}

function hideAuthGateOverlay() {
    const gate = document.getElementById('aethel-auth-gate');
    if (!gate) return;
    gate.classList.add('hidden');
}

async function submitAuthForm(event) {
    if (event) event.preventDefault();
    const gate = document.getElementById('aethel-auth-gate');
    const pwdInput = document.getElementById('auth-gate-password');
    const confirmInput = document.getElementById('auth-gate-password-confirm');
    const submitBtn = document.getElementById('auth-gate-submit');

    const configured = gate?.dataset.configured !== 'false';
    const password = pwdInput ? pwdInput.value : '';
    const confirmPassword = confirmInput ? confirmInput.value : '';

    if (!password || password.length < 1) {
        showAuthError('Bitte gib dein Operator-Passwort ein.');
        return;
    }

    if (!configured) {
        if (password.length < 10) {
            showAuthError('Das Server-Passwort muss mindestens 10 Zeichen lang sein.');
            return;
        }
        if (password !== confirmPassword) {
            showAuthError('Die eingegebenen Passwörter stimmen nicht überein.');
            return;
        }
    }

    if (submitBtn) submitBtn.disabled = true;
    try {
        const endpoint = configured ? '/v1/auth/login' : '/v1/auth/setup';
        const resp = await window.fetch(endpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
            body: JSON.stringify({ password }),
            credentials: 'same-origin',
        });
        const payload = await resp.json().catch(() => ({}));
        if (!resp.ok) {
            if (resp.status === 428 || payload.configured === false) {
                if (gate) gate.dataset.configured = 'false';
                configureGateFormMode(false);
            }
            showAuthError(payload.error || `Anmeldung fehlgeschlagen (${resp.status}).`);
            return;
        }

        currentCSRFToken = payload.csrf_token || '';
        if (pwdInput) pwdInput.value = '';
        if (confirmInput) confirmInput.value = '';
        hideAuthGateOverlay();
        applyServerChromeMode(true, true);

        if (typeof activeAuthResolve === 'function') {
            const resolve = activeAuthResolve;
            activeAuthResolve = null;
            resolve(true);
        }
    } catch (_) {
        showAuthError('Server nicht erreichbar. Bitte Verbindung prüfen.');
    } finally {
        if (submitBtn) submitBtn.disabled = false;
    }
}

async function handleServerLogout() {
    try {
        await window.fetch('/v1/auth/logout', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'same-origin',
        });
    } catch (_) {
        // Ignore network error on logout
    }
    currentCSRFToken = '';
    applyServerChromeMode(true, false);
    showAuthGateOverlay(true);
}

function bindAuthGateEvents() {
    const form = document.getElementById('aethel-auth-form');
    if (form && !form.dataset.bound) {
        form.dataset.bound = 'true';
        form.addEventListener('submit', submitAuthForm);
    }
    const logoutBtn = document.getElementById('btn-server-logout');
    if (logoutBtn && !logoutBtn.dataset.bound) {
        logoutBtn.dataset.bound = 'true';
        logoutBtn.addEventListener('click', handleServerLogout);
    }
}

export async function ensureServerAuthGate() {
    installFetchCSRFInterceptor();
    bindAuthGateEvents();

    try {
        const resp = await window.fetch('/v1/auth/status', {
            method: 'GET',
            headers: { Accept: 'application/json' },
            credentials: 'same-origin',
        });
        if (!resp.ok) {
            return;
        }
        const status = await resp.json();
        const isServer = status.mode === 'server' || Boolean(status.auth_required);

        if (!status.auth_required) {
            applyServerChromeMode(isServer, false);
            hideAuthGateOverlay();
            return;
        }

        if (status.authenticated) {
            currentCSRFToken = status.csrf_token || '';
            applyServerChromeMode(true, true);
            hideAuthGateOverlay();
            return;
        }

        applyServerChromeMode(true, false);
        showAuthGateOverlay(Boolean(status.configured));

        await new Promise((resolve) => {
            activeAuthResolve = resolve;
        });
    } catch (_) {
        // In local desktop startup fallback, proceed normally
    }
}
