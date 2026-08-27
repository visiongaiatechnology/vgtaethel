// STATUS: DIAMANT VGT SUPREME
// SHADOW OSINT renders all backend data through textContent/DOM construction.

import { ShadowCommandGlobe } from './shadow_globe.js';

const API = '/v1/shadow';
let initialized = false;
let shadowState = { status: {}, snapshot: {}, reports: [], regions: [], conflictLinks: [] };
let commandGlobe = null;

function el(tag, className = '', text = '') {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== '') node.textContent = String(text);
    return node;
}

function button(text, handler, className = 'shadow-button') {
    const node = el('button', className, text); node.type = 'button'; node.addEventListener('click', handler); return node;
}

function input(placeholder = '', type = 'text') {
    const node = el('input', 'shadow-input'); node.type = type; node.placeholder = placeholder; return node;
}

async function request(path, options = {}) {
    const controller = new AbortController();
    const timeoutMs = path === '/analyze' || path === '/daily' ? 180000 : 30000;
    const timeout = window.setTimeout(() => controller.abort(), timeoutMs);
    try {
        const response = await fetch(API + path, { credentials: 'same-origin', ...options, signal: options.signal || controller.signal });
        if (!response.ok) {
            let message = `HTTP ${response.status}`;
            try { message = String((await response.json()).error || message); } catch (_) { /* opaque fallback */ }
            throw new Error(message);
        }
        const type = response.headers.get('content-type') || '';
        return type.includes('json') ? response.json() : response.text();
    } finally {
        window.clearTimeout(timeout);
    }
}

