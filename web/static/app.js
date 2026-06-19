const API = window.location.origin;
let ws = null;
let selectedDevice = null;
let devices = [];

document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
    loadDevices();
    initUpload();
    loadChatHistory();
    initChatInput();
    initMobileNav();
});

/* ── WebSocket ── */
function initWebSocket() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${proto}//${location.host}/ws`);
    ws.onopen = () => { setConnStatus(true); };
    ws.onmessage = e => { try { handleMsg(JSON.parse(e.data)); } catch(_){} };
    ws.onclose = () => { setConnStatus(false); setTimeout(initWebSocket, 3000); };
    ws.onerror = () => setConnStatus(false);
}

function setConnStatus(ok) {
    const el = document.getElementById('conn-status');
    if (ok) {
        el.className = 'flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-medium';
        el.innerHTML = '<span class="relative flex h-2 w-2"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span><span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span></span>已连接';
    } else {
        el.className = 'flex items-center gap-2 px-3 py-1.5 rounded-full bg-amber-500/10 border border-amber-500/20 text-amber-400 text-xs font-medium';
        el.innerHTML = '<span class="relative flex h-2 w-2"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span><span class="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span></span>重连中…';
    }
}

function handleMsg(msg) {
    if (msg.type === 'chat') return appendMsg(msg, false);
    if (msg.type === 'device_found') { toast(`发现设备: ${msg.name}`, 'info'); loadDevices(); }
    if (msg.type === 'device_lost') { toast(`设备离线: ${msg.name}`, 'info'); loadDevices(); }
}

/* ── Devices ── */
async function loadDevices() {
    try {
        const r = await fetch(`${API}/api/peers`);
        devices = await r.json();
    } catch { devices = []; }
    renderDevices();
}

function renderDevices() {
    const el = document.getElementById('device-list');
    document.getElementById('peers-count').textContent = devices.length;
    if (!devices.length) {
        el.innerHTML = `<div class="flex flex-col items-center justify-center h-full text-center py-12 opacity-50"><i class="ri-radar-line text-4xl text-slate-500 mb-3"></i><p class="text-sm text-slate-400 font-medium">搜索设备中…</p><p class="text-xs text-slate-600 mt-1">确保其他设备已启动</p></div>`;
        return;
    }
    el.innerHTML = devices.map((d,i) => `
        <div class="device-item glass-card rounded-xl p-3 flex items-center gap-3 cursor-pointer ${selectedDevice===d.id?'active':''}"
             onclick="pickDevice('${d.id}')" style="animation-delay:${i*50}ms">
            <div class="w-11 h-11 rounded-xl bg-gradient-to-br from-brand-500/80 to-purple-500/80 flex items-center justify-center text-xl flex-shrink-0 shadow-lg shadow-brand-500/20">${icon4type(d.type)}</div>
            <div class="flex-1 min-w-0">
                <p class="text-sm font-semibold text-white truncate">${esc(d.name)}</p>
                <p class="text-[11px] text-slate-500 flex items-center gap-1.5 mt-0.5"><span class="w-1.5 h-1.5 rounded-full bg-emerald-500 flex-shrink-0"></span>${d.ip}</p>
            </div>
            <i class="ri-arrow-right-s-line text-slate-600"></i>
        </div>`).join('');
}

function icon4type(t) {
    return {windows:'💻',android:'📱',linux:'🐧',macos:'🍎'}[t]||'💻';
}

function pickDevice(id) {
    selectedDevice = id;
    document.getElementById('message-input').disabled = false;
    document.getElementById('send-btn').disabled = false;
    renderDevices();
    const d = devices.find(x=>x.id===id);
    if (d) toast(`已选择 ${d.name}`, 'success');
}

/* ── Upload ── */
function initUpload() {
    const zone = document.getElementById('upload-zone');
    const input = document.getElementById('file-input');
    document.getElementById('upload-btn').onclick = e => { e.stopPropagation(); input.click(); };
    zone.onclick = () => input.click();
    zone.ondragover = e => { e.preventDefault(); zone.classList.add('dragover'); };
    zone.ondragleave = () => zone.classList.remove('dragover');
    zone.ondrop = e => { e.preventDefault(); zone.classList.remove('dragover'); uploadFiles(e.dataTransfer.files); };
    input.onchange = e => { uploadFiles(e.target.files); e.target.value=''; };
}

async function uploadFiles(fileList) {
    if (!selectedDevice) return toast('请先选择目标设备','error');
    for (const f of fileList) await doUpload(f);
}

