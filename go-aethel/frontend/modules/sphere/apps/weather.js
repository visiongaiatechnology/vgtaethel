// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/weather.js
// Purpose: Live Weather Telemetry & City Lookup for Sphere Desktop

import { state } from '../../state.js';

export function renderWeatherApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:10px; color:#fff;';

    const kicker = document.createElement('div');
    kicker.className = 'sphere-app-kicker';
    kicker.style.cssText = 'font-size:10px; color:var(--vgt-cyan); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'LIVE WEATHER // SATELLITE TELEMETRY';

    const form = document.createElement('form');
    form.style.cssText = 'display:flex; gap:6px;';

    const cityInput = document.createElement('input');
    cityInput.type = 'text';
    cityInput.maxLength = 80;
    cityInput.value = localStorage.getItem('aethel_sphere_weather_city') || 'Köln';
    cityInput.placeholder = 'Stadt eingeben';
    cityInput.style.cssText = 'flex:1; background:rgba(0,0,0,0.4); border:1px solid rgba(0,240,255,0.25); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none; font-size:11px;';

    const refreshBtn = document.createElement('button');
    refreshBtn.type = 'submit';
    refreshBtn.className = 'cyber-button';
    refreshBtn.style.cssText = 'font-size:10px; padding:4px 10px; background:rgba(0,240,255,0.15); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    refreshBtn.textContent = 'ABRUFEN';

    form.append(cityInput, refreshBtn);

    const display = document.createElement('div');
    display.className = 'sphere-weather-display glass-card';
    display.style.cssText = 'background:rgba(0,0,0,0.35); border:1px solid rgba(255,255,255,0.06); border-radius:8px; padding:14px; display:flex; flex-direction:column; gap:6px;';

    const render = async () => {
        const query = cityInput.value.trim();
        if (!query) return;
        refreshBtn.disabled = true;
        display.textContent = 'Wetterdaten werden abgerufen...';

        try {
            const response = await fetch(`${state.API_BASE}/v1/weather?city=${encodeURIComponent(query)}`);
            if (!response.ok) throw new Error((await response.text()).slice(0, 160));
            const weather = await response.json();
            localStorage.setItem('aethel_sphere_weather_city', query);

            display.replaceChildren();
            const temp = document.createElement('strong');
            temp.style.cssText = 'font-size:24px; color:#fff;';
            temp.textContent = `${Number(weather.temperature_c || 0).toFixed(1)} °C`;

            const summary = document.createElement('span');
            summary.style.cssText = 'font-size:12px; color:var(--vgt-cyan);';
            summary.textContent = `${weather.summary || 'Klar'} · Wind ${Number(weather.wind_speed_kmh || 0).toFixed(1)} km/h`;

            const place = document.createElement('small');
            place.style.cssText = 'font-size:10px; color:var(--vgt-text-dim);';
            place.textContent = `${weather.city || query}${weather.country ? `, ${weather.country}` : ''} · Stand: ${weather.observed_at || 'aktuell'}`;

            display.append(temp, summary, place);
        } catch (error) {
            display.textContent = `Wetter nicht verfügbar: ${error.message}`;
        } finally {
            refreshBtn.disabled = false;
        }
    };

    form.addEventListener('submit', (e) => {
        e.preventDefault();
        void render();
    });

    container.append(kicker, form, display);
    body.appendChild(container);
    void render();
}
