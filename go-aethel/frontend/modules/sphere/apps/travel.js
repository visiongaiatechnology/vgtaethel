// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/travel.js
// Purpose: Trip Planner & Global Watch Intelligence Bridge (Itineraries, Budget, Hotels, Flights & Live Risk Signals)

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';

let tripsList = [];
let activeTrip = null;

/**
 * Render Trip Planner & Global Watch Bridge inside a window body
 * @param {HTMLElement} body
 */
export async function renderTravelApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-travel-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:10px; color:#fff;';

    // 1. Topbar with Trip Selector, New Trip Button & Sync
    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(255,123,0,0.2); padding-bottom:8px;';

    const leftBox = document.createElement('div');
    leftBox.style.cssText = 'display:flex; align-items:center; gap:8px;';

    const tripSelect = document.createElement('select');
    tripSelect.id = 'sphere-travel-trip-select';
    tripSelect.style.cssText = 'background:rgba(0,0,0,0.4); border:1px solid rgba(255,123,0,0.3); color:#fff; padding:4px 8px; border-radius:4px; font-size:11px; font-family:var(--font-mono); outline:none;';

    leftBox.append(tripSelect);

    const rightActions = document.createElement('div');
    rightActions.style.cssText = 'display:flex; gap:6px;';

    const btnNewTrip = document.createElement('button');
    btnNewTrip.type = 'button';
    btnNewTrip.className = 'cyber-button';
    btnNewTrip.style.cssText = 'font-size:10px; padding:4px 10px; background:rgba(255,123,0,0.15); border:1px solid var(--vgt-orange); color:var(--vgt-orange); cursor:pointer; width:auto;';
    btnNewTrip.textContent = '+ NEUE REISE';
    btnNewTrip.addEventListener('click', () => openNewTripModal());

    const btnSendTo = document.createElement('button');
    btnSendTo.type = 'button';
    btnSendTo.className = 'cyber-button';
    btnSendTo.style.cssText = 'font-size:10px; padding:4px 10px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnSendTo.textContent = '➔ SEND TO...';
    btnSendTo.addEventListener('click', () => {
        if (activeTrip) {
            contextBus.openSendToDialog({
                sourceApp: 'travel',
                title: activeTrip.title,
                content: `Reise: ${activeTrip.destination} (${activeTrip.start_date} – ${activeTrip.end_date})\nBudget: €${activeTrip.budget_total_eur}`,
                objectType: 'trip'
            });
        }
    });

    rightActions.append(btnNewTrip, btnSendTo);
    header.append(leftBox, rightActions);

    // 2. Main Scrollable Content Area
    const content = document.createElement('div');
    content.className = 'sphere-travel-content';
    content.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:12px; padding-right:4px;';

    container.append(header, content);
    body.appendChild(container);

    // Fetch and render trips
    async function loadTrips() {
        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/trips`);
            if (res.ok) {
                const data = await res.json();
                tripsList = data.trips || [];
                if (tripsList.length === 0) {
                    // Create default Japan 2027 trip if none exists
                    await createDefaultSampleTrip();
                    return;
                }
                activeTrip = tripsList[0];
                updateTripSelect();
                renderTripDetails();
            }
        } catch (err) {
            console.error('Failed to load trips', err);
        }
    }

    function updateTripSelect() {
        tripSelect.replaceChildren();
        tripsList.forEach(t => {
            const opt = document.createElement('option');
            opt.value = t.id;
            opt.textContent = `${t.title} (${t.destination})`;
            if (activeTrip && t.id === activeTrip.id) opt.selected = true;
            tripSelect.appendChild(opt);
        });
    }

    tripSelect.addEventListener('change', () => {
        activeTrip = tripsList.find(t => t.id === tripSelect.value);
        renderTripDetails();
    });

    function renderTripDetails() {
        content.replaceChildren();
        if (!activeTrip) {
            const empty = document.createElement('div');
            empty.style.cssText = 'text-align:center; padding:40px 10px; color:var(--vgt-text-dim);';
            empty.textContent = 'Keine aktive Reise ausgewählt. Klicke auf "+ NEUE REISE".';
            content.appendChild(empty);
            return;
        }

        // 1. Overview Card (Destination, Dates, Budget Progress)
        const overviewCard = document.createElement('article');
        overviewCard.className = 'glass-card';
        overviewCard.style.cssText = 'background:linear-gradient(135deg, rgba(255,123,0,0.08), rgba(0,0,0,0.4)); border:1px solid rgba(255,123,0,0.3); border-radius:10px; padding:14px; display:flex; flex-direction:column; gap:8px;';

        const topRow = document.createElement('div');
        topRow.style.cssText = 'display:flex; justify-content:space-between; align-items:flex-start;';

        const titleBox = document.createElement('div');
        const mainTitle = document.createElement('h3');
        mainTitle.style.cssText = 'margin:0; font-size:15px; color:#fff; font-weight:bold;';
        mainTitle.textContent = activeTrip.title;

        const subDest = document.createElement('small');
        subDest.style.cssText = 'color:var(--vgt-orange); font-size:11px;';
        subDest.textContent = `📍 ${activeTrip.destination} · ${activeTrip.duration_days || 12} Tage · ${activeTrip.start_date || '2027-10-10'} bis ${activeTrip.end_date || '2027-10-22'}`;
        titleBox.append(mainTitle, subDest);

        const statusBadge = document.createElement('span');
        statusBadge.style.cssText = 'font-size:9px; background:rgba(255,123,0,0.2); border:1px solid var(--vgt-orange); color:var(--vgt-orange); padding:2px 8px; border-radius:4px; text-transform:uppercase; font-weight:bold;';
        statusBadge.textContent = activeTrip.status || 'PLANNING';

        topRow.append(titleBox, statusBadge);

        // Budget Bar
        const budgetBox = document.createElement('div');
        budgetBox.style.cssText = 'display:flex; flex-direction:column; gap:4px; margin-top:6px;';

        const budgetLabels = document.createElement('div');
        budgetLabels.style.cssText = 'display:flex; justify-content:space-between; font-size:10px; color:rgba(255,255,255,0.8);';
        budgetLabels.innerHTML = `<span>Budget: <b>€${(activeTrip.budget_spent_eur || 0).toLocaleString('de-DE')}</b> von €${(activeTrip.budget_total_eur || 2500).toLocaleString('de-DE')}</span><span>Verbleibend: €${((activeTrip.budget_total_eur || 2500) - (activeTrip.budget_spent_eur || 0)).toLocaleString('de-DE')}</span>`;

        const budgetBar = document.createElement('div');
        budgetBar.style.cssText = 'height:6px; background:rgba(255,255,255,0.1); border-radius:3px; overflow:hidden;';
        
        const spentPct = Math.min(100, Math.round(((activeTrip.budget_spent_eur || 0) / (activeTrip.budget_total_eur || 2500)) * 100));
        const budgetFill = document.createElement('div');
        budgetFill.style.cssText = `height:100%; width:${spentPct}%; background:linear-gradient(90deg, var(--vgt-cyan), var(--vgt-orange));`;
        budgetBar.appendChild(budgetFill);

        budgetBox.append(budgetLabels, budgetBar);
        overviewCard.append(topRow, budgetBox);
        content.appendChild(overviewCard);

        // 2. Global Watch Risk Impact Cards (Intelligence Bridge)
        const risks = activeTrip.global_watch_risks || [];
        if (risks.length > 0) {
            const riskSection = document.createElement('section');
            riskSection.style.cssText = 'display:flex; flex-direction:column; gap:8px;';

            const riskHeader = document.createElement('div');
            riskHeader.style.cssText = 'display:flex; align-items:center; gap:6px; font-size:11px; color:var(--vgt-orange); font-weight:bold;';
            riskHeader.innerHTML = `<span>⚠️ GLOBAL WATCH // REISE-LAGEWARNUNG</span>`;
            riskSection.appendChild(riskHeader);

            risks.forEach(r => {
                const card = document.createElement('article');
                card.className = 'glass-card';
                card.style.cssText = 'background:linear-gradient(135deg, rgba(255,0,79,0.12), rgba(0,0,0,0.5)); border:1px solid rgba(255,0,79,0.4); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:6px;';

                const cardTop = document.createElement('div');
                cardTop.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

                const rTitle = document.createElement('strong');
                rTitle.style.cssText = 'font-size:11px; color:#fff;';
                rTitle.textContent = `${r.region}: ${r.signal}`;

                const conf = document.createElement('span');
                conf.style.cssText = 'font-size:8px; background:rgba(255,0,79,0.2); color:var(--vgt-red); padding:1px 4px; border-radius:3px; border:1px solid var(--vgt-red);';
                conf.textContent = `CONFIDENCE: ${r.confidence || 'HIGH'}`;

                cardTop.append(rTitle, conf);

                const desc = document.createElement('p');
                desc.style.cssText = 'margin:0; font-size:10px; color:rgba(255,255,255,0.8); line-height:1.4;';
                desc.textContent = `Mögliche Auswirkungen: ${r.potential_impact || 'Flugverspätung / Störung'} (Betroffener Zeitraum: ${r.affected_dates || 'Anreise'})`;

                const btnRow = document.createElement('div');
                btnRow.style.cssText = 'display:flex; gap:6px; margin-top:4px;';

                const btnGW = document.createElement('button');
                btnGW.type = 'button';
                btnGW.className = 'cyber-button';
                btnGW.style.cssText = 'font-size:9px; padding:2px 8px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
                btnGW.textContent = 'IN GLOBAL WATCH ÖFFNEN';
                btnGW.addEventListener('click', () => {
                    const gwNav = document.getElementById('nav-btn-osint');
                    gwNav?.click();
                });

                const btnIgnore = document.createElement('button');
                btnIgnore.type = 'button';
                btnIgnore.className = 'cyber-button';
                btnIgnore.style.cssText = 'font-size:9px; padding:2px 8px; background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.2); color:#fff; cursor:pointer; width:auto;';
                btnIgnore.textContent = 'IGNORIEREN';
                btnIgnore.addEventListener('click', () => {
                    card.remove();
                });

                btnRow.append(btnGW, btnIgnore);
                card.append(cardTop, desc, btnRow);
                riskSection.appendChild(card);
            });
            content.appendChild(riskSection);
        }

        // 3. Lodging & Itinerary Candidates Grid
        const grid = document.createElement('div');
        grid.style.cssText = 'display:grid; grid-template-columns:1fr 1fr; gap:10px;';

        // Lodgings column
        const hotelCol = document.createElement('div');
        hotelCol.style.cssText = 'background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.08); border-radius:8px; padding:10px; display:flex; flex-direction:column; gap:6px;';
        hotelCol.innerHTML = `<strong style="font-size:11px; color:var(--vgt-cyan);">🏨 HOTEL-KANDIDATEN</strong>`;
        
        const sampleHotels = activeTrip.lodging_options || [
            { name: 'Hotel Niwa Tokyo', loc: 'Chiyoda, Tokio', price: '€165/Nacht', rating: '9.1' },
            { name: 'Richmond Hotel Premier', loc: 'Asakusa, Tokio', price: '€140/Nacht', rating: '8.8' }
        ];

        sampleHotels.forEach(h => {
            const hCard = document.createElement('div');
            hCard.style.cssText = 'background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.05); border-radius:6px; padding:8px; font-size:10px;';
            hCard.innerHTML = `<div style="display:flex; justify-content:space-between;"><b>${h.name}</b><span style="color:var(--vgt-green);">${h.price || '€150'}</span></div><small style="color:var(--vgt-text-dim);">${h.loc || h.location} · Bewertung ${h.rating || '9.0'}</small>`;
            hotelCol.appendChild(hCard);
        });

        // Flights column
        const flightCol = document.createElement('div');
        flightCol.style.cssText = 'background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.08); border-radius:8px; padding:10px; display:flex; flex-direction:column; gap:6px;';
        flightCol.innerHTML = `<strong style="font-size:11px; color:var(--vgt-purple);">✈ FLÜGE & ANREISE</strong>`;

        const sampleFlights = activeTrip.flight_options || [
            { airline: 'ANA All Nippon Airways', dep: 'FRA 12:10', arr: 'HND 07:55+1', price: '€890', direct: true },
            { airline: 'Lufthansa', dep: 'MUC 13:40', arr: 'HND 09:10+1', price: '€920', direct: true }
        ];

        sampleFlights.forEach(f => {
            const fCard = document.createElement('div');
            fCard.style.cssText = 'background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.05); border-radius:6px; padding:8px; font-size:10px;';
            fCard.innerHTML = `<div style="display:flex; justify-content:space-between;"><b>${f.airline}</b><span style="color:var(--vgt-cyan);">${f.price || '€890'}</span></div><small style="color:var(--vgt-text-dim);">${f.dep || f.departure} ➔ ${f.arr || f.arrival} (Direktflug)</small>`;
            flightCol.appendChild(fCard);
        });

        grid.append(hotelCol, flightCol);
        content.appendChild(grid);
    }

    async function createDefaultSampleTrip() {
        const defaultTrip = {
            title: 'Reise Japan 2027',
            destination: 'Tokyo, Japan',
            destinations: ['Kyoto', 'Osaka'],
            start_date: '2027-10-10',
            end_date: '2027-10-22',
            budget_total_eur: 2500,
            budget_spent_eur: 890,
            status: 'planning',
            global_watch_risks: [
                {
                    alert_id: 'ALERT-TYPHOON-1',
                    region: 'Tokyo',
                    signal: 'Schwere Unwetterwarnung / Wetterfront',
                    confidence: 'HIGH',
                    potential_impact: 'Mögliche Verspätung bei Ankunft',
                    affected_dates: '14. Oktober',
                    severity: 'WATCH'
                }
            ]
        };

        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/trips`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(defaultTrip)
            });
            if (res.ok) {
                const created = await res.json();
                tripsList = [created];
                activeTrip = created;
                updateTripSelect();
                renderTripDetails();
            }
        } catch (err) {
            console.error('Failed to create sample trip', err);
        }
    }

    function openNewTripModal() {
        let dialog = document.getElementById('sphere-new-trip-dialog');
        if (dialog) dialog.remove();

        dialog = document.createElement('div');
        dialog.id = 'sphere-new-trip-dialog';
        dialog.style.cssText = 'position:fixed; inset:0; z-index:10080; background:rgba(3,6,15,0.85); backdrop-filter:blur(10px); display:flex; justify-content:center; align-items:center; font-family:var(--font-mono);';

        const modal = document.createElement('div');
        modal.className = 'glass-card';
        modal.style.cssText = 'width:90%; max-width:480px; background:linear-gradient(145deg, rgba(16,8,28,0.98), rgba(4,6,14,0.99)); border:1px solid var(--vgt-orange); border-radius:12px; padding:20px; box-shadow:0 0 40px rgba(255,123,0,0.25); display:flex; flex-direction:column; gap:12px; color:#fff;';

        modal.innerHTML = `
            <div style="display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(255,123,0,0.2); padding-bottom:10px;">
                <strong style="color:var(--vgt-orange); font-size:13px;">🗺 NEUE REISE PLANEN</strong>
                <button id="trip-modal-close" style="background:none; border:none; color:var(--vgt-text-dim); font-size:16px; cursor:pointer;">✕</button>
            </div>
            <div style="display:flex; flex-direction:column; gap:10px; font-size:11px;">
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">REISETITEL</label>
                    <input id="trip-input-title" type="text" placeholder="z.B. Städtereise Paris 2027" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                </div>
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">HAUPTREISEZIEL</label>
                    <input id="trip-input-dest" type="text" placeholder="z.B. Paris, Frankreich" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                </div>
                <div style="display:grid; grid-template-columns:1fr 1fr; gap:8px;">
                    <div>
                        <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">STARTDATUM</label>
                        <input id="trip-input-start" type="date" value="2027-05-01" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:5px 8px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                    </div>
                    <div>
                        <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">ENDDATUM</label>
                        <input id="trip-input-end" type="date" value="2027-05-07" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:5px 8px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                    </div>
                </div>
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">GESAMTBUDGET (€)</label>
                    <input id="trip-input-budget" type="number" value="1800" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                </div>
            </div>
            <div style="display:flex; justify-content:flex-end; gap:8px; margin-top:8px;">
                <button id="trip-modal-cancel" class="cyber-button" style="font-size:10px; padding:6px 12px; background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.1); color:#fff; cursor:pointer; width:auto;">ABBRECHEN</button>
                <button id="trip-modal-save" class="cyber-button" style="font-size:10px; padding:6px 14px; background:rgba(255,123,0,0.2); border:1px solid var(--vgt-orange); color:var(--vgt-orange); cursor:pointer; width:auto;">REISE ERSTELLEN</button>
            </div>
        `;

        dialog.appendChild(modal);
        document.body.appendChild(dialog);

        dialog.querySelector('#trip-modal-close').addEventListener('click', () => dialog.remove());
        dialog.querySelector('#trip-modal-cancel').addEventListener('click', () => dialog.remove());

        dialog.querySelector('#trip-modal-save').addEventListener('click', async () => {
            const title = dialog.querySelector('#trip-input-title').value.trim();
            const dest = dialog.querySelector('#trip-input-dest').value.trim();
            const start = dialog.querySelector('#trip-input-start').value;
            const end = dialog.querySelector('#trip-input-end').value;
            const budget = parseFloat(dialog.querySelector('#trip-input-budget').value) || 2000;

            if (!title || !dest) {
                alert('Bitte Titel und Reiseziel eingeben.');
                return;
            }

            try {
                const res = await fetch(`${state.API_BASE}/v1/sphere/trips`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        title: title,
                        destination: dest,
                        start_date: start,
                        end_date: end,
                        budget_total_eur: budget,
                        status: 'planning'
                    })
                });
                if (res.ok) {
                    dialog.remove();
                    await loadTrips();
                }
            } catch (e) {
                alert('Fehler beim Erstellen der Reise: ' + e.message);
            }
        });
    }

    void loadTrips();
}
