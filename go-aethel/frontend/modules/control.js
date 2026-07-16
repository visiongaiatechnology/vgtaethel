import { state } from './state.js';
import { refreshSecurityHUD, fetchActiveLeasesList } from './security.js';
import { speak } from './voice.js';

let viewportRefreshInterval = null;

export function startViewportRefresh() {
    const controlView = document.getElementById("view-control");
    const sphereView = document.getElementById("view-sphere");

    const addrBar = document.getElementById("browser-url-input");
    if (addrBar) addrBar.value = "desktop://primary_viewport (Sovereign Live Control)";

    const sphereAddrBar = document.getElementById("sphere-browser-url");
    if (sphereAddrBar) sphereAddrBar.value = "desktop://primary_viewport (Live Desktop Monitor)";

    // Run first fetch immediately
    const isSphereFirst = sphereView && !sphereView.classList.contains("hidden");
    refreshViewportImage(isSphereFirst);

    // Set up loop (800ms for snappier refresh rate)
    if (viewportRefreshInterval) clearInterval(viewportRefreshInterval);
    viewportRefreshInterval = setInterval(() => {
        const isControlVisible = controlView && !controlView.classList.contains("hidden");
        const isSphereVisible = sphereView && !sphereView.classList.contains("hidden");
        
        if (isControlVisible || isSphereVisible) {
            refreshViewportImage(isSphereVisible);
        } else {
            stopViewportRefresh();
        }
    }, 800);
}

let isFetchingViewport = false;

function refreshViewportImage(isSphere = false) {
    if (isFetchingViewport) return;

    const imgId = isSphere ? "sphere-browser-screenshot" : "browser-screenshot";
    const placeholderId = isSphere ? "sphere-browser-placeholder" : "browser-placeholder";

    const img = document.getElementById(imgId);
    const placeholder = document.getElementById(placeholderId);
    if (!img) return;

    isFetchingViewport = true;
    
    // Safety timeout in case load fails completely without error event
    const timeoutId = setTimeout(() => {
        isFetchingViewport = false;
    }, 5000);

    const screenshotUrl = isSphere ? `${state.API_BASE}/browser/screenshot.png` : `${state.API_BASE}/v1/viewport/screenshot`;
    img.src = `${screenshotUrl}?t=${Date.now()}`;

    img.onload = () => {
        clearTimeout(timeoutId);
        isFetchingViewport = false;
        img.classList.remove("hidden");
        if (placeholder) placeholder.classList.add("hidden");

        // If it's the sphere browser, also refresh the URL address bar from the status endpoint
        if (isSphere) {
            fetch(`${state.API_BASE}/v1/viewport/status`)
                .then(r => r.json())
                .then(data => {
                    const sphereAddrBar = document.getElementById("sphere-browser-url");
                    if (sphereAddrBar && data && data.browser_url) {
                        sphereAddrBar.value = data.browser_url;
                    }
                })
                .catch(e => console.warn("Failed to fetch browser status", e));
        }
    };
    img.onerror = (e) => {
        clearTimeout(timeoutId);
        isFetchingViewport = false;
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
        btnPause.addEventListener("click", async () => {
            const resume = btnPause.dataset.runState === 'paused';
            const action = resume ? 'resume' : 'pause';
            try {
                if (state.activeRunId) {
                    const response = await fetch(`${state.API_BASE}/v1/runs/${encodeURIComponent(state.activeRunId)}/${action}`, { method: 'POST' });
                    if (!response.ok) throw new Error(await response.text());
                }
                state.agentPaused = !resume;
                if (!resume) state.activeInferenceController?.abort();
                btnPause.dataset.runState = resume ? 'running' : 'paused';
                btnPause.textContent = resume ? 'PAUSE RUN' : 'RESUME RUN';
                speak(resume ? "Ausführung fortgesetzt." : "Ausführung pausiert.");
                window.showAethelToast?.(resume ? 'Persistenter Run fortgesetzt.' : 'Persistenter Run pausiert.', 'success');
            } catch (error) {
                window.showAethelToast?.(`Run-Steuerung fehlgeschlagen: ${error.message}`, 'error');
            }
        });
    }

    if (btnKill) {
        btnKill.addEventListener("click", async () => {
            try {
                if (state.activeRunId) {
                    const response = await fetch(`${state.API_BASE}/v1/runs/${encodeURIComponent(state.activeRunId)}/cancel`, { method: 'POST' });
                    if (!response.ok) throw new Error(await response.text());
                }
            } catch (error) {
                window.showAethelToast?.(`Run-Abbruch fehlgeschlagen: ${error.message}`, 'error');
                return;
            }
            state.agentPaused = true;
            state.activeInferenceController?.abort();
            state.pendingToolRequest = null;
            state.pendingToolCallId = "";
            state.pendingToolCallName = "";
            state.agenticTurnCount = 0;
            state.agenticStuckCount = 0;
            state.activeRunId = null;
            speak("Ausführung abgebrochen.");
            window.showAethelToast?.('Persistenter Run abgebrochen; lokale Freigabeanforderung verworfen.', 'success');
        });
    }

    if (btnRevokeAll) {
        btnRevokeAll.addEventListener("click", async () => {
            try {
                const res = await fetch(`${state.API_BASE}/v1/security/leases`);
                const leases = await res.json();
                
                for (let l of leases) {
                    await fetch(`${state.API_BASE}/v1/security/leases?id=${encodeURIComponent(l.lease_id)}`, { method: 'DELETE' });
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

    // Monitor control/sphere view visibility to start/stop screenshot stream automatically
    const navBtnControl = document.getElementById("nav-btn-control");
    if (navBtnControl) {
        navBtnControl.addEventListener("click", () => {
            startViewportRefresh();
        });
    }

    const navBtnSphere = document.getElementById("nav-btn-sphere");
    if (navBtnSphere) {
        navBtnSphere.addEventListener("click", () => {
            startViewportRefresh();
        });
    }
}
