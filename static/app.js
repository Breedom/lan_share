let state = { files: [], peers: [] };
const POLL_INTERVAL = 5000;
let pollTimer = null;

document.addEventListener('DOMContentLoaded', () => {
  const dropZone = document.getElementById('dropZone');
  const fileInput = document.getElementById('fileInput');
  const uploadLink = document.getElementById('uploadLink');

  uploadLink.addEventListener('click', (e) => {
    e.preventDefault();
    fileInput.click();
  });

  fileInput.addEventListener('change', () => {
    if (fileInput.files.length) {
      uploadFiles(fileInput.files);
      fileInput.value = '';
    }
  });

  dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
  });

  dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
  });

  dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    if (e.dataTransfer.files.length) {
      uploadFiles(e.dataTransfer.files);
    }
  });

  refreshFiles();
  pollTimer = setInterval(refreshFiles, POLL_INTERVAL);
});

async function refreshFiles() {
  try {
    const [filesRes, peersRes] = await Promise.all([
      fetch('/api/files'),
      fetch('/api/peers')
    ]);
    const filesData = await filesRes.json();
    const peersData = await peersRes.json();
    state.files = filesData.files || [];
    state.peers = peersData.peers || [];
    renderFiles(filesData);
    renderPeers();
    updateStats(filesData);
  } catch (err) {
    console.error('Refresh failed:', err);
  }
}

function renderFiles(data) {
  const list = document.getElementById('fileList');
  const files = data.files || [];

  if (files.length === 0) {
    list.innerHTML = '<div class="empty">No files yet - drag something in!</div>';
    return;
  }

  document.getElementById('shareName').textContent = data.share_name || '-';

  list.innerHTML = files.map(f => {
    const isDir = f.is_dir;
    const icon = isDir
      ? '<span class="icon folder"><svg viewBox="0 0 24 24" width="18" height="18"><path fill="currentColor" d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg></span>'
      : '<span class="icon file"><svg viewBox="0 0 24 24" width="18" height="18"><path fill="currentColor" d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm-1 7V3.5L18.5 9H13z"/></svg></span>';

    const downloadBtn = `<button class="btn btn-sm" onclick="downloadFile('${f.name}')" title="Download"><svg viewBox="0 0 24 24" width="14" height="14"><path fill="currentColor" d="M13 5v6h3l-4 4-4-4h3V5h2zM5 17v2h14v-2H5z"/></svg></button>`;
    const copyBtn = `<button class="btn btn-sm" onclick="copyLink('${f.name}')" title="Copy link"><svg viewBox="0 0 24 24" width="14" height="14"><path fill="currentColor" d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/></svg></button>`;
    const delBtn = `<button class="btn btn-sm btn-danger" onclick="deleteFile('${f.name}')" title="Delete"><svg viewBox="0 0 24 24" width="14" height="14"><path fill="currentColor" d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zm2.46-7.12l1.41-1.41L12 12.59l2.12-2.12 1.41 1.41L13.41 14l2.12 2.12-1.41 1.41L12 15.41l-2.12 2.12-1.41-1.41L10.59 14l-2.13-2.12zM15.5 4l-1-1h-5l-1 1H5v2h14V4h-3.5z"/></svg></button>`;

    return `<div class="file-item">
      <div class="name">${icon}<span>${escapeHtml(f.name)}</span></div>
      <div class="size">${f.size}</div>
      <div class="date">${f.modified}</div>
      <div class="actions">${downloadBtn}${copyBtn}${delBtn}</div>
    </div>`;
  }).join('');
}

