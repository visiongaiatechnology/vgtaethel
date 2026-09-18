// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/markets.js
// Purpose: Multi-Asset Live Market Feed (Crypto, Commodities, Equities, Forex)

import { state } from '../../state.js';

export function renderMarketApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:8px; height:100%; color:#fff;';

    const kicker = document.createElement('div');
    kicker.style.cssText = 'font-size:10px; color:var(--vgt-orange); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'LIVE MARKET ENGINE // MULTI-ASSET FEED';

    const toolbar = document.createElement('div');
    toolbar.style.cssText = 'display:flex; justify-content:space-between; align-items:center; gap:8px;';

    const tabsContainer = document.createElement('div');
    tabsContainer.style.cssText = 'display:flex; gap:4px;';

    const categories = [
        { id: 'all', label: 'ALLE' },
        { id: 'commodity', label: '🛢️ ROHSTOFFE' },
        { id: 'crypto', label: '₿ KRYPTO' },
        { id: 'index', label: '📈 INDIZES' }
    ];

    let activeCategory = 'all';
    let rawQuotes = [];

    categories.forEach(cat => {
        const tabBtn = document.createElement('button');
        tabBtn.type = 'button';
        tabBtn.className = `sphere-app-action ${cat.id === activeCategory ? 'active' : ''}`;
        tabBtn.style.cssText = `font-size:9px; padding:3px 6px; background:none; border:1px solid rgba(255,255,255,0.1); color:#fff; border-radius:4px; cursor:pointer; ${cat.id === activeCategory ? 'background:rgba(255,123,0,0.2); border-color:var(--vgt-orange);' : ''}`;
        tabBtn.textContent = cat.label;
        tabBtn.addEventListener('click', () => {
            tabsContainer.querySelectorAll('button').forEach(b => {
                b.style.background = 'none';
                b.style.borderColor = 'rgba(255,255,255,0.1)';
            });
            tabBtn.style.background = 'rgba(255,123,0,0.2)';
            tabBtn.style.borderColor = 'var(--vgt-orange)';
            activeCategory = cat.id;
            filterAndRenderQuotes();
        });
        tabsContainer.appendChild(tabBtn);
    });

    const refreshBtn = document.createElement('button');
    refreshBtn.type = 'button';
    refreshBtn.className = 'cyber-button';
    refreshBtn.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(255,123,0,0.15); border:1px solid var(--vgt-orange); color:var(--vgt-orange); cursor:pointer; width:auto;';
    refreshBtn.textContent = '🔄 REFRESH';

    toolbar.append(tabsContainer, refreshBtn);

    const grid = document.createElement('div');
    grid.className = 'sphere-market-grid';
    grid.style.cssText = 'display:grid; grid-template-columns:repeat(auto-fill, minmax(160px, 1fr)); gap:8px; overflow-y:auto; flex:1; padding-right:4px;';

    container.append(kicker, toolbar, grid);
    body.appendChild(container);

    function filterAndRenderQuotes() {
        grid.replaceChildren();
        let filtered = rawQuotes;

        if (activeCategory !== 'all') {
            filtered = filtered.filter(q => q.category === activeCategory);
        }

        if (filtered.length === 0) {
            const empty = document.createElement('span');
            empty.style.cssText = 'font-size:11px; color:var(--vgt-text-dim); padding:10px;';
            empty.textContent = 'Keine Marktwerte in dieser Kategorie.';
            grid.appendChild(empty);
            return;
        }

        for (const quote of filtered) {
            const card = document.createElement('article');
            const isUp = Number(quote.change_24h_percent || 0) >= 0;
            card.style.cssText = `background:rgba(0,0,0,0.4); border:1px solid ${isUp ? 'rgba(57, 255, 20, 0.25)' : 'rgba(255, 0, 79, 0.25)'}; border-radius:6px; padding:8px 10px; display:flex; flex-direction:column; gap:3px;`;

            const symbolRow = document.createElement('div');
            symbolRow.style.cssText = 'display:flex; justify-content:space-between; align-items:baseline;';

            const symbol = document.createElement('strong');
            symbol.style.cssText = 'color:#fff; font-size:11px;';
            symbol.textContent = quote.symbol;

            const change = document.createElement('span');
            change.style.cssText = `font-size:9px; font-weight:bold; color:${isUp ? 'var(--vgt-green)' : 'var(--vgt-red)'};`;
            change.textContent = `${isUp ? '+' : ''}${Number(quote.change_24h_percent || 0).toFixed(2)}%`;

            symbolRow.append(symbol, change);

            const price = document.createElement('b');
            price.style.cssText = 'font-size:13px; color:#fff;';
            const curr = quote.currency === 'EUR' ? '€' : '$';
            price.textContent = `${curr}${Number(quote.price || 0).toLocaleString('de-DE', { minimumFractionDigits: 2 })}`;

            const name = document.createElement('small');
            name.style.cssText = 'font-size:9px; color:var(--vgt-text-dim); white-space:nowrap; overflow:hidden; text-overflow:ellipsis;';
            name.textContent = quote.name;

            card.append(symbolRow, price, name);
            grid.appendChild(card);
        }
    }

    const fetchQuotes = async () => {
        refreshBtn.disabled = true;
        try {
            const response = await fetch(`${state.API_BASE}/v1/markets`);
            if (response.ok) {
                const payload = await response.json();
                rawQuotes = payload.quotes || [];
                filterAndRenderQuotes();
            }
        } catch (error) {
            console.error('Failed to load quotes', error);
        } finally {
            refreshBtn.disabled = false;
        }
    };

    refreshBtn.addEventListener('click', () => void fetchQuotes());
    void fetchQuotes();
}