function jsonRequest(path, method, body) {
    return request(path, { method, headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' }, body: JSON.stringify(body) });
}

function flash(message, level = 'info') {
    const node = document.getElementById('shadow-flash'); if (!node) return; node.textContent = message; node.dataset.level = level;
}

function panel(id, title, className = '') {
    const root = el('section', `shadow-panel ${className}`.trim()); root.id = id;
    const head = el('div', 'shadow-panel-head'); head.append(el('h2', '', title)); root.append(head); return root;
}

function metric(id, label) {
    const root = el('div', 'shadow-metric'); const value = el('strong', '', '0'); value.id = `shadow-metric-${id}`; root.append(value, el('span', '', label)); return root;
}

function record(title, metadata = [], detail = '') {
    const root = el('article', 'shadow-record'); root.append(el('div', 'shadow-record-title', title));
    const meta = el('div', 'shadow-record-meta'); metadata.forEach(value => meta.append(el('span', '', value))); root.append(meta);
    if (detail) root.append(el('div', 'shadow-report-body', detail)); return root;
}

function buildShadowUI() {
    const view = document.getElementById('view-shadow'); if (!view) return;
    const shell = el('div', 'shadow-shell');
    const header = el('header', 'shadow-header'); const identity = el('div');
    identity.append(el('div', 'shadow-kicker', 'VGT AETHEL // STRATEGIC COMMAND ENVIRONMENT'), el('h1', '', 'SHADOW // GLOBAL COMMAND'), el('p', '', 'Military · Geopolitics · Energy · Economy · Cyber · Space'));
    const headerStatus=el('div','shadow-header-status');const live=el('span','shadow-live','LIVE INTELLIGENCE');const classification=el('div','shadow-classification','BETA v3 // LOCAL SOVEREIGN');headerStatus.append(live,classification);header.append(identity,headerStatus);
    const metrics = el('div', 'shadow-metrics');
    [['sources','Sources'],['enabled','Enabled'],['pending','Pending Intel'],['processed','Processed'],['reports','Dossiers'],['regions','AI Regions']].forEach(([id,label]) => metrics.append(metric(id,label)));

    const tabs = el('nav', 'shadow-tabs');
    [['tactical','TACTICAL OVERVIEW'],['intercepts','LATEST INTERCEPTS'],['reports','DOSSIER ARCHIVE'],['sources','SOURCE CONTROL'],['doctrine','SHADOW DOCTRINE']].forEach(([id,label], index) => {
        const tab = button(label, () => activateTab(id), 'shadow-tab'); tab.dataset.shadowTab = id; if (index === 0) tab.classList.add('active'); tabs.append(tab);
    });

    const tactical = el('main', 'shadow-grid'); tactical.dataset.shadowWorkspace = 'tactical';
    const mapPanel = panel('shadow-map-panel', 'ORBITAL CONFLICT COMMAND // AI-EVIDENCE ONLY');
    const mapWrap = el('div', 'shadow-map-wrap'); const canvas = el('canvas', 'shadow-map'); canvas.id = 'shadow-conflict-globe';const overlay=el('canvas','shadow-map shadow-map-overlay');overlay.id='shadow-conflict-overlay';
    const legend = el('div', 'shadow-map-legend'); [['#38c878','STABLE'],['#e6c84c','TENSION'],['#ff7a2f','ESCALATION'],['#ff244d','WAR']].forEach(([color,label]) => { const item=el('span'); const dot=el('i'); dot.style.backgroundColor=color; item.append(dot,document.createTextNode(label)); legend.append(item); }); mapWrap.append(canvas,legend); mapPanel.append(mapWrap);
    const telemetry=el('div','shadow-globe-telemetry');telemetry.append(el('span','','WEBGL // 3D'),el('span','','DRAG TO ROTATE'),el('span','','WHEEL TO ZOOM'));const selection=el('aside','shadow-globe-selection','SELECT A REGION OR CONFLICT VECTOR');selection.id='shadow-globe-selection';mapWrap.replaceChildren(canvas,overlay,telemetry,selection,legend);
    const control = panel('shadow-batch-control','INTELLIGENCE REACTOR'); const progressLabel=el('div','shadow-record-meta','WAITING FOR BUFFER');progressLabel.id='shadow-progress-label';const progress=el('div','shadow-progress');const bar=el('span');bar.id='shadow-progress-bar';progress.append(bar);const actions=el('div','shadow-actions');
    const collect=button('COLLECT NEXT SOURCES',()=>void collectShadow());collect.id='shadow-collect';const analyze=button('ANALYZE 40–60',()=>void analyzeShadow(false));analyze.id='shadow-analyze';const autonomy=button('AUTONOMY: OFF',()=>void toggleShadowAutonomy());autonomy.id='shadow-autonomy';autonomy.setAttribute('aria-pressed','false');const daily=button('GENERATE DAILY MASTER REPORT',()=>void analyzeShadow(true));daily.id='shadow-daily';const clear=button('ALLES LÖSCHEN',()=>void clearShadowData(),'shadow-button shadow-button-danger');clear.id='shadow-clear-data';const analysisState=el('div','shadow-analysis-state','MANUAL CONTROL READY');analysisState.id='shadow-analysis-state';analysisState.setAttribute('role','status');analysisState.setAttribute('aria-live','polite');actions.append(collect,analyze,autonomy,daily,clear);control.append(progressLabel,progress,actions,analysisState);
    const regionList=panel('shadow-region-list-panel','REGIONAL SECURITY MATRIX');regionList.append(el('div','shadow-list'));regionList.lastChild.id='shadow-region-list';
    const conflictList=panel('shadow-conflict-list-panel','DIRECTED CONFLICT VECTORS');conflictList.append(el('div','shadow-list'));conflictList.lastChild.id='shadow-conflict-list';
    const forecasts=panel('shadow-forecast-panel','72H FORECAST MATRIX // EVIDENCE + MARKET PULSE','shadow-span-2');const marketSnapshot=el('div','shadow-market-strip');marketSnapshot.id='shadow-market-snapshot';const forecastList=el('div','shadow-forecast-grid');forecastList.id='shadow-forecast-list';forecasts.append(marketSnapshot,forecastList);
    const latestDossiers=panel('shadow-latest-dossiers','LATEST DOSSIERS // EXPANDABLE','shadow-span-2');latestDossiers.append(el('div','shadow-dossier-stack'));latestDossiers.lastChild.id='shadow-latest-dossier-list';
    tactical.append(mapPanel,control,regionList,conflictList,forecasts,latestDossiers);

    const intercepts=el('main','shadow-grid');intercepts.dataset.shadowWorkspace='intercepts';intercepts.classList.add('hidden');const stream=panel('shadow-intel-stream','LATEST INTERCEPTS','shadow-span-2');stream.append(el('div','shadow-list'));stream.lastChild.id='shadow-stream-list';intercepts.append(stream);

    const reports = el('main','shadow-grid');reports.dataset.shadowWorkspace='reports';reports.classList.add('hidden');const archive=panel('shadow-report-list-panel','DOSSIER ARCHIVE');archive.append(el('div','shadow-list'));archive.lastChild.id='shadow-report-list';const viewer=panel('shadow-report-viewer','REPORT READER');viewer.append(el('div','shadow-report-body','Select a dossier.'));viewer.lastChild.id='shadow-report-body';reports.append(archive,viewer);

    const sources = el('main','shadow-grid');sources.dataset.shadowWorkspace='sources';sources.classList.add('hidden');const sourceControl=panel('shadow-source-control','EDITABLE SOURCE REGISTRY','shadow-span-2');const form=el('form','shadow-form-grid');const name=input('Source name');const url=input('https://…','url');const kind=el('select','shadow-select');['rss','telegram','web'].forEach(value=>{const option=el('option','',value.toUpperCase());option.value=value;kind.append(option)});const domain=input('military / cyber / energy');const add=el('button','shadow-button','ADD SOURCE');add.type='submit';form.append(name,url,kind,domain,add);form.addEventListener('submit',event=>{event.preventDefault();void saveNewSource({name,url,kind,domain})});const list=el('div','shadow-list');list.id='shadow-source-list';sourceControl.append(form,list);sources.append(sourceControl);

    const doctrine = el('main','shadow-grid');doctrine.dataset.shadowWorkspace='doctrine';doctrine.classList.add('hidden');const doctrinePanel=panel('shadow-doctrine-panel','SHADOW SYSTEM PROMPT','shadow-span-2');const textarea=el('textarea','shadow-textarea');textarea.id='shadow-system-prompt';const savePrompt=button('SAVE SEALED DOCTRINE',()=>void saveDoctrine());doctrinePanel.append(textarea,el('div','shadow-actions'));doctrinePanel.lastChild.append(savePrompt);doctrine.append(doctrinePanel);

    const flashNode=el('div','shadow-flash','SHADOW mode standby.');flashNode.id='shadow-flash';flashNode.setAttribute('role','status');flashNode.setAttribute('aria-live','polite');
    shell.append(header,metrics,tabs,tactical,intercepts,reports,sources,doctrine,flashNode);view.replaceChildren(shell);
}

function activateTab(id) {
    document.querySelectorAll('[data-shadow-workspace]').forEach(node => node.classList.toggle('hidden', node.dataset.shadowWorkspace !== id));
    document.querySelectorAll('[data-shadow-tab]').forEach(node => node.classList.toggle('active', node.dataset.shadowTab === id));
}

export function initShadowOSINT() {
    if (initialized || !document.getElementById('view-shadow')) return; initialized=true;buildShadowUI();
    window.setInterval(()=>{const view=document.getElementById('view-shadow');if(view&&!view.classList.contains('hidden'))void refreshShadowOSINT()},5000);
    const mapWrap=document.querySelector('#shadow-map-panel .shadow-map-wrap');if(mapWrap){commandGlobe=new ShadowCommandGlobe(mapWrap,renderGlobeSelection);commandGlobe.start()}
}

export async function refreshShadowOSINT() {
    if (!initialized) initShadowOSINT();
    try {
        const [status,snapshot,reports,regions]=await Promise.all([request('/status'),request('/snapshot'),request('/reports'),request('/regions')]);
        shadowState={status,snapshot,reports:reports.reports||[],regions:regions.regions||[],conflictLinks:regions.conflict_links||[]};renderStatus();renderStream();renderRegions();renderConflictLinks();renderReports();renderForecastMatrix();renderLatestDossiers();renderSources();
        const prompt=document.getElementById('shadow-system-prompt');if(prompt&&document.activeElement!==prompt)prompt.value=snapshot.system_prompt||'';if(commandGlobe)commandGlobe.setData(shadowState.regions,shadowState.conflictLinks);if(status.last_analysis_error)flash(`LAST ANALYSIS FAILED: ${status.last_analysis_error}`,'error');else if(status.analysis_running)flash(`SHADOW analysis running with ${status.analysis_model_id||'selected core'}…`);else flash('SHADOW command fabric synchronized.','success');
    } catch(error){flash(`SHADOW unavailable: ${error.message}`,'error')}
}

function renderStatus(){const s=shadowState.status;const values={sources:s.sources||0,enabled:s.enabled_sources||0,pending:s.pending_items||0,processed:s.processed_items||0,reports:s.reports||0,regions:shadowState.regions.length};Object.entries(values).forEach(([id,value])=>{const node=document.getElementById(`shadow-metric-${id}`);if(node)node.textContent=String(value)});const pct=Math.min(100,((s.pending_items||0)/(s.batch_min||40))*100);const bar=document.getElementById('shadow-progress-bar');if(bar)bar.style.width=`${pct}%`;const label=document.getElementById('shadow-progress-label');const windowLabel=`${s.intake_window_hours||24}H WINDOW // ${s.context_dossiers||0} CONTEXT DOSSIERS`;if(label)label.textContent=s.analysis_running?`AI ANALYSIS RUNNING // ${s.analysis_model_id||'SELECTED CORE'} // ${windowLabel}`:(s.pending_items||0)>=40?`${Math.min(s.pending_items,60)} ITEMS // ANALYSIS READY // ${windowLabel}`:`${s.pending_items||0} / ${s.batch_min||40} ITEMS // ACCUMULATING // ${windowLabel}`;const analyze=document.getElementById('shadow-analyze');if(analyze){analyze.disabled=(s.pending_items||0)<40||s.analysis_running;analyze.title=s.analysis_running?'An analysis is already running.':(s.pending_items||0)<40?`At least ${s.batch_min||40} pending items from the last ${s.intake_window_hours||24} hours are required.`:'Start one evidence-bound batch.'}const autonomy=document.getElementById('shadow-autonomy');if(autonomy){autonomy.textContent=`AUTONOMY: ${s.autonomy_enabled?'ON':'OFF'}`;autonomy.classList.toggle('active',Boolean(s.autonomy_enabled));autonomy.setAttribute('aria-pressed',String(Boolean(s.autonomy_enabled)))}const clear=document.getElementById('shadow-clear-data');if(clear){clear.disabled=Boolean(s.analysis_running||s.collection_running);clear.title=clear.disabled?'Wait until the active SHADOW operation has finished.':'Deletes intercepts, dossiers and assessments; preserves sources, doctrine and API keys.'}const state=document.getElementById('shadow-analysis-state');if(state){state.dataset.level=s.last_analysis_error?'error':s.analysis_running?'running':'ready';state.textContent=s.last_analysis_error?`LAST RUN FAILED // ${s.last_analysis_error}`:s.analysis_running?`${s.autonomy_enabled?'AUTONOMOUS':'MANUAL'} DOSSIER GENERATION IN PROGRESS${s.analysis_started_at?` // SINCE ${new Date(s.analysis_started_at).toLocaleTimeString()}`:''}`:s.last_analysis_at?`LAST ANALYSIS ${new Date(s.last_analysis_at).toLocaleString()} // READY`:`${s.autonomy_enabled?'AUTONOMOUS COLLECTION + BATCH ANALYSIS ARMED':'MANUAL CONTROL READY'}`}const daily=document.getElementById('shadow-daily');if(daily)daily.disabled=s.analysis_running||shadowState.reports.filter(report=>report.kind==='batch'&&new Date(report.created_at).toDateString()===new Date().toDateString()).length<2}

function renderStream(){const list=document.getElementById('shadow-stream-list');if(!list)return;list.replaceChildren();const items=(shadowState.snapshot.buffer||[]).slice().reverse().slice(0,80);if(!items.length){list.append(el('div','shadow-empty','NO INTERCEPTS'));return}items.forEach(item=>list.append(record(item.title,[item.source_name,item.domain,item.processed?'PROCESSED':'PENDING',new Date(item.published_at).toLocaleString()],item.summary||'')))}

function regionColor(region){if(region.conflict_level==='WAR')return'#ff244d';if(region.conflict_level==='ESCALATION')return'#ff7a2f';if(region.conflict_level==='TENSION')return'#e6c84c';return'#38c878'}
function renderRegions(){const list=document.getElementById('shadow-region-list');if(!list)return;list.replaceChildren();if(!shadowState.regions.length){list.append(el('div','shadow-empty','AWAITING AI-EVIDENCE ASSESSMENT'));return}shadowState.regions.forEach(region=>{const row=record(region.region_name,[`SECURITY ${region.security_score}/100`,region.conflict_level,`CONF ${region.confidence}%`,region.trend],region.assessment);row.style.borderLeftColor=regionColor(region);list.append(row)})}

function renderConflictLinks(){const list=document.getElementById('shadow-conflict-list');if(!list)return;list.replaceChildren();if(!shadowState.conflictLinks.length){list.append(el('div','shadow-empty','NO EVIDENCE-BOUND HOSTILE VECTORS'));return}shadowState.conflictLinks.forEach(link=>{const hostile=link.action!=='MILITARY_SUPPORT';const row=record(`${link.attacker_name}  →  ${link.target_name}`,[link.action,`CONF ${link.confidence}%`,`${link.evidence_ids?.length||0} EVIDENCE`],link.assessment);row.classList.add(hostile?'shadow-vector-hostile':'shadow-vector-support');row.tabIndex=0;const select=()=>renderGlobeSelection({type:'link',value:link});row.addEventListener('click',select);row.addEventListener('keydown',event=>{if(event.key==='Enter'||event.key===' '){event.preventDefault();select()}});list.append(row)})}

function renderGlobeSelection(selection){const node=document.getElementById('shadow-globe-selection');if(!node)return;node.replaceChildren();if(!selection?.value){node.textContent='SELECT A REGION OR CONFLICT VECTOR';return}const value=selection.value;if(selection.type==='link'){node.append(el('span','shadow-selection-kicker',value.action),el('strong','',`${value.attacker_name} → ${value.target_name}`),el('p','',value.assessment||'No assessment supplied.'),el('small','',`CONFIDENCE ${value.confidence}% // ${value.evidence_ids?.length||0} EVIDENCE OBJECTS`));return}node.append(el('span','shadow-selection-kicker',value.conflict_level),el('strong','',value.region_name),el('p','',value.assessment||'No assessment supplied.'),el('small','',`SECURITY ${value.security_score}/100 // CONFIDENCE ${value.confidence}%`))}

function reportText(report){const vectors=(report.conflict_links||[]).map(link=>`${link.attacker_name} → ${link.target_name} // ${link.action} // CONF ${link.confidence}%\n${link.assessment}`).join('\n\n')||'—';const forecasts=(report.forecast_matrix||[]).map(item=>`${item.sector} // ${item.horizon} // ${item.direction||'SCENARIO'} // ${item.probability}%\n${(item.instruments||[]).join(' · ')}\n${item.prediction}`).join('\n\n')||'—';const markets=(report.market_snapshot||[]).map(item=>`${item.symbol} // ${formatMarketPrice(item)} ${item.currency||''} // ${(Number(item.change_24h_percent)||0).toFixed(2)}% / 24H`).join('\n')||'—';return `THREAT: ${report.threat_level}\nITEMS: ${report.items_analyzed}\nSHA-256: ${report.content_sha256}\n\nEXECUTIVE SUMMARY\n${report.summary}\n\nTACTICAL SITUATION\n${report.situation}\n\nCUI BONO\n${report.cui_bono}\n\nSTRATEGIC REALITY\n${report.strategic_reality}\n\nDIVERGENCES\n${report.divergences||'—'}\n\nCONFIRMED VECTORS\n${report.confirmed_vectors||'—'}\n\nDIRECTED CONFLICT VECTORS\n${vectors}\n\n72H FORECAST MATRIX\n${forecasts}\n\nSPHERE MARKET PULSE SNAPSHOT\n${markets}`}
function renderReports(){const list=document.getElementById('shadow-report-list');if(!list)return;list.replaceChildren();if(!shadowState.reports.length){list.append(el('div','shadow-empty','NO DOSSIERS'));return}shadowState.reports.forEach(report=>{const actions=el('div','shadow-actions');actions.append(button('READ',()=>{const body=document.getElementById('shadow-report-body');if(body)body.textContent=reportText(report)}),button('JSON',()=>void exportReport(report.id,'json')),button('MARKDOWN',()=>void exportReport(report.id,'markdown')));const row=record(`${report.kind==='daily'?'MASTER // ':''}${report.threat_level} // ${new Date(report.created_at).toLocaleString()}`,[`${report.items_analyzed} ITEMS`,`${report.regions?.length||0} REGIONS`,`${report.conflict_links?.length||0} VECTORS`],report.summary);row.append(actions);list.append(row)})}

function renderLatestDossiers(){const list=document.getElementById('shadow-latest-dossier-list');if(!list)return;list.replaceChildren();const reports=shadowState.reports.slice(0,6);if(!reports.length){list.append(el('div','shadow-empty','NO COMPLETED DOSSIERS'));return}reports.forEach((report,index)=>{const details=el('details','shadow-dossier-disclosure');details.open=index===0;const summary=el('summary');summary.append(el('strong','',`${report.kind==='daily'?'MASTER // ':''}${report.threat_level}`),el('span','',`${new Date(report.created_at).toLocaleString()} // ${report.items_analyzed} ITEMS`));const body=el('div','shadow-dossier-preview',report.summary);const actions=el('div','shadow-actions');actions.append(button('OPEN ARCHIVE',()=>{activateTab('reports');const reader=document.getElementById('shadow-report-body');if(reader)reader.textContent=reportText(report)}),button('JSON',()=>void exportReport(report.id,'json')),button('MARKDOWN',()=>void exportReport(report.id,'markdown')));details.append(summary,body,actions);list.append(details)})}

function formatMarketPrice(point){if(!Number.isFinite(Number(point.price)))return'—';const digits=Number(point.price)<10?4:2;return new Intl.NumberFormat(undefined,{maximumFractionDigits:digits}).format(Number(point.price))}
function renderForecastMatrix(){const market=document.getElementById('shadow-market-snapshot');const list=document.getElementById('shadow-forecast-list');if(!market||!list)return;market.replaceChildren();list.replaceChildren();const report=shadowState.reports[0];if(!report){market.append(el('div','shadow-empty','AWAITING TRUSTED MARKET PULSE SNAPSHOT'));list.append(el('div','shadow-empty','AWAITING FIRST EVIDENCE-BOUND DOSSIER'));return}const priority=['BTC','GOLD','BRENT','WTI','ETH','SP500','NASDAQ','DAX','EURUSD','USDJPY'];const points=(report.market_snapshot||[]).slice().sort((a,b)=>priority.indexOf(a.symbol)-priority.indexOf(b.symbol));points.forEach(point=>{const card=el('article','shadow-market-point');const change=Number(point.change_24h_percent)||0;card.dataset.direction=change>0?'up':change<0?'down':'flat';card.append(el('strong','',point.symbol),el('span','',`${formatMarketPrice(point)} ${point.currency||''}`),el('small','',`${change>=0?'+':''}${change.toFixed(2)}% / 24H`));market.append(card)});const forecasts=(report.forecast_matrix||[]).slice().sort((a,b)=>(a.horizon==='72h'?0:1)-(b.horizon==='72h'?0:1));if(!forecasts.length){list.append(el('div','shadow-empty','DOSSIER CONTAINS NO FORECAST MATRIX'));return}forecasts.forEach(forecast=>{const card=el('article','shadow-forecast-card');card.dataset.direction=String(forecast.direction||'').toLowerCase();const head=el('div','shadow-forecast-head');head.append(el('strong','',forecast.sector||'SCENARIO'),el('span','',`${forecast.horizon||'—'} // ${forecast.probability||0}%`));const instruments=(forecast.instruments||[]).join(' · ');card.append(head,el('div','shadow-forecast-direction',`${forecast.direction||'SCENARIO'}${instruments?` // ${instruments}`:''}`),el('p','',forecast.prediction||''),el('small','',`${forecast.evidence_ids?.length||0} EVIDENCE OBJECTS // SCENARIO, NOT CERTAINTY`));list.append(card)})}

function renderSources(){const list=document.getElementById('shadow-source-list');if(!list)return;list.replaceChildren();(shadowState.snapshot.sources||[]).forEach(source=>{const row=el('div','shadow-source-row');const name=input();name.value=source.name;const url=input('', 'url');url.value=source.url;const kind=el('select','shadow-select');['rss','telegram','web'].forEach(value=>{const option=el('option','',value.toUpperCase());option.value=value;option.selected=source.type===value;kind.append(option)});kind.title='Collector type';const domain=input();domain.value=source.domain;domain.classList.add('shadow-source-domain');const enabled=el('input','shadow-toggle');enabled.type='checkbox';enabled.checked=source.enabled;enabled.title='Enabled';const actions=el('div','shadow-actions');actions.append(button('SAVE',()=>void updateSource(source,name,url,kind,domain,enabled)),button('DELETE',()=>void deleteSource(source.id)));row.append(name,url,kind,domain,enabled,actions);if(source.last_error)row.title=source.last_error;list.append(row)})}

async function collectShadow(){const node=document.getElementById('shadow-collect');if(node)node.disabled=true;flash('Collecting next bounded source rotation…');try{const result=await jsonRequest('/collect','POST',{source_limit:8});flash(`${result.added} new intercepts collected.`,'success');await refreshShadowOSINT()}catch(error){flash(`Collection failed: ${error.message}`,'error')}finally{if(node)node.disabled=false}}
async function analyzeShadow(daily){const id=daily?'shadow-daily':'shadow-analyze';const node=document.getElementById(id);const state=document.getElementById('shadow-analysis-state');if(node)node.disabled=true;if(state){state.dataset.level='running';state.textContent=daily?'DAILY MASTER SYNTHESIS RUNNING':'MANUAL 40–60 ITEM ANALYSIS RUNNING'}flash(daily?'Generating daily master synthesis…':'Analyzing evidence-bound 40–60 item batch…');try{const model=document.getElementById('model-dropdown')?.value||'';const report=await jsonRequest(daily?'/daily':'/analyze','POST',{model_id:model});if(state){state.dataset.level='ready';state.textContent=`DOSSIER GENERATED // ${report.items_analyzed} ITEMS`}flash(`SHADOW ${report.kind} dossier generated.`,'success');await refreshShadowOSINT();activateTab('reports')}catch(error){if(state){state.dataset.level='error';state.textContent=`ANALYSIS FAILED // ${error.message}`}flash(`Analysis failed: ${error.message}`,'error')}finally{if(node)node.disabled=false}}
async function toggleShadowAutonomy(){const current=Boolean(shadowState.status.autonomy_enabled);const node=document.getElementById('shadow-autonomy');if(node)node.disabled=true;try{const model=document.getElementById('model-dropdown')?.value||'';const status=await jsonRequest('/autonomy','PUT',{enabled:!current,model_id:model});shadowState.status={...shadowState.status,...status};renderStatus();flash(`SHADOW autonomy ${status.autonomy_enabled?'armed':'disarmed'}.`,'success')}catch(error){const state=document.getElementById('shadow-analysis-state');if(state){state.dataset.level='error';state.textContent=`AUTONOMY CHANGE FAILED // ${error.message}`}flash(`Autonomy change failed: ${error.message}`,'error')}finally{if(node)node.disabled=false}}
async function saveNewSource(fields){try{await jsonRequest('/sources','POST',{name:fields.name.value.trim(),url:fields.url.value.trim(),type:fields.kind.value,domain:fields.domain.value.trim()||'military',enabled:true,priority:3});fields.name.value='';fields.url.value='';await refreshShadowOSINT();flash('Source added.','success')}catch(error){flash(`Source rejected: ${error.message}`,'error')}}
async function updateSource(source,name,url,kind,domain,enabled){try{await jsonRequest('/sources','PUT',{...source,name:name.value.trim(),url:url.value.trim(),type:kind.value,domain:domain.value.trim(),enabled:enabled.checked});await refreshShadowOSINT();flash('Source updated.','success')}catch(error){flash(`Source update failed: ${error.message}`,'error')}}
async function deleteSource(id){try{await request(`/sources?id=${encodeURIComponent(id)}`,{method:'DELETE'});await refreshShadowOSINT();flash('Source deleted.','success')}catch(error){flash(`Source deletion failed: ${error.message}`,'error')}}
async function clearShadowData(){const confirmed=window.confirm('Alle SHADOW-Meldungen, Dossiers, Regionen und Forecasts dauerhaft löschen?\n\nQuellen, Systemprompt und API-Keys bleiben erhalten. Die Autonomie wird deaktiviert.');if(!confirmed)return;const node=document.getElementById('shadow-clear-data');if(node)node.disabled=true;flash('Clearing SHADOW operational data…');try{await jsonRequest('/data','DELETE',{confirmation:'DELETE SHADOW DATA'});shadowState={...shadowState,status:{},snapshot:{},reports:[],regions:[],conflictLinks:[]};if(commandGlobe)commandGlobe.setData([],[]);await refreshShadowOSINT();flash('SHADOW operational data cleared. Sources, doctrine and API keys were preserved.','success')}catch(error){flash(`SHADOW data deletion failed: ${error.message}`,'error')}finally{if(node)node.disabled=false}}
async function saveDoctrine(){const prompt=document.getElementById('shadow-system-prompt');try{await jsonRequest('/prompt','PUT',{system_prompt:prompt.value});flash('SHADOW doctrine sealed.','success')}catch(error){flash(`Doctrine rejected: ${error.message}`,'error')}}
async function exportReport(id,format){try{const response=await fetch(`${API}/export?id=${encodeURIComponent(id)}&format=${encodeURIComponent(format)}`);if(!response.ok)throw new Error(`HTTP ${response.status}`);const blob=await response.blob();const url=URL.createObjectURL(blob);const link=el('a');link.href=url;link.download=`shadow-${id}.${format==='markdown'?'md':'json'}`;link.click();URL.revokeObjectURL(url)}catch(error){flash(`Export failed: ${error.message}`,'error')}}
