@echo off
rem Builds build\CombineBaler.exe. Needs Go 1.24+ (https://go.dev/dl/), nothing else.
setlocal
cd /d "%~dp0"
where go >nul 2>nul
if errorlevel 1 (
  echo Go is not installed. Get it from https://go.dev/dl/
  exit /b 1
)
go vet . || exit /b 1
if /i not "%1"=="notest" (
  go test -count=1 . || exit /b 1
)
if not exist build mkdir build
go build -trimpath -ldflags "-H=windowsgui -s -w" -o build\CombineBaler.exe . || exit /b 1
echo.
echo Built build\CombineBaler.exe
