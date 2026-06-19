import asyncio
import io
import json
import mimetypes
import os
import time
import zipfile
from pathlib import Path
from urllib.parse import unquote, urlparse

from discovery import Discovery


class FileServer:
    def __init__(self, share_dir: str | Path, port: int = 8000):
        self.share_dir = Path(share_dir).resolve()
        self.share_dir.mkdir(parents=True, exist_ok=True)
        self.port = port
        self.host = self._get_local_ip()
        self.discovery = Discovery(port=self.port, host=self.host)
        self._start_time = time.time()
        self._upload_count = 0
        self._download_count = 0
        self._delete_count = 0

    def _get_local_ip(self) -> str:
        import socket
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        try:
            s.connect(("10.255.255.255", 1))
            ip = s.getsockname()[0]
        except Exception:
            ip = "127.0.0.1"
        finally:
            s.close()
        return ip

    def _get_file_size(self, path: Path) -> str:
        if path.is_dir():
            total = 0
            for root, dirs, files in os.walk(path):
                for f in files:
                    try:
                        total += (Path(root) / f).stat().st_size
                    except OSError:
                        pass
            size = total
        else:
            size = path.stat().st_size
        for unit in ("B", "KB", "MB", "GB", "TB"):
            if size < 1024:
                return f"{size:.1f} {unit}"
            size /= 1024
        return f"{size:.1f} PB"

    def _build_file_list(self) -> list[dict]:
        files = []
        for f in sorted(self.share_dir.iterdir(), key=lambda p: (not p.is_dir(), p.name.lower())):
            if f.name.startswith("."):
                continue
            stat = f.stat()
            files.append({
                "name": f.name,
                "size": self._get_file_size(f),
                "size_bytes": stat.st_size if not f.is_dir() else 0,
                "is_dir": f.is_dir(),
                "modified": time.strftime(
                    "%Y-%m-%d %H:%M:%S", time.localtime(stat.st_mtime)
                ),
            })
        return files

    async def handle_request(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
        try:
            request_data = await asyncio.wait_for(reader.read(65536), timeout=30)
        except asyncio.TimeoutError:
            writer.close()
            return

        # Split headers from body at \r\n\r\n
        header_end = request_data.find(b"\r\n\r\n")
        if header_end == -1:
            writer.close()
            return

        header_part = request_data[:header_end].decode("utf-8", errors="replace")
        body_data = request_data[header_end + 4:]

        lines = header_part.split("\r\n")
        if not lines:
            writer.close()
            return

        method, path, _ = lines[0].split(" ", 2) if len(lines[0].split(" ", 2)) == 3 else ("GET", "/", "HTTP/1.1")
        path = unquote(path)
        parsed = urlparse(path)
        clean_path = parsed.path

        headers = {}
        for line in lines[1:]:
            if ": " in line:
                k, v = line.split(": ", 1)
                headers[k.lower()] = v

        if method == "GET" and clean_path == "/":
            await self._serve_index(writer)
        elif method == "GET" and clean_path == "/api/files":
            await self._serve_json(writer, {
                "files": self._build_file_list(),
                "host": self.host,
                "port": self.port,
                "share_name": self.share_dir.name,
                "uptime": int(time.time() - self._start_time),
                "upload_count": self._upload_count,
                "download_count": self._download_count,
                "delete_count": self._delete_count,
            })
        elif method == "GET" and clean_path == "/api/peers":
            await self._serve_json(writer, {"peers": self.discovery.get_peers()})
        elif method == "GET" and clean_path.startswith("/download/"):
            filename = clean_path[len("/download/"):]
            await self._serve_file(writer, filename)
        elif method == "POST" and clean_path == "/upload":
            await self._handle_upload(writer, body_data, headers)
        elif method == "DELETE" and clean_path.startswith("/api/files/"):
            filename = clean_path[len("/api/files/"):]
            await self._handle_delete(writer, filename)
        elif method == "GET" and clean_path.startswith("/static/"):
            await self._serve_static(writer, clean_path[len("/static/"):])
        else:
            await self._serve_error(writer, 404, "Not Found")

    async def _serve_index(self, writer: asyncio.StreamWriter):
        body = INDEX_HTML.encode("utf-8")
        await self._send_response(writer, 200, "text/html; charset=utf-8", body)

    async def _serve_json(self, writer: asyncio.StreamWriter, data: dict):
        body = json.dumps(data, ensure_ascii=False).encode("utf-8")
        await self._send_response(writer, 200, "application/json", body)

    async def _serve_static(self, writer: asyncio.StreamWriter, filename: str):
        pkg_dir = Path(__file__).parent
        static_path = pkg_dir / "static" / filename
        if not static_path.exists() or not static_path.is_file():
            await self._serve_error(writer, 404, "Not Found")
            return
        content_type, _ = mimetypes.guess_type(str(static_path))
        content_type = content_type or "application/octet-stream"
        body = static_path.read_bytes()
        await self._send_response(writer, 200, content_type, body)

    async def _serve_file(self, writer: asyncio.StreamWriter, filename: str):
        safe_name = Path(filename).name
        file_path = self.share_dir / safe_name
        if not file_path.exists():
            await self._serve_error(writer, 404, "File not found")
            return

        self._download_count += 1

        if file_path.is_dir():
            await self._serve_directory_zip(writer, file_path, safe_name)
            return

        content_type, _ = mimetypes.guess_type(str(file_path))
        content_type = content_type or "application/octet-stream"
        file_size = file_path.stat().st_size

        response_headers = (
            f"HTTP/1.1 200 OK\r\n"
            f"Content-Type: {content_type}\r\n"
            f"Content-Length: {file_size}\r\n"
            f"Content-Disposition: attachment; filename=\"{safe_name}\"\r\n"
            f"Access-Control-Allow-Origin: *\r\n"
            f"Connection: close\r\n"
            f"\r\n"
        ).encode("utf-8")
        writer.write(response_headers)
        await writer.drain()

        with open(file_path, "rb") as f:
            while chunk := f.read(65536):
                writer.write(chunk)
                await writer.drain()
        writer.close()

    async def _serve_directory_zip(self, writer: asyncio.StreamWriter, dir_path: Path, dir_name: str):
        buf = io.BytesIO()
        with zipfile.ZipFile(buf, 'w', zipfile.ZIP_DEFLATED) as zf:
            for root, dirs, files in os.walk(dir_path):
                for f in files:
                    full = Path(root) / f
                    arcname = full.relative_to(dir_path)
                    zf.write(full, arcname)
        buf.seek(0)
        zip_data = buf.getvalue()

        response_headers = (
            f"HTTP/1.1 200 OK\r\n"
            f"Content-Type: application/zip\r\n"
            f"Content-Length: {len(zip_data)}\r\n"
            f"Content-Disposition: attachment; filename=\"{dir_name}.zip\"\r\n"
            f"Access-Control-Allow-Origin: *\r\n"
            f"Connection: close\r\n"
            f"\r\n"
        ).encode("utf-8")
        writer.write(response_headers + zip_data)
        await writer.drain()
        writer.close()

    async def _handle_upload(self, writer: asyncio.StreamWriter, request_data: bytes, headers: dict):
        content_type = headers.get("content-type", "")
        if "multipart/form-data" not in content_type:
            await self._serve_json(writer, {"error": "Only multipart uploads supported"})
            return

        boundary = content_type.split("boundary=")[-1].strip()
        if not boundary:
            await self._serve_json(writer, {"error": "No boundary found"})
            return

        boundary_bytes = boundary.encode("utf-8")
        parts = request_data.split(b"--" + boundary_bytes)

        uploaded = []
        for part in parts:
            if b"Content-Disposition: form-data" not in part:
                continue
            header_end = part.find(b"\r\n\r\n")
            if header_end == -1:
                continue
            part_headers_raw = part[:header_end].decode("utf-8", errors="replace")
            part_body = part[header_end + 4:]
            part_body = part_body.rstrip(b"\r\n--")

            filename = None
            for line in part_headers_raw.split("\r\n"):
                if 'filename="' in line:
                    filename = line.split('filename="')[1].split('"')[0]
                    break

            if filename and part_body:
                safe_name = Path(filename).name
                dest = self.share_dir / safe_name
                # Avoid overwriting: append number if exists
                counter = 1
                orig = dest
                while dest.exists():
                    stem = orig.stem
                    suffix = orig.suffix
                    dest = orig.parent / f"{stem}_{counter}{suffix}"
                    counter += 1

                dest.write_bytes(part_body)
                self._upload_count += 1
                uploaded.append(dest.name)

        await self._serve_json(writer, {"uploaded": uploaded})

    async def _handle_delete(self, writer: asyncio.StreamWriter, filename: str):
        safe_name = Path(filename).name
        file_path = self.share_dir / safe_name
        if not file_path.exists():
            await self._serve_error(writer, 404, "File not found")
            return
        if file_path.is_dir():
            import shutil
            shutil.rmtree(file_path)
        else:
            file_path.unlink()
        self._delete_count += 1
        await self._serve_json(writer, {"deleted": safe_name})

    async def _serve_error(self, writer: asyncio.StreamWriter, code: int, message: str):
        body = json.dumps({"error": message}).encode("utf-8")
        await self._send_response(writer, code, "application/json", body, status_message=message)

    async def _send_response(
        self, writer: asyncio.StreamWriter, status: int,
        content_type: str, body: bytes, status_message: str | None = None
    ):
        reason = status_message or {200: "OK", 404: "Not Found", 500: "Internal Server Error"}.get(status, "Unknown")
        response = (
            f"HTTP/1.1 {status} {reason}\r\n"
            f"Content-Type: {content_type}\r\n"
            f"Content-Length: {len(body)}\r\n"
            f"Access-Control-Allow-Origin: *\r\n"
            f"Connection: close\r\n"
            f"\r\n"
        ).encode("utf-8") + body
        writer.write(response)
        await writer.drain()
        writer.close()

    async def start(self):
        server = await asyncio.start_server(self.handle_request, "0.0.0.0", self.port)
        addr = server.sockets[0].getsockname()
        print(f"  LAN Share running at:")
        print(f"  Local:   http://localhost:{addr[1]}")
        print(f"  Network: http://{self.host}:{addr[1]}")
        print(f"  Share:   {self.share_dir}")
        print()

        async with server:
            discovery_task = asyncio.create_task(self.discovery.start())
            try:
                await server.serve_forever()
            except asyncio.CancelledError:
                pass
            finally:
                discovery_task.cancel()
                try:
                    await discovery_task
                except asyncio.CancelledError:
                    pass

    def run(self):
        try:
            asyncio.run(self.start())
        except KeyboardInterrupt:
            print("\nShutting down...")


INDEX_HTML = r"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>LAN Share</title>
<link rel="stylesheet" href="/static/style.css">
</head>
<body>
<div class="container">
  <header>
    <div class="logo">
      <svg viewBox="0 0 48 48" width="36" height="36">
        <rect x="8" y="28" width="32" height="12" rx="3" fill="currentColor" opacity=".7"/>
        <rect x="4" y="24" width="40" height="5" rx="2" fill="currentColor" opacity=".5"/>
        <rect x="14" y="12" width="20" height="13" rx="2" fill="currentColor" opacity=".9"/>
        <rect x="20" y="8" width="8" height="5" rx="1.5" fill="currentColor"/>
      </svg>
      <h1>LAN Share</h1>
    </div>
    <div class="status-bar">
      <span class="status-dot" id="statusDot"></span>
      <span id="statusText">Running</span>
      <span class="sep">|</span>
      <span id="peerCount">0 peers</span>
    </div>
  </header>

  <div class="peer-bar" id="peerBar" style="display:none">
    <span class="peer-label">Peers on LAN:</span>
    <div class="peer-list" id="peerList"></div>
  </div>

  <div class="toolbar">
    <div class="share-info">
      <span class="label">Sharing:</span>
      <span id="shareName">-</span>
    </div>
    <div class="toolbar-actions">
      <button class="btn btn-sm" onclick="refreshFiles()" title="Refresh">
        <svg viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M17.65 6.35A7.96 7.96 0 0 0 12 4a8 8 0 0 0-8 8h2a6 6 0 0 1 6-6c1.66 0 3.14.66 4.24 1.76L17 10h5V5l-1.35 1.35zM6.35 17.65A7.96 7.96 0 0 0 12 20a8 8 0 0 0 8-8h-2a6 6 0 0 1-6 6c-1.66 0-3.14-.66-4.24-1.76L7 14H2v5l1.35-1.35z"/></svg>
      </button>
    </div>
  </div>

  <div class="drop-zone" id="dropZone">
    <div class="drop-content">
      <svg viewBox="0 0 24 24" width="48" height="48"><path fill="currentColor" d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z" opacity=".5"/></svg>
      <p>Drag & drop files here, or <a href="#" id="uploadLink">click to browse</a></p>
      <p class="drop-hint">Files are shared with everyone on your LAN</p>
    </div>
    <input type="file" id="fileInput" multiple style="display:none">
    <div class="upload-progress" id="uploadProgress" style="display:none">
      <div class="progress-bar"><div class="progress-fill" id="progressFill"></div></div>
      <span id="progressText">Uploading...</span>
    </div>
  </div>

  <div class="upload-queue" id="uploadQueue"></div>

  <div class="file-list-header">
    <span class="col-name">Name</span>
    <span class="col-size">Size</span>
    <span class="col-date">Modified</span>
    <span class="col-actions">Actions</span>
  </div>
  <div class="file-list" id="fileList">
    <div class="loading">Loading files...</div>
  </div>
</div>

<footer>
  <span id="statsInfo"></span>
</footer>

<script src="/static/app.js"></script>
</body>
</html>"""

