@echo off
REM ============================================
REM AI Gateway - Startup Script (Windows)
REM ============================================

setlocal EnableDelayedExpansion

REM Get script directory
set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."

REM Default values
set "WITH_MONITORING=false"
set "ACTION=start"

REM Print banner
echo.
echo   _    ___   __  ____  ____  ____  ____  ____
echo  / \  ^|_ ^| /  \/ ___^|^|  _ \/ ___^|^|  _ ^|^|  _ \
echo / _ \  ^| ^| / _ \___ \^| ^|_) \___ \^| ^|_) ^|^| ^|_) ^|
echo/ ___ \ ^| ^|/ ___ \__) ^|  __/ ___) ^|  __/^|  _ ^<
echo/_/   \_\___/_/   \_____^|_^|   ^|_____^|_^|   ^|_^| \_\
echo.
echo AI Gateway - One-Click Deployment
echo.

REM Parse arguments
:parse_args
if "%~1"=="" goto :end_parse
if /i "%~1"=="-m" set "WITH_MONITORING=true"
if /i "%~1"=="--monitoring" set "WITH_MONITORING=true"
if /i "%~1"=="-s" set "ACTION=stop"
if /i "%~1"=="--stop" set "ACTION=stop"
if /i "%~1"=="-r" set "ACTION=restart"
if /i "%~1"=="--restart" set "ACTION=restart"
if /i "%~1"=="-l" set "ACTION=logs"
if /i "%~1"=="--logs" set "ACTION=logs"
if /i "%~1"=="-h" goto :show_usage
if /i "%~1"=="--help" goto :show_usage
shift
goto :parse_args

:show_usage
echo Usage: %~nx0 [OPTIONS]
echo.
echo Options:
echo   -m, --monitoring    Start with monitoring stack
echo   -s, --stop          Stop all services
echo   -r, --restart       Restart all services
echo   -l, --logs          Show logs
echo   -h, --help          Show this help message
echo.
exit /b 0

:end_parse

REM Check Docker installation
echo [1/5] Checking Docker installation...
where docker >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not installed.
    echo Please install Docker Desktop from: https://docs.docker.com/docker-for-windows/
    exit /b 1
)

docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker daemon is not running.
    echo Please start Docker Desktop and try again.
    exit /b 1
)

echo [OK] Docker is installed and running

REM Check environment file
echo [2/5] Checking environment configuration...
if not exist "%PROJECT_DIR%\.env" (
    echo Creating .env file from template...
    if exist "%PROJECT_DIR%\.env.example" (
        copy "%PROJECT_DIR%\.env.example" "%PROJECT_DIR%\.env" >nul
        echo [OK] Created .env file. Please edit it to add your API keys.
    ) else (
        echo [WARN] .env.example not found, creating minimal .env
        (
            echo # AI Gateway Environment Configuration
            echo.
            echo # Server Ports
            echo GATEWAY_PORT=8000
            echo WEB_PORT=3000
            echo REDIS_PORT=6379
            echo.
            echo # API Keys (Configure these!)
            echo OPENAI_API_KEY=your-openai-api-key-here
            echo ANTHROPIC_API_KEY=your-anthropic-api-key-here
            echo AZURE_OPENAI_API_KEY=
            echo AZURE_OPENAI_ENDPOINT=
        ) > "%PROJECT_DIR%\.env"
        echo [OK] Created minimal .env file
    )
) else (
    echo [OK] .env file exists
)

REM Create necessary directories
echo [3/5] Creating necessary directories...
if not exist "%PROJECT_DIR%\data" mkdir "%PROJECT_DIR%\data"
if not exist "%PROJECT_DIR%\logs" mkdir "%PROJECT_DIR%\logs"
echo [OK] Directories created

REM Execute action
if "%ACTION%"=="stop" goto :stop_services
if "%ACTION%"=="restart" goto :restart_services
if "%ACTION%"=="logs" goto :show_logs
goto :start_services

:start_services
echo [4/5] Pulling Docker images...
cd /d "%PROJECT_DIR%"

if "%WITH_MONITORING%"=="true" (
    echo [5/5] Starting with monitoring stack...
    docker compose --profile monitoring pull
    docker compose --profile monitoring up -d --build
) else (
    echo [4/5] Pulling Docker images...
    docker compose pull
    echo [5/5] Starting basic services...
    docker compose up -d --build
)

echo.
echo =============================================
echo    AI Gateway Started Successfully!
echo =============================================
echo.
echo Services:
echo   Gateway API:    http://localhost:8000
echo   Web Dashboard:  http://localhost:3000
echo   Redis:          localhost:6379

if "%WITH_MONITORING%"=="true" (
    echo.
    echo Monitoring:
    echo   Prometheus:     http://localhost:9090
    echo   Grafana:        http://localhost:3001
    echo     User: admin
    echo     Pass: admin123
)

echo.
echo Commands:
echo   View logs:     %~nx0 --logs
echo   Stop services: %~nx0 --stop
echo.
goto :end

:stop_services
echo Stopping AI Gateway services...
cd /d "%PROJECT_DIR%"
docker compose --profile monitoring down
echo [OK] Services stopped
goto :end

:restart_services
call :stop_services
timeout /t 2 /nobreak >nul
call :start_services
goto :end

:show_logs
cd /d "%PROJECT_DIR%"
docker compose logs -f
goto :end

:end
endlocal
