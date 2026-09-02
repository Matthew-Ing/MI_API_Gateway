@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0go-overlay.ps1" %*
exit /b %ERRORLEVEL%
