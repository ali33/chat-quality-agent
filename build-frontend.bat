@echo off
setlocal EnableExtensions
cd /d "%~dp0"

set "ROOT=%~dp0"
set "OUTDIR=%ROOT%build"

if not exist "%OUTDIR%" mkdir "%OUTDIR%"

where npm >nul 2>&1
if errorlevel 1 (
  echo [LOI] Khong tim thay npm trong PATH. Cai Node.js LTS tu https://nodejs.org/
  exit /b 1
)

echo [INFO] Build frontend (npm ci + vite build)...
pushd "%ROOT%frontend"
call npm ci
if errorlevel 1 (
  echo [LOI] npm ci that bai.
  popd
  exit /b 1
)
call npm run build
if errorlevel 1 (
  echo [LOI] npm run build that bai.
  popd
  exit /b 1
)
popd

if not exist "%ROOT%frontend\dist" (
  echo [LOI] Khong thay frontend\dist sau khi build.
  exit /b 1
)

if exist "%OUTDIR%\static" rmdir /s /q "%OUTDIR%\static"
mkdir "%OUTDIR%\static"
xcopy /E /I /Y "%ROOT%frontend\dist\*" "%OUTDIR%\static\" >nul

echo [OK] Da copy frontend dist -^> build\static
exit /b 0