function renderPeers() {
  const bar = document.getElementById('peerBar');
  const list = document.getElementById('peerList');
  const countEl = document.getElementById('peerCount');

  if (state.peers.length === 0) {
    bar.style.display = 'none';
    countEl.textContent = '0 peers';
    return;
  }

  bar.style.display = 'flex';
  countEl.textContent = `${state.peers.length} peer${state.peers.length > 1 ? 's' : ''}`;

  list.innerHTML = state.peers.map(p => {
    const url = `http://${p.host}:${p.port}`;
    return `<a href="${url}" class="peer-chip" target="_blank">
      <span class="dot"></span>${escapeHtml(p.hostname)}
    </a>`;
  }).join('');
}

function updateStats(data) {
  const el = document.getElementById('statsInfo');
  const uptime = formatUptime(data.uptime || 0);
  el.textContent = `Uptime: ${uptime}  |  Upload: ${data.upload_count || 0}  |  Download: ${data.download_count || 0}  |  Deleted: ${data.delete_count || 0}`;
}

async function uploadFiles(files) {
  const queue = document.getElementById('uploadQueue');
  const progress = document.getElementById('uploadProgress');
  const fill = document.getElementById('progressFill');
  const text = document.getElementById('progressText');

  const items = [];
  for (const file of files) {
    const div = document.createElement('div');
    div.className = 'upload-item';
    div.innerHTML = `<span class="name">${escapeHtml(file.name)}</span><span class="status">queued</span>`;
    queue.appendChild(div);
    items.push({ file, el: div, statusEl: div.querySelector('.status') });
  }

  const formData = new FormData();
  for (const item of items) {
    formData.append('files', item.file);
  }

  progress.style.display = 'flex';
  text.textContent = `Uploading ${files.length} file${files.length > 1 ? 's' : ''}...`;

  try {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/upload', true);

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        const pct = Math.round((e.loaded / e.total) * 100);
        fill.style.width = pct + '%';
      }
    };

    await new Promise((resolve, reject) => {
      xhr.onload = () => {
        if (xhr.status === 200) resolve();
        else reject(new Error(`Upload failed: ${xhr.status}`));
      };
      xhr.onerror = () => reject(new Error('Upload failed'));
      xhr.send(formData);
    });

    for (const item of items) {
      item.statusEl.textContent = 'done';
      item.statusEl.className = 'status ok';
    }
  } catch (err) {
    for (const item of items) {
      item.statusEl.textContent = 'failed';
      item.statusEl.className = 'status err';
    }
  } finally {
    setTimeout(() => {
      progress.style.display = 'none';
      fill.style.width = '0%';
      setTimeout(() => { queue.innerHTML = ''; }, 2000);
    }, 1000);
    refreshFiles();
  }
}

function downloadFile(name) {
  window.location.href = '/download/' + encodeURIComponent(name);
}

function copyLink(name) {
  const url = window.location.origin + '/download/' + encodeURIComponent(name);

  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(url).then(
      () => showToast('Link copied: ' + url),
      () => fallbackCopy(url)
    );
  } else {
    fallbackCopy(url);
  }
}

function fallbackCopy(text) {
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand('copy');
    showToast('Link copied');
  } catch (_) {
    showToast('Copy not supported, select manually');
  }
  document.body.removeChild(ta);
}

function showToast(msg) {
  let toast = document.getElementById('toast');
  if (!toast) {
    toast = document.createElement('div');
    toast.id = 'toast';
    document.body.appendChild(toast);
  }
  toast.textContent = msg;
  toast.className = 'toast show';
  clearTimeout(toast._hide);
  toast._hide = setTimeout(() => { toast.className = 'toast'; }, 2500);
}

async function deleteFile(name) {
  if (!confirm(`Delete "${name}"?`)) return;

  try {
    const res = await fetch('/api/files/' + encodeURIComponent(name), { method: 'DELETE' });
    const data = await res.json();
    if (data.deleted) {
      showToast(`Deleted: ${data.deleted}`);
      refreshFiles();
    } else {
      showToast('Delete failed: ' + (data.error || 'unknown error'));
    }
  } catch (err) {
    showToast('Delete failed: ' + err.message);
  }
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function formatUptime(seconds) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}
