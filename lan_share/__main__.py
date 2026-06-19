import argparse
import os
import sys
from pathlib import Path

from .server import FileServer


def get_default_share_dir() -> Path:
    return Path(__file__).resolve().parent / "shared"


def main():
    default_dir = get_default_share_dir()
    parser = argparse.ArgumentParser(
        description="LAN Share - Peer-to-peer file sharing over local network",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  lan-share                       # Use ./shared/ folder (auto-created)
  lan-share ./docs --port 9000    # Share ./docs on port 9000
  lan-share D:\\Share --port 8080  # Share a specific directory
        """,
    )
    parser.add_argument(
        "directory", nargs="?", default=str(default_dir),
        help=f"Directory to share (default: {default_dir})"
    )
    parser.add_argument(
        "-p", "--port", type=int, default=8000,
        help="Port to listen on (default: 8000)"
    )
    args = parser.parse_args()

    share_dir = Path(args.directory).resolve()
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