async function doUpload(file) {
    const id = 't-'+Date.now()+'-'+Math.random().toString(36).slice(2,7);
    addTransferItem(id, file.name, file.size);
    const fd = new FormData(); fd.append('file', file);
    try {
        const r = await fetch(`${API}/api/upload/Downloads`, {method:'POST',body:fd});
        if (r.ok) { setTransferDone(id, true); toast(`${file.name} 发送成功`,'success'); }
        else { setTransferDone(id, false); toast(`${file.name} 发送失败`,'error'); }
    } catch { setTransferDone(id, false); toast(`${file.name} 出错`,'error'); }
}

function addTransferItem(id, name, size) {
    const el = document.getElementById('transfer-list');
    const ext = name.split('.').pop().toLowerCase();
    const icon = {jpg:'🖼',jpeg:'🖼',png:'🖼',gif:'🖼',mp4:'🎬',avi:'🎬',mp3:'🎵',pdf:'📄',doc:'📄',zip:'📦',rar:'📦'}[ext]||'📄';
    const html = `<div id="${id}" class="glass-card rounded-xl p-3.5 flex items-center gap-3 animate-fade-in">
        <div class="w-11 h-11 rounded-xl bg-brand-500/10 flex items-center justify-center text-xl flex-shrink-0">${icon}</div>
        <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-white truncate">${esc(name)}</p>
            <div class="mt-2 h-1.5 rounded-full bg-white/5 overflow-hidden"><div class="progress-bar h-full rounded-full" style="width:0%"></div></div>
            <div class="flex justify-between mt-1.5"><span class="text-[11px] text-slate-500 upload-status">发送中…</span><span class="text-[11px] text-slate-500">${fmtSize(size)}</span></div>
        </div>
    </div>`;
    el.insertAdjacentHTML('afterbegin', html);
}

function setTransferDone(id, ok) {
    const el = document.getElementById(id);
    if (!el) return;
    const bar = el.querySelector('.progress-bar');
    const status = el.querySelector('.upload-status');
    bar.style.width = '100%';
    bar.classList.remove('progress-bar');
    bar.style.background = ok ? '#10b981' : '#ef4444';
    status.textContent = ok ? '✓ 完成' : '✕ 失败';
    status.className = `text-[11px] ${ok?'text-emerald-400':'text-red-400'}`;
}

function fmtSize(b) {
    if (!b) return '0 B';
    const u = ['B','KB','MB','GB','TB'];
    const i = Math.floor(Math.log(b)/Math.log(1024));
    return (b/Math.pow(1024,i)).toFixed(1)+' '+u[i];
}

/* ── Chat ── */
async function loadChatHistory() {
    try { const r = await fetch(`${API}/api/chat/history`); (await r.json()).forEach(m=>appendMsg(m,m.from==='local')); } catch {}
}

function appendMsg(msg, sent) {
    const el = document.getElementById('chat-messages');
    const t = new Date(msg.timestamp).toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit'});
    el.insertAdjacentHTML('beforeend', `
        <div class="flex ${sent?'justify-end':'justify-start'} animate-fade-in">
            <div class="msg-bubble ${sent?'msg-sent':'msg-recv'} px-4 py-2.5 rounded-2xl ${sent?'rounded-br-md':'rounded-bl-md'}">
                <p class="text-sm leading-relaxed">${esc(msg.content)}</p>
                <p class="text-[10px] ${sent?'text-white/50':'text-slate-500'} mt-1">${sent?'我':esc(msg.from_name||'')} · ${t}</p>
            </div>
        </div>`);
    el.scrollTop = el.scrollHeight;
}

function initChatInput() {
    const input = document.getElementById('message-input');
    const btn = document.getElementById('send-btn');
    const send = () => {
        const c = input.value.trim();
        if (!c||!selectedDevice||!ws||ws.readyState!==1) return;
        ws.send(JSON.stringify({type:'chat',payload:{to:selectedDevice,content:c}}));
        appendMsg({from:'local',from_name:'我',content:c,timestamp:new Date().toISOString()},true);
        input.value = '';
    };
    btn.onclick = send;
    input.onkeydown = e => { if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();send();} };
}

/* ── Mobile Nav ── */
function initMobileNav() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.onclick = () => {
            document.querySelectorAll('.nav-btn').forEach(b=>{b.classList.remove('active');b.classList.add('text-slate-500');});
            btn.classList.add('active'); btn.classList.remove('text-slate-500');
            ['devices','transfer','chat'].forEach(p => {
                document.getElementById('panel-'+p).classList.toggle('hidden', p!==btn.dataset.tab);
            });
        };
    });
}

