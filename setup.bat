@echo off
setlocal enabledelayedexpansion

echo ================================================================
echo                     DevCost AI Setup
echo ================================================================
echo.

:: Check if Docker is installed
echo Checking Docker installation...
where docker >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not installed
    echo Please install Docker Desktop: https://docs.docker.com/desktop/windows/
    pause
    exit /b 1
)
echo [OK] Docker is installed

:: Check if Docker is running
echo Checking if Docker is running...
docker info >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not running
    echo Please start Docker Desktop
    pause
    exit /b 1
)
echo [OK] Docker is running

:: Check port availability
echo.
echo Checking port availability...

netstat -an | findstr ":3000 " >nul 2>nul
if %errorlevel% equ 0 (
    echo [WARN] Port 3000 ^(Frontend^) is in use
) else (
    echo [OK] Port 3000 ^(Frontend^) is available
)

netstat -an | findstr ":8080 " >nul 2>nul
if %errorlevel% equ 0 (
    echo [WARN] Port 8080 ^(Backend^) is in use
) else (
    echo [OK] Port 8080 ^(Backend^) is available
)

netstat -an | findstr ":5432 " >nul 2>nul
if %errorlevel% equ 0 (
    echo [WARN] Port 5432 ^(PostgreSQL^) is in use
) else (
    echo [OK] Port 5432 ^(PostgreSQL^) is available
)

:: Create .env file
echo.
echo Setting up environment configuration...

if exist .env (
    echo [WARN] .env file already exists
    set /p overwrite="Overwrite? (y/N): "
    if /i not "!overwrite!"=="y" (
        echo Keeping existing .env file
        goto :instructions
    )
)

copy .env.example .env >nul
echo [OK] Created .env file from .env.example

:instructions
echo.
echo ================================================================
echo                  Configuration Options
echo ================================================================
echo.
echo To configure DevCost AI for your environment:
echo   1. Edit .env file
echo   2. Add your AWS credentials:
echo      AWS_ACCESS_KEY_ID=your-key
echo      AWS_SECRET_ACCESS_KEY=your-secret
echo      AWS_REGION=us-east-1
echo.
echo Optional configurations:
echo   - Slack integration: Add SLACK_BOT_TOKEN and SLACK_SIGNING_SECRET
echo   - Email alerts: Add SMTP_* settings
echo   - AI Analysis: Set AI_ENABLED=true (requires Ollama)
echo.

:: Ask to start services
set /p startnow="Start DevCost AI now? (Y/n): "
if /i "!startnow!"=="n" (
    echo.
    echo To start later, run: docker compose up -d --build
    pause
    exit /b 0
)

:: Start services
echo.
echo ================================================================
echo                     Starting Services
echo ================================================================
echo.
echo Building and starting containers...
echo This may take a few minutes on first run...
echo.

docker compose up -d --build

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Failed to start services
    echo Check Docker logs: docker compose logs
    pause
    exit /b 1
)

echo.
echo Waiting for services to be healthy...
timeout /t 15 /nobreak >nul

docker compose ps

echo.
echo ================================================================
echo                    Setup Complete!
echo ================================================================
echo.
echo DevCost AI is now running!
echo.
echo   Frontend:  http://localhost:3000
echo   Backend:   http://localhost:8080
echo   API Docs:  http://localhost:8080/swagger/index.html
echo   Health:    http://localhost:8080/health
echo.
echo Useful commands:
echo   View logs:     docker compose logs -f
echo   Stop:          docker compose down
echo   Restart:       docker compose restart
echo   Rebuild:       docker compose up -d --build
echo.

pause
