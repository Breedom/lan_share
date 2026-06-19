const API_BASE = window.location.origin;
let ws = null;
let selectedDevice = null;

document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
    loadDevices();
    initUploadArea();
    loadChatHistory();
    initChatInput();
});

function initWebSocket() {
    ws = new WebSocket(`ws://${window.location.host}/ws`);
    
    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        handleWebSocketMessage(msg);
    };
    
    ws.onclose = () => {
        setTimeout(initWebSocket, 3000);
    };
}

function handleWebSocketMessage(msg) {
    if (msg.type === 'chat') {
        appendChatMessage(msg, false);
    } else if (msg.type === 'transfer') {
        updateTransferProgress(msg);
    }
}

async function loadDevices() {
    try {
        const response = await fetch(`${API_BASE}/api/peers`);
        const devices = await response.json();
        renderDevices(devices);
    } catch (error) {
        console.error('Failed to load devices:', error);
        renderDevices([]);
    }
}

function renderDevices(devices) {
    const container = document.getElementById('device-list');
    
    if (devices.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>正在搜索局域网设备...</p>
                <p style="font-size: 0.9rem; margin-top: 8px;">请确保其他设备已开启 LanShare</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = devices.map(device => `
        <div class="device-item" data-id="${device.id}" onclick="selectDevice('${device.id}')">
            <div class="device-icon">${getDeviceIcon(device.type)}</div>
            <div class="device-info">
                <div class="device-name">${device.name}</div>
                <div class="device-status">${device.ip}</div>
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
        el.classList.remove('selected');
    });
    document.querySelector(`[data-id="${deviceId}"]`).classList.add('selected');
}

function initUploadArea() {
    const uploadArea = document.getElementById('upload-area');
    const fileInput = document.getElementById('file-input');
    
    uploadArea.addEventListener('click', () => fileInput.click());
    
    uploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadArea.classList.add('dragover');
    });
    
    uploadArea.addEventListener('dragleave', () => {
        uploadArea.classList.remove('dragover');
    });
    
    uploadArea.addEventListener('drop', (e) => {
        e.preventDefault();
        uploadArea.classList.remove('dragover');
        handleFiles(e.dataTransfer.files);
    });
    
    fileInput.addEventListener('change', (e) => {
        handleFiles(e.target.files);
    });
}

async function handleFiles(files) {
    if (!selectedDevice) {
        alert('请先选择目标设备');
        return;
    }
    
    for (const file of files) {
        await uploadFile(file);
    }
}

async function uploadFile(file) {
    const formData = new FormData();
    formData.append('file', file);
    
    const transferId = Date.now();
    addTransferItem(transferId, file.name, file.size);
    
    try {
        const response = await fetch(`${API_BASE}/api/upload/Downloads`, {
            method: 'POST',
            body: formData
        });
        
        if (response.ok) {
            updateTransferStatus(transferId, '完成', 100);
        } else {
            updateTransferStatus(transferId, '失败', 0);
        }
    } catch (error) {
        updateTransferStatus(transferId, '错误', 0);
    }
}

function addTransferItem(id, name, size) {
    const container = document.getElementById('transfer-list');
    const sizeStr = formatSize(size);
    
    const html = `
        <div class="transfer-item" id="transfer-${id}">
            <div class="transfer-icon">📄</div>
            <div class="transfer-info">
                <div class="transfer-name">${name}</div>
                <div class="transfer-progress">
                    <div class="transfer-progress-bar" style="width: 0%"></div>
                </div>
                <div class="transfer-status">等待中... (${sizeStr})</div>
            </div>
        </div>
    `;
    
    container.insertAdjacentHTML('afterbegin', html);
}

function updateTransferStatus(id, status, progress) {
    const item = document.getElementById(`transfer-${id}`);
    if (!item) return;
    
    const progressBar = item.querySelector('.transfer-progress-bar');
    const statusText = item.querySelector('.transfer-status');
    
    progressBar.style.width = `${progress}%`;
    statusText.textContent = status;
}

function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB';
    return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB';
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
    const time = new Date(msg.timestamp).toLocaleTimeString();
    
    const html = `
        <div class="message ${isSent ? 'sent' : 'received'}">
            <div class="message-bubble">${escapeHtml(msg.content)}</div>
            <div class="message-time">${msg.from_name || '未知'} ${time}</div>
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
    };
    
    btn.addEventListener('click', sendMessage);
    input.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') sendMessage();
    });
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

setInterval(loadDevices, 5000);
