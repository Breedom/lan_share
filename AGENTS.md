# LAN Share — Agent Guide

## What This Is

Zero-config LAN file sharing tool. Python stdlib only, no dependencies. Async HTTP server + UDP broadcast discovery. Includes a tkinter GUI management panel and can be packaged as a single-file exe.

## Run

```powershell
# CLI mode
python __main__.py                     # shares ./shared_data/, port 8000
python __main__.py D:\Share --port 9000
.\lan_share.ps1 -Directory .\shared_data -Port 8000

# GUI mode
python __main__.py --gui               # launch management panel
python gui.py                          # same thing

# Exe (no Python required)
.\dist\LanShare.exe                    # auto-launches GUI
```

No install step. No virtualenv. Just `python __main__.py` from repo root.

## Architecture

Source files live at repo root (no package subdirectory). All imports are absolute:

```python
from server import FileServer    # __main__.py, gui.py
from discovery import Discovery  # server.py
```

- `__main__.py` — CLI entry, argparse, `--gui` flag, creates `FileServer`
- `gui.py` — tkinter management panel (logs, peers, files, start/stop)
- `server.py` — Async HTTP server (`FileServer`), all routes, inline `INDEX_HTML`
- `discovery.py` — UDP broadcast peer discovery (port 42069, 5s interval, 15s timeout)
- `static/app.js` — Frontend logic (SVG icons, folder nav, upload/download/delete)
- `static/style.css` — Dark theme CSS
- `lan_share.ps1` — PowerShell launcher
- `build_exe.spec` — PyInstaller spec for single-file exe
- `build_exe.py` — Build script (`python build_exe.py`)

`shared_data/` is the default share directory. Auto-created on first run. **Gitignored** — never commit files into it.

## Key Quirks

- HTML is embedded in `server.py` as `INDEX_HTML` (not a separate file). Edit the raw string to change the page.
- Path traversal guard: download and delete check `str(file_path).startswith(str(self.share_dir))`.
- Duplicate filenames get `_1`, `_2` suffixes on upload.
- Folders download as in-memory zip (`io.BytesIO` + `zipfile`). Fine for LAN, not for huge trees.
- Hidden files (starting with `.`) are excluded from the file list.
- `static/` files served via `FileServer._serve_static`, not by a framework.
- `get_resource_path()` in `server.py` handles both normal Python and PyInstaller bundle paths (`sys._MEIPASS`).
- PyInstaller exe is single-file, console=False (GUI only). Built with `python build_exe.py`. Output: `dist/LanShare.exe` (~11 MB).
- When running as PyInstaller bundle, `__main__.py` auto-launches GUI (no `--gui` flag needed).

## GUI Management Panel (`gui.py`)

- Directory picker, port input, start/stop buttons
- Three tabs: Logs (real-time stdout), Online Peers (polls `/api/peers`), Files (polls `/api/files`)
- Server runs in a daemon thread; stdout/stderr redirected via queue
- Dark theme matching the web UI

## API Routes

| Method | Path | Notes |
|--------|------|-------|
| GET | `/` | Serves `INDEX_HTML` |
| GET | `/api/files?path=subdir` | File list, supports subdirectory |
| GET | `/api/peers` | Discovered LAN peers |
| GET | `/download/<name>` | File or folder (zip) download |
| POST | `/upload?path=subdir` | Multipart file upload |
| DELETE | `/api/files/<name>` | Delete file or folder |
| GET | `/static/<name>` | Static assets |

## Git Conventions

- Commit messages in Chinese, descriptive style (e.g. `feat: ...`, `fix: ...`)
- Push to `https://github.com/Breedom/lan_share`
- Python 3.13, Windows environment

## Testing

No test suite exists. Manual verification: run the server (CLI or GUI), open browser, upload/download/delete files, check peer discovery on LAN.

## Exe Packaging

```powershell
pip install pyinstaller    # one-time install
python build_exe.py        # outputs dist/LanShare.exe (~11 MB)
```

PyInstaller spec excludes: `tkinter.test`, `unittest`, `test`, `pydoc`, `doctest`. Do NOT exclude `email`, `http`, `xml` — they are required by `urllib`.
