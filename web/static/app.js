const API_BASE = window.location.origin;
let ws = null;
let selectedDevice = null;
let devices = [];

document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
    loadDevices();
    initUploadZone();
    loadChatHistory();
    initChatInput();
    initMobileNav();
    initToasts();
});

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onopen = () => {
        showToast('已连接到服务器', 'success');
        updateConnectionStatus(true);
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            handleWebSocketMessage(msg);
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    };

    ws.onclose = () => {
        updateConnectionStatus(false);
        setTimeout(initWebSocket, 3000);
    };

    ws.onerror = () => {
        updateConnectionStatus(false);
    };
}

function updateConnectionStatus(connected) {
    const badge = document.querySelector('.status-badge');
    const dot = document.querySelector('.status-dot');
    const text = document.getElementById('device-count');

    if (connected) {
        badge.style.background = 'rgba(16, 185, 129, 0.15)';
        badge.style.borderColor = 'rgba(16, 185, 129, 0.3)';
        dot.style.background = '#10b981';
        text.textContent = '已连接';
    } else {
        badge.style.background = 'rgba(245, 158, 11, 0.15)';
        badge.style.borderColor = 'rgba(245, 158, 11, 0.3)';
        dot.style.background = '#f59e0b';
        text.textContent = '重连中...';
    }
}

function handleWebSocketMessage(msg) {
    switch (msg.type) {
        case 'chat':
            appendChatMessage(msg, false);
            break;
        case 'transfer':
            updateTransferProgress(msg);
            break;
        case 'device_found':
            showToast(`发现新设备: ${msg.name}`, 'info');
            loadDevices();
            break;
        case 'device_lost':
            showToast(`设备离线: ${msg.name}`, 'info');
            loadDevices();
            break;
    }
}

async function loadDevices() {
    try {
        const response = await fetch(`${API_BASE}/api/peers`);
        devices = await response.json();
        renderDevices(devices);
        updatePeersCount(devices.length);
    } catch (error) {
        console.error('Failed to load devices:', error);
        renderDevices([]);
    }
}

function updatePeersCount(count) {
    document.getElementById('peers-count').textContent = count;
}

function renderDevices(deviceList) {
    const container = document.getElementById('device-list');

    if (deviceList.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🔍</div>
                <div class="empty-title">搜索设备中...</div>
                <div class="empty-desc">确保其他设备已开启 LanShare</div>
            </div>
        `;
        return;
    }

    container.innerHTML = deviceList.map(device => `
        <div class="device-item ${selectedDevice === device.id ? 'active' : ''}" 
             data-id="${device.id}" onclick="selectDevice('${device.id}')">
            <div class="device-avatar">${getDeviceIcon(device.type)}</div>
            <div class="device-details">
                <div class="device-name">${escapeHtml(device.name)}</div>
                <div class="device-meta">
                    <span class="device-status-indicator"></span>
                    ${device.ip}
                </div>
            </div>
        </div>
    `).join('');
}

function getDeviceIcon(type) {
    const icons = {
        'windows': '💻',
        'android': '📱',
        'linux': '🐧',
        'macos': '🍎'
    };
    return icons[type] || '💻';
}

function selectDevice(deviceId) {
    selectedDevice = deviceId;

    document.querySelectorAll('.device-item').forEach(el => {
        el.classList.remove('active');
    });

    const selected = document.querySelector(`[data-id="${deviceId}"]`);
    if (selected) {
        selected.classList.add('active');
    }

    const sendBtn = document.getElementById('send-btn');
    sendBtn.disabled = false;

    const device = devices.find(d => d.id === deviceId);
    if (device) {
        showToast(`已选择: ${device.name}`, 'info');
    }
}

function initUploadZone() {
    const zone = document.getElementById('upload-zone');
    const fileInput = document.getElementById('file-input');
    const uploadBtn = document.getElementById('upload-btn');

    uploadBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        fileInput.click();
    });

    zone.addEventListener('click', () => fileInput.click());

    zone.addEventListener('dragover', (e) => {
        e.preventDefault();
        zone.classList.add('dragover');
    });

    zone.addEventListener('dragleave', () => {
        zone.classList.remove('dragover');
    });

    zone.addEventListener('drop', (e) => {
        e.preventDefault();
        zone.classList.remove('dragover');
        handleFiles(e.dataTransfer.files);
    });

    fileInput.addEventListener('change', (e) => {
        handleFiles(e.target.files);
        e.target.value = '';
    });
}

async function handleFiles(files) {
    if (!selectedDevice) {
        showToast('请先选择目标设备', 'error');
        return;
    }

    if (files.length === 0) return;

    showToast(`准备发送 ${files.length} 个文件`, 'info');

    for (const file of files) {
        await uploadFile(file);
    }
}

async function uploadFile(file) {
    const formData = new FormData();
    formData.append('file', file);

    const transferId = `transfer-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    addTransferItem(transferId, file.name, file.size, 'uploading');

    try {
        const response = await fetch(`${API_BASE}/api/upload/Downloads`, {
            method: 'POST',
            body: formData
        });

        if (response.ok) {
            updateTransferStatus(transferId, 'success', 100, '完成');
            showToast(`${file.name} 发送成功`, 'success');
        } else {
            updateTransferStatus(transferId, 'error', 0, '失败');
            showToast(`${file.name} 发送失败`, 'error');
        }
    } catch (error) {
        updateTransferStatus(transferId, 'error', 0, '错误');
        showToast(`${file.name} 发送出错`, 'error');
    }
}

function addTransferItem(id, name, size, status) {
    const container = document.getElementById('transfer-list');
    const sizeStr = formatSize(size);
    const icon = getFileIcon(name);

    const html = `
        <div class="transfer-item" id="${id}">
            <div class="transfer-file-icon">${icon}</div>
            <div class="transfer-details">
                <div class="transfer-name">${escapeHtml(name)}</div>
                <div class="transfer-progress">
                    <div class="transfer-progress-bar" style="width: 0%"></div>
                </div>
                <div class="transfer-meta">
                    <span class="transfer-status ${status}">等待中...</span>
                    <span>${sizeStr}</span>
                </div>
            </div>
        </div>
    `;

    container.insertAdjacentHTML('afterbegin', html);
}

function updateTransferStatus(id, status, progress, text) {
    const item = document.getElementById(id);
    if (!item) return;

    const progressBar = item.querySelector('.transfer-progress-bar');
    const statusEl = item.querySelector('.transfer-status');

    progressBar.style.width = `${progress}%`;
    statusEl.className = `transfer-status ${status}`;
    statusEl.textContent = text;
}

function getFileIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const icons = {
        'jpg': '🖼️', 'jpeg': '🖼️', 'png': '🖼️', 'gif': '🖼️', 'svg': '🖼️',
        'mp4': '🎬', 'avi': '🎬', 'mov': '🎬', 'mkv': '🎬',
        'mp3': '🎵', 'wav': '🎵', 'flac': '🎵',
        'pdf': '📄', 'doc': '📄', 'docx': '📄', 'txt': '📄',
        'xls': '📊', 'xlsx': '📊', 'csv': '📊',
        'zip': '📦', 'rar': '📦', '7z': '📦',
        'exe': '⚙️', 'dmg': '⚙️',
    };
    return icons[ext] || '📄';
}

