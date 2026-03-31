@echo off
setlocal EnableExtensions
cd /d "%~dp0"

set "ROOT=%~dp0"
set "OUTDIR=%ROOT%build"
set "BACKEND=%ROOT%backend"
set "EXE=%OUTDIR%\cqa-server.exe"

set "VERSION=dev"
if defined CI_VERSION (
  set "VERSION=%CI_VERSION%"
) else (
  where powershell >nul 2>&1 && for /f "delims=" %%i in ('powershell -NoProfile -Command "Get-Date -Format yyyy.MM.dd"') do set "VERSION=%%i"
)

if not exist "%OUTDIR%" mkdir "%OUTDIR%"

where go >nul 2>&1
if errorlevel 1 (
  echo [LOI] Khong tim thay Go trong PATH. Cai dat tu https://go.dev/dl/ va mo lai CMD.
  exit /b 1
)

rem SQLite (mattn/go-sqlite3) can CGO: can gcc tren PATH (MSYS2 MinGW-w64, TDM-GCC, v.v.)
set "CGO_ENABLED=1"

echo [INFO] Build Go -^> "%EXE%"
pushd "%BACKEND%"
echo [INFO] go mod tidy...
go mod tidy
if errorlevel 1 (
  echo [LOI] go mod tidy that bai.
  popd
  exit /b 1
)
go build -trimpath -ldflags="-s -w -X main.version=%VERSION%" -o "%EXE%" .
set "ERR=%ERRORLEVEL%"
popd
if not "%ERR%"=="0" (
  echo.
  echo [LOI] go build that bai. Neu loi lien quan CGO/gcc:
  echo        - Cai MSYS2 + mingw-w64, them C:\msys64\mingw64\bin vao PATH
  echo        - Hoac dung MSYS2 UCRT64 / TDM-GCC co gcc.exe
  exit /b 1
)

echo.
echo [XONG] File: %EXE%
echo        Chay production: dat APP_ENV=production, cd /d build, chay cqa-server.exe
echo        (static phai o build\static neu da build frontend)
exit /b 0
