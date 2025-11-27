@echo off
chcp 65001 >nul
echo ============================================
echo Starting Backend Server on Port 9999
echo ============================================
echo.

REM Set environment variables if needed
REM set ARLIAI_API_KEY=your_key_here

REM Run the server
echo Starting server...
go run cmd/server/main.go

echo.
echo ============================================
echo Server stopped
echo ============================================
pause
