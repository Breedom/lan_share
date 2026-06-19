"""GUI management panel for LAN Share."""

import io
import json
import queue
import sys
import threading
import time
import tkinter as tk
from tkinter import filedialog, messagebox, ttk
from pathlib import Path
from urllib.request import urlopen, Request
from urllib.error import URLError


class LogRedirector(io.TextIOBase):
    """Redirect print() output to a queue for GUI consumption."""

    def __init__(self, log_queue: queue.Queue):
        self._queue = log_queue

    def write(self, text: str):
        if text and text.strip():
            self._queue.put(text.rstrip("\n"))
        return len(text) or 0

    def flush(self):
        pass


class LanShareGUI:
    def __init__(self):
        self.root = tk.Tk()
        self.root.title("LAN Share 管理面板")
        self.root.geometry("720x560")
        self.root.minsize(600, 450)

        self._server_thread: threading.Thread | None = None
        self._server_running = False
        self._server_instance = None
        self._log_queue: queue.Queue[str] = queue.Queue()
        self._start_time: float = 0
        self._peers: list[dict] = []
        self._files: list[dict] = []
        self._current_path: str = ""

        self._build_ui()
        self._poll_logs()
        self._poll_status()

    # ── UI Construction ──────────────────────────────────────────────

    def _build_ui(self):
        self.root.configure(bg="#1e1e2e")

        style = ttk.Style()
        style.theme_use("clam")
        style.configure("TFrame", background="#1e1e2e")
        style.configure("TLabel", background="#1e1e2e", foreground="#cdd6f4", font=("Segoe UI", 10))
        style.configure("Header.TLabel", background="#1e1e2e", foreground="#cba6f7", font=("Segoe UI", 13, "bold"))
        style.configure("Status.TLabel", background="#1e1e2e", foreground="#a6e3a1", font=("Segoe UI", 9))
        style.configure("TButton", font=("Segoe UI", 10))
        style.configure("Accent.TButton", foreground="#1e1e2e")
        style.configure("TEntry", fieldbackground="#313244", foreground="#cdd6f4", insertcolor="#cdd6f4")
        style.configure("TNotebook", background="#1e1e2e")
        style.configure("TNotebook.Tab", background="#313244", foreground="#cdd6f4", padding=[12, 4])
        style.map("TNotebook.Tab",
                   background=[("selected", "#45475a")],
                   foreground=[("selected", "#cba6f7")])

        # ── Top section: config + controls ──
        top = ttk.Frame(self.root)
        top.pack(fill="x", padx=16, pady=(12, 6))

        ttk.Label(top, text="LAN Share 管理面板", style="Header.TLabel").pack(anchor="w")

        cfg = ttk.Frame(top)
        cfg.pack(fill="x", pady=(8, 0))

        ttk.Label(cfg, text="共享目录:").grid(row=0, column=0, sticky="w", padx=(0, 6))
        self.dir_var = tk.StringVar(value=str(Path(__file__).resolve().parent / "shared_data"))
        self.dir_entry = ttk.Entry(cfg, textvariable=self.dir_var, width=50)
        self.dir_entry.grid(row=0, column=1, sticky="ew", padx=(0, 6))
        ttk.Button(cfg, text="浏览", width=6, command=self._browse_dir).grid(row=0, column=2)

        ttk.Label(cfg, text="端口:").grid(row=1, column=0, sticky="w", padx=(0, 6), pady=(6, 0))
        self.port_var = tk.StringVar(value="8000")
        port_entry = ttk.Entry(cfg, textvariable=self.port_var, width=10)
        port_entry.grid(row=1, column=1, sticky="w", pady=(6, 0))

        cfg.columnconfigure(1, weight=1)

        # ── Control buttons ──
        ctrl = ttk.Frame(top)
        ctrl.pack(fill="x", pady=(10, 0))

        self.start_btn = ttk.Button(ctrl, text="▶ 启动", command=self._start_server)
        self.start_btn.pack(side="left", padx=(0, 6))

        self.stop_btn = ttk.Button(ctrl, text="■ 停止", command=self._stop_server, state="disabled")
        self.stop_btn.pack(side="left", padx=(0, 12))

        self.status_var = tk.StringVar(value="○ 未启动")
        ttk.Label(ctrl, textvariable=self.status_var, style="Status.TLabel").pack(side="left")

        # ── Notebook: logs / peers / files ──
        nb_frame = ttk.Frame(self.root)
        nb_frame.pack(fill="both", expand=True, padx=16, pady=(6, 0))

        self.notebook = ttk.Notebook(nb_frame)
        self.notebook.pack(fill="both", expand=True)

        # Log tab
        log_frame = ttk.Frame(self.notebook)
        self.notebook.add(log_frame, text=" 日志 ")
        self.log_text = tk.Text(log_frame, bg="#313244", fg="#cdd6f4", insertbackground="#cdd6f4",
                                font=("Consolas", 9), wrap="word", state="disabled", borderwidth=0)
        log_scroll = ttk.Scrollbar(log_frame, command=self.log_text.yview)
        self.log_text.configure(yscrollcommand=log_scroll.set)
        log_scroll.pack(side="right", fill="y")
        self.log_text.pack(side="left", fill="both", expand=True)

        # Peers tab
        peers_frame = ttk.Frame(self.notebook)
        self.notebook.add(peers_frame, text=" 在线设备 ")
        self.peers_text = tk.Text(peers_frame, bg="#313244", fg="#cdd6f4",
                                  font=("Consolas", 9), wrap="word", state="disabled", borderwidth=0)
        self.peers_text.pack(fill="both", expand=True)

        # Files tab
        files_frame = ttk.Frame(self.notebook)
        self.notebook.add(files_frame, text=" 文件列表 ")
        files_top = ttk.Frame(files_frame)
        files_top.pack(fill="x", padx=4, pady=(4, 0))
        self.path_var = tk.StringVar(value="/")
        ttk.Label(files_top, text="路径:").pack(side="left")
        ttk.Label(files_top, textvariable=self.path_var, foreground="#89b4fa").pack(side="left", padx=(4, 8))
        ttk.Button(files_top, text="返回上级", command=self._go_up).pack(side="right")

        self.files_text = tk.Text(files_frame, bg="#313244", fg="#cdd6f4",
                                  font=("Consolas", 9), wrap="none", state="disabled", borderwidth=0)
        files_scroll = ttk.Scrollbar(files_frame, command=self.files_text.xview, orient="horizontal")
        self.files_text.configure(xscrollcommand=files_scroll.set)
        files_scroll.pack(side="bottom", fill="x")
        self.files_text.pack(fill="both", expand=True)

        # ── Status bar ──
        bar = ttk.Frame(self.root)
        bar.pack(fill="x", padx=16, pady=(4, 10))
        self.uptime_var = tk.StringVar(value="")
        ttk.Label(bar, textvariable=self.uptime_var, font=("Segoe UI", 9), foreground="#6c7086").pack(anchor="w")

    # ── Actions ──────────────────────────────────────────────────────

    def _browse_dir(self):
        d = filedialog.askdirectory(initialdir=self.dir_var.get())
        if d:
            self.dir_var.set(d)

    def _start_server(self):
        if self._server_running:
            return

        share_dir = self.dir_var.get()
        try:
            port = int(self.port_var.get())
        except ValueError:
            messagebox.showerror("错误", "端口必须是数字")
            return

        if not Path(share_dir).is_dir():
            messagebox.showerror("错误", f"目录不存在: {share_dir}")
            return

        self._server_running = True
        self._start_time = time.time()
        self.start_btn.configure(state="disabled")
        self.stop_btn.configure(state="normal")
        self.status_var.set("● 运行中")
        self._log(f"启动服务器 — 目录: {share_dir}, 端口: {port}")

        self._server_thread = threading.Thread(
            target=self._run_server, args=(share_dir, port), daemon=True
        )
        self._server_thread.start()

    def _stop_server(self):
        if not self._server_running:
            return
        self._server_running = False
        if self._server_instance:
            self._server_instance._force_stop = True
        self.start_btn.configure(state="normal")
        self.stop_btn.configure(state="disabled")
        self.status_var.set("○ 已停止")
        self._log("服务器已停止")

    def _run_server(self, share_dir: str, port: int):
        import asyncio
        from server import FileServer

        # Redirect stdout to queue
        old_stdout = sys.stdout
        old_stderr = sys.stderr
        sys.stdout = LogRedirector(self._log_queue)
        sys.stderr = LogRedirector(self._log_queue)

        try:
            server = FileServer(share_dir=share_dir, port=port)
            self._server_instance = server
            server._force_stop = False

            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

            async def _run():
                s = await asyncio.start_server(server.handle_request, "0.0.0.0", port)
                addr = s.sockets[0].getsockname()
                print(f"  LAN Share running at:")
                print(f"  Local:   http://localhost:{addr[1]}")
                print(f"  Network: http://{server.host}:{addr[1]}")
                print(f"  Share:   {server.share_dir}")
                print()

                discovery_task = asyncio.create_task(server.discovery.start())
                try:
                    while self._server_running and not getattr(server, '_force_stop', False):
                        await asyncio.sleep(0.5)
                finally:
                    discovery_task.cancel()
                    try:
                        await discovery_task
                    except asyncio.CancelledError:
                        pass
                    s.close()
                    await s.wait_closed()

            loop.run_until_complete(_run())
            loop.close()
        except Exception as e:
            print(f"服务器错误: {e}")
        finally:
            sys.stdout = old_stdout
            sys.stderr = old_stderr
            self._server_instance = None

    # ── Polling ──────────────────────────────────────────────────────

    def _poll_logs(self):
        while not self._log_queue.empty():
            try:
                msg = self._log_queue.get_nowait()
                self.log_text.configure(state="normal")
                self.log_text.insert("end", msg + "\n")
                self.log_text.see("end")
                self.log_text.configure(state="disabled")
            except queue.Empty:
                break
        self.root.after(100, self._poll_logs)

    def _poll_status(self):
        if self._server_running and self._start_time:
            elapsed = int(time.time() - self._start_time)
            h, m, s = elapsed // 3600, (elapsed % 3600) // 60, elapsed % 60
            self.uptime_var.set(f"已运行 {h:02d}:{m:02d}:{s:02d}")

            # Fetch peers
            self._fetch_peers()
            # Fetch files
            self._fetch_files()
        else:
            self.uptime_var.set("")

        self.root.after(3000, self._poll_status)

    def _fetch_peers(self):
        try:
            port = self.port_var.get()
            req = Request(f"http://localhost:{port}/api/peers")
            with urlopen(req, timeout=2) as resp:
                data = json.loads(resp.read())
                self._peers = data.get("peers", [])
                self._render_peers()
        except Exception:
            pass

    def _render_peers(self):
        self.peers_text.configure(state="normal")
        self.peers_text.delete("1.0", "end")
        if not self._peers:
            self.peers_text.insert("1.0", "暂无在线设备\n\n局域网中运行 LAN Share 的设备会自动显示在此处。")
        else:
            self.peers_text.insert("1.0", f"共 {len(self._peers)} 台设备在线\n\n")
            for i, p in enumerate(self._peers, 1):
                self.peers_text.insert("end", f"{i}. {p.get('hostname', '?')}\n")
                self.peers_text.insert("end", f"   地址: http://{p.get('host', '?')}:{p.get('port', '?')}\n\n")
        self.peers_text.configure(state="disabled")

    def _fetch_files(self):
        try:
            port = self.port_var.get()
            path_param = f"?path={self._current_path}" if self._current_path else ""
            req = Request(f"http://localhost:{port}/api/files{path_param}")
            with urlopen(req, timeout=2) as resp:
                data = json.loads(resp.read())
                self._files = data.get("files", [])
                self._render_files(data)
        except Exception:
            pass

    def _render_files(self, data: dict):
        self.files_text.configure(state="normal")
        self.files_text.delete("1.0", "end")
        self.path_var.set("/" + self._current_path if self._current_path else "/")

        files = data.get("files", [])
        if not files:
            self.files_text.insert("1.0", "目录为空")
        else:
            dirs = [f for f in files if f.get("is_dir")]
            regular = [f for f in files if not f.get("is_dir")]

            if dirs:
                self.files_text.insert("end", "📁 文件夹\n")
                for d in dirs:
                    self.files_text.insert("end", f"  📁 {d['name']}\n")
                self.files_text.insert("end", "\n")

            if regular:
                self.files_text.insert("end", "📄 文件\n")
                for f in regular:
                    self.files_text.insert("end", f"  {f['name']}  ({f.get('size', '?')})\n")
        self.files_text.configure(state="disabled")

    def _go_up(self):
        if not self._current_path:
            return
        parts = self._current_path.split("/")
        parts = parts[:-1]
        self._current_path = "/".join(p for p in parts if p)
        self._fetch_files()

    # ── Helpers ──────────────────────────────────────────────────────

    def _log(self, msg: str):
        self._log_queue.put(msg)

    def run(self):
        self.root.protocol("WM_DELETE_WINDOW", self._on_close)
        self.root.mainloop()

    def _on_close(self):
        if self._server_running:
            self._stop_server()
        self.root.destroy()


def main():
    app = LanShareGUI()
    app.run()


if __name__ == "__main__":
    main()
