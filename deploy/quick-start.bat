@echo off
REM ============================================
REM AI Gateway - Quick Start Script (Windows)
REM 一键启动 - 无需手动配置
REM ============================================

setlocal EnableDelayedExpansion

REM Get script directory
set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."

REM Print banner
cls
echo.
echo  ========================================
echo      AI Gateway - Quick Start
echo  ========================================
echo.

REM Step 1: Check Docker
echo [Step 1/5] Checking Docker...
where docker >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not installed!
    echo.
    echo Please install Docker Desktop:
    echo https://www.docker.com/products/docker-desktop
    echo.
    pause
    exit /b 1
)

docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not running!
    echo Please start Docker Desktop and try again.
    echo.
    pause
    exit /b 1
)
echo [OK] Docker is ready

REM Step 2: Setup environment
echo.
echo [Step 2/5] Setting up environment...
if not exist "%PROJECT_DIR%\.env" (
    if exist "%PROJECT_DIR%\.env.example" (
        copy "%PROJECT_DIR%\.env.example" "%PROJECT_DIR%\.env" >nul
        echo [OK] Created .env file
    ) else (
        (
            echo # AI Gateway Configuration
            echo GATEWAY_PORT=8000
            echo WEB_PORT=3000
            echo REDIS_PORT=6379
            echo.
            echo # API Keys - Please configure these!
            echo OPENAI_API_KEY=
            echo ANTHROPIC_API_KEY=
            echo AZURE_OPENAI_API_KEY=
            echo AZURE_OPENAI_ENDPOINT=
        ) > "%PROJECT_DIR%\.env"
        echo [OK] Created default .env file
    )
) else (
    echo [OK] .env file already exists
)

REM Step 3: Check API keys
echo.
echo [Step 3/5] Checking API keys...
findstr /C:"OPENAI_API_KEY=sk-" "%PROJECT_DIR%\.env" >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] OpenAI API key configured
    set "HAS_KEY=true"
) else (
    set "HAS_KEY=false"
)

if "%HAS_KEY%"=="false" (
    echo [WARNING] No API keys configured!
    echo.
    echo Please edit .env file and add your API keys:
    echo %PROJECT_DIR%\.env
    echo.
    echo Get your keys from:
    echo - OpenAI: https://platform.openai.com/api-keys
    echo - Anthropic: https://console.anthropic.com/settings/keys
    echo.
    choice /C YN /M "Continue without API keys"
    if errorlevel 2 exit /b 1
)

REM Step 4: Pull images
echo.
echo [Step 4/5] Pulling Docker images...
cd /d "%PROJECT_DIR%"
docker compose pull
if %errorlevel% neq 0 (
    echo [ERROR] Failed to pull images
    pause
    exit /b 1
)
echo [OK] Images pulled

REM Step 5: Start services
echo.
echo [Step 5/5] Starting services...
docker compose up -d --build
if %errorlevel% neq 0 (
    echo [ERROR] Failed to start services
    pause
    exit /b 1
)

REM Wait for services to be ready
echo.
echo Waiting for services to start...
timeout /t 5 /nobreak >nul

REM Check if gateway is healthy
curl -f http://localhost:8000/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Gateway is healthy
) else (
    echo [WARNING] Gateway may still be starting...
)

REM Success message
cls
echo.
echo  ========================================
echo      AI Gateway Started Successfully!
echo  ========================================
echo.
echo  Access Points:
echo.
echo    Gateway API:    http://localhost:8000
echo    Web Dashboard:  http://localhost:3000
echo    Health Check:   http://localhost:8000/health
echo.
echo  Quick Start Guide:
echo.
echo    1. Open http://localhost:3000 in your browser
echo    2. Configure your API keys in Settings
echo    3. Start making API requests!
echo.
echo  Management Commands:
echo.
echo    Stop:    scripts\start-gateway.bat --stop
echo    Logs:    scripts\start-gateway.bat --logs
echo    Restart: scripts\start-gateway.bat --restart
echo.
echo  ========================================
echo.

REM Open browser
choice /C YN /M "Open web dashboard in browser"
if errorlevel 1 (
    start http://localhost:3000
)

endlocal
pause
