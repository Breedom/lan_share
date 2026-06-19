"""Build script for LAN Share — creates single-file exe via PyInstaller."""

import subprocess
import sys
from pathlib import Path


def main():
    root = Path(__file__).resolve().parent
    spec_file = root / "build_exe.spec"

    if not spec_file.exists():
        print(f"Error: {spec_file} not found")
        sys.exit(1)

    print("=" * 50)
    print("  Building LAN Share exe")
    print("=" * 50)
    print()

    cmd = [
        sys.executable, "-m", "PyInstaller",
        "--clean",
        "--noconfirm",
        str(spec_file),
    ]

    print(f"Running: {' '.join(cmd)}")
    print()

    result = subprocess.run(cmd, cwd=str(root))

    if result.returncode == 0:
        out = root / "dist" / "LanShare.exe"
        if out.exists():
            size_mb = out.stat().st_size / (1024 * 1024)
            print()
            print("=" * 50)
            print(f"  Build succeeded!")
            print(f"  Output: {out}")
            print(f"  Size:   {size_mb:.1f} MB")
            print("=" * 50)
        else:
            print("Build completed but exe not found at expected path")
    else:
        print(f"Build failed with exit code {result.returncode}")
        sys.exit(result.returncode)


if __name__ == "__main__":
    main()
