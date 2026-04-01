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

rem SQLite dung github.com/glebarez/sqlite (pure Go, khong can gcc). Tat CGO de build tren Windows khong MSYS2.
set "CGO_ENABLED=0"

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

set "STATIC_OUT=%OUTDIR%\static"
if exist "%ROOT%frontend\dist\index.html" (
  echo [INFO] Copy frontend dist -^> "%STATIC_OUT%"
  if not exist "%STATIC_OUT%" mkdir "%STATIC_OUT%"
  xcopy /E /I /Y "%ROOT%frontend\dist\*" "%STATIC_OUT%\" >nul
) else (
  echo [CANH BAO] Khong co "%ROOT%frontend\dist\index.html". Chay: cd frontend ^&^& npm run build
  echo            Sau do chay lai build-backend, hoac copy dist vao "%STATIC_OUT%" thu cong.
)

echo.
echo [XONG] File: %EXE%
echo        Chay production: APP_ENV=production — static nam canh exe: build\static
echo        Co the chay exe tu bat ky thu muc; hoac set STATIC_DIR tro toi thu muc chua index.html
exit /b 0
