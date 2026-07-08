import { state } from './state.js';
import { refreshSecurityHUD, fetchActiveLeasesList } from './security.js';
import { speak } from './voice.js';

let viewportRefreshInterval = null;

export function startViewportRefresh() {
    const img = document.getElementById("browser-screenshot");
    const placeholder = document.getElementById("browser-placeholder");
    const addrBar = document.getElementById("browser-url-input");
    const loadStatus = document.getElementById("browser-load-status");

    if (!img) return;

    if (addrBar) addrBar.value = "desktop://primary_viewport (Sovereign Live Control)";

    // Run first fetch immediately
    refreshViewportImage();

    // Set up loop (800ms for snappier refresh rate)
    if (viewportRefreshInterval) clearInterval(viewportRefreshInterval);
    viewportRefreshInterval = setInterval(() => {
        // Only fetch if the control panel is actually visible on screen
        const controlView = document.getElementById("view-control");
        if (controlView && !controlView.classList.contains("hidden")) {
            refreshViewportImage();
        } else {
            stopViewportRefresh();
        }
    }, 800);
}

function refreshViewportImage() {
    const img = document.getElementById("browser-screenshot");
    const placeholder = document.getElementById("browser-placeholder");
    if (!img) return;

    img.src = `${state.API_BASE}/v1/viewport/screenshot?t=${Date.now()}`;
    img.onload = () => {
        img.classList.remove("hidden");
        if (placeholder) placeholder.classList.add("hidden");
    };
    img.onerror = (e) => {
        console.warn("Failed to load viewport screenshot", e);
    };
}

export function stopViewportRefresh() {
    if (viewportRefreshInterval) {
        clearInterval(viewportRefreshInterval);
        viewportRefreshInterval = null;
    }
}

export function setupControlUIEvents() {
    const btnPause = document.getElementById("control-btn-pause");
    const btnKill = document.getElementById("control-btn-kill");
    const btnRevokeAll = document.getElementById("control-btn-revoke");

    if (btnPause) {
        btnPause.addEventListener("click", () => {
            speak("Ausführung pausiert.");
            alert("Operator: Agenten-Ausführung pausiert.");
        });
    }

    if (btnKill) {
        btnKill.addEventListener("click", () => {
            speak("Ausführung abgebrochen.");
            alert("Operator: Agenten-Ausführung hart abgebrochen.");
        });
    }

    if (btnRevokeAll) {
        btnRevokeAll.addEventListener("click", async () => {
            try {
                const res = await fetch(`${state.API_BASE}/v1/security/leases`);
                const leases = await res.json();
                
                for (let l of leases) {
                    await fetch(`${state.API_BASE}/v1/security/leases?id=${l.lease_id}`, { method: 'DELETE' });
                }
                speak("Alle Freigaben widerrufen.");
                refreshSecurityHUD();
                fetchActiveLeasesList();
            } catch(e) {
                console.error("Failed to revoke all leases", e);
            }
        });
    }

    const controlBtnMic = document.getElementById("control-btn-mic");
    if (controlBtnMic) {
        controlBtnMic.addEventListener("click", () => {
            const mainBtnMic = document.getElementById("btn-mic");
            if (mainBtnMic) mainBtnMic.click();
        });
    }

    // Monitor control view visibility to start/stop screenshot stream automatically
    const navBtnControl = document.getElementById("nav-btn-control");
    if (navBtnControl) {
        navBtnControl.addEventListener("click", () => {
            startViewportRefresh();
        });
    }
}