function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

async function loadChatHistory() {
    try {
        const response = await fetch(`${API_BASE}/api/chat/history`);
        const messages = await response.json();
        messages.forEach(msg => appendChatMessage(msg, msg.from === 'local'));
    } catch (error) {
        console.error('Failed to load chat history:', error);
    }
}

function appendChatMessage(msg, isSent) {
    const container = document.getElementById('chat-messages');
    const time = new Date(msg.timestamp).toLocaleTimeString('zh-CN', { 
        hour: '2-digit', 
        minute: '2-digit' 
    });

    const html = `
        <div class="message ${isSent ? 'sent' : 'received'}">
            <div class="message-bubble">${escapeHtml(msg.content)}</div>
            <div class="message-info">
                <span class="message-sender">${isSent ? '我' : escapeHtml(msg.from_name || '未知')}</span>
                <span class="message-time">${time}</span>
            </div>
        </div>
    `;

    container.insertAdjacentHTML('beforeend', html);
    container.scrollTop = container.scrollHeight;
}

function initChatInput() {
    const input = document.getElementById('message-input');
    const btn = document.getElementById('send-btn');

    const sendMessage = () => {
        const content = input.value.trim();
        if (!content || !selectedDevice) return;

        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'chat',
                payload: {
                    to: selectedDevice,
                    content: content
                }
            }));

            appendChatMessage({
                from: 'local',
                from_name: '我',
                content: content,
                timestamp: new Date().toISOString()
            }, true);

            input.value = '';
        } else {
            showToast('未连接到服务器', 'error');
        }
    };

    btn.addEventListener('click', sendMessage);
    input.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    input.addEventListener('input', () => {
        btn.disabled = !input.value.trim() || !selectedDevice;
    });
}

function initMobileNav() {
    const navItems = document.querySelectorAll('.nav-item');

    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const tab = item.dataset.tab;

            navItems.forEach(n => n.classList.remove('active'));
            item.classList.add('active');

            document.querySelectorAll('.card').forEach(card => {
                card.classList.add('hidden');
            });

            const targetCard = document.getElementById(`${tab}-card`);
            if (targetCard) {
                targetCard.classList.remove('hidden');
            }
        });
    });
}

function initToasts() {
    window.showToast = showToast;
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    const id = `toast-${Date.now()}`;

    const icons = {
        success: '✓',
        error: '✕',
        info: 'ℹ'
    };

    const html = `
        <div class="toast ${type}" id="${id}">
            <span class="toast-icon">${icons[type]}</span>
            <span class="toast-message">${escapeHtml(message)}</span>
            <button class="toast-close" onclick="removeToast('${id}')">✕</button>
        </div>
    `;

    container.insertAdjacentHTML('beforeend', html);

    setTimeout(() => removeToast(id), 4000);
}

function removeToast(id) {
    const toast = document.getElementById(id);
    if (toast) {
        toast.style.animation = 'toastIn 0.3s ease reverse';
        setTimeout(() => toast.remove(), 300);
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

setInterval(loadDevices, 5000);
