@echo off
REM Claude helper command
REM Usage: claude [command]

if "%1"=="" (
    set "COMMAND=help"
) else (
    set "COMMAND=%1"
)

powershell -ExecutionPolicy Bypass -File "%~dp0claude.ps1" %COMMAND%

