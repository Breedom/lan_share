import argparse
import os
import sys
from pathlib import Path


def get_default_share_dir() -> Path:
    return Path(__file__).resolve().parent / "shared_data"


def main():
    default_dir = get_default_share_dir()
    parser = argparse.ArgumentParser(
        description="LAN Share - Peer-to-peer file sharing over local network",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python __main__.py                       # Use ./shared_data/ folder (auto-created)
  python __main__.py ./docs --port 9000    # Share ./docs on port 9000
  python __main__.py D:\\Share --port 8080  # Share a specific directory
  python __main__.py --gui                 # Launch GUI management panel
        """,
    )
    parser.add_argument(
        "directory", nargs="?", default=None,
        help=f"Directory to share (default: {default_dir})"
    )
    parser.add_argument(
        "-p", "--port", type=int, default=8000,
        help="Port to listen on (default: 8000)"
    )
    parser.add_argument(
        "--gui", action="store_true", default=False,
        help="Launch GUI management panel"
    )
    args = parser.parse_args()

    # In PyInstaller bundle, always launch GUI
    is_bundled = getattr(sys, 'frozen', False)
    if args.gui or is_bundled:
        from gui import main as gui_main
        gui_main()
        return

    from server import FileServer

    share_dir = Path(args.directory or default_dir).resolve()
    share_dir.mkdir(parents=True, exist_ok=True)

    print("=" * 50)
    print("  LAN Share")
    print("  Share files securely over your local network")
    print("=" * 50)
    print()

    server = FileServer(share_dir=share_dir, port=args.port)
    server.run()


if __name__ == "__main__":
    main()
