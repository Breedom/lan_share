@echo off
echo [*] Building LanShare...

where rsrc >nul 2>nul
if %errorlevel% neq 0 (
    echo [*] Installing rsrc tool...
    go install github.com/akavel/rsrc@latest
)

echo [*] Generating manifest...
rsrc -manifest lanshare.manifest -o lanshare.syso

echo [*] Building executable...
go build -o lanshare.exe -ldflags "-H windowsgui" cmd/lanshare/main.go

echo [✓] Build complete: lanshare.exe
pause