/* ── Toast ── */
function toast(msg, type='info') {
    const c = document.getElementById('toast-container');
    const icons = {success:'ri-checkbox-circle-fill text-emerald-400',error:'ri-close-circle-fill text-red-400',info:'ri-information-fill text-sky-400'};
    const bg = {success:'bg-emerald-500/10 border-emerald-500/20',error:'bg-red-500/10 border-red-500/20',info:'bg-sky-500/10 border-sky-500/20'};
    const id = 'toast-'+Date.now();
    c.insertAdjacentHTML('beforeend', `<div id="${id}" class="toast pointer-events-auto glass ${bg[type]} border rounded-xl px-4 py-3 flex items-center gap-3 shadow-2xl min-w-[260px] max-w-[360px]"><i class="${icons[type]} text-xl flex-shrink-0"></i><span class="text-sm text-slate-200 flex-1">${esc(msg)}</span><button onclick="document.getElementById('${id}').remove()" class="text-slate-500 hover:text-white transition-colors"><i class="ri-close-line"></i></button></div>`);
    setTimeout(()=>{const t=document.getElementById(id);if(t){t.classList.add('toast-exit');setTimeout(()=>t.remove(),300);}},3500);
}

function esc(s) { const d=document.createElement('div'); d.textContent=s; return d.innerHTML; }

/* ── QR Code ── */
function openQRCode() {
    document.getElementById('qrcode-modal').classList.remove('hidden');
    document.getElementById('qrcode-url').textContent = window.location.href;
}

function closeQRCode() {
    document.getElementById('qrcode-modal').classList.add('hidden');
}

function copyURL() {
    navigator.clipboard.writeText(window.location.href).then(() => {
        toast('链接已复制', 'success');
    }).catch(() => {
        const input = document.createElement('input');
        input.value = window.location.href;
        document.body.appendChild(input);
        input.select();
        document.execCommand('copy');
        document.body.removeChild(input);
        toast('链接已复制', 'success');
    });
}

/* ── Settings ── */
let currentSettings = {};

async function loadSettings() {
    try {
        const r = await fetch(`${API}/api/settings`);
        currentSettings = await r.json();
    } catch { currentSettings = {}; }
}

function openSettings() {
    loadSettings().then(() => {
        document.getElementById('setting-device-name').value = currentSettings.device_name || '';
        document.getElementById('setting-http-port').value = currentSettings.http_port || 8080;
        document.getElementById('setting-encryption').checked = currentSettings.encryption !== false;
        document.getElementById('setting-clipboard').checked = currentSettings.clipboard_sync !== false;
        renderShares(currentSettings.shares || []);
        document.getElementById('settings-modal').classList.remove('hidden');
    });
}

function closeSettings() {
    document.getElementById('settings-modal').classList.add('hidden');
}

function renderShares(shares) {
    const el = document.getElementById('shares-list');
    if (!shares.length) {
        el.innerHTML = '<p class="text-xs text-slate-500 text-center py-3">暂无共享文件夹</p>';
        return;
    }
    el.innerHTML = shares.map((s, i) => `
        <div class="flex items-center gap-3 p-3 rounded-xl bg-white/5">
            <i class="ri-folder-3-fill text-amber-400 text-lg"></i>
            <div class="flex-1 min-w-0">
                <p class="text-sm text-white truncate">${esc(s.name)}</p>
                <p class="text-[11px] text-slate-500 truncate">${esc(s.path)}</p>
            </div>
            <button onclick="removeShare(${i})" class="w-7 h-7 rounded-lg bg-red-500/10 flex items-center justify-center text-red-400 hover:bg-red-500/20 transition-all">
                <i class="ri-delete-bin-line text-sm"></i>
            </button>
        </div>
    `).join('');
}

function addShare() {
    const name = prompt('共享名称:');
    if (!name) return;
    const path = prompt('文件夹路径:');
    if (!path) return;
    if (!currentSettings.shares) currentSettings.shares = [];
    currentSettings.shares.push({ name, path, readonly: false });
    renderShares(currentSettings.shares);
}

function removeShare(index) {
    currentSettings.shares.splice(index, 1);
    renderShares(currentSettings.shares);
}

async function saveSettings() {
    const settings = {
        device_name: document.getElementById('setting-device-name').value || 'My PC',
        http_port: parseInt(document.getElementById('setting-http-port').value) || 8080,
        encryption: document.getElementById('setting-encryption').checked,
        clipboard_sync: document.getElementById('setting-clipboard').checked,
        shares: currentSettings.shares || []
    };
    try {
        const r = await fetch(`${API}/api/settings`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(settings)
        });
        if (r.ok) {
            toast('设置已保存', 'success');
            closeSettings();
        } else {
            toast('保存失败', 'error');
        }
    } catch {
        toast('保存出错', 'error');
    }
}

setInterval(loadDevices, 5000);
