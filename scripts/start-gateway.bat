@echo off
setlocal EnableDelayedExpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
set "CONFIG_PATH=%CONFIG_PATH%"
if "%CONFIG_PATH%"=="" set "CONFIG_PATH=%PROJECT_DIR%\configs\config.json"
set "ACTION=start"
set "WITH_MONITORING=false"
set "SERVICES="

if not exist "%CONFIG_PATH%" (
  if exist "%PROJECT_DIR%\configs\config.example.json" (
    copy "%PROJECT_DIR%\configs\config.example.json" "%CONFIG_PATH%" >nul
  )
)

:parse_args
if "%~1"=="" goto :args_done
if /i "%~1"=="-m" set "WITH_MONITORING=true"
if /i "%~1"=="--monitoring" set "WITH_MONITORING=true"
if /i "%~1"=="-s" set "ACTION=stop"
if /i "%~1"=="--stop" set "ACTION=stop"
if /i "%~1"=="-r" set "ACTION=restart"
if /i "%~1"=="--restart" set "ACTION=restart"
if /i "%~1"=="-l" set "ACTION=logs"
if /i "%~1"=="--logs" set "ACTION=logs"
if /i "%~1"=="-h" goto :usage
if /i "%~1"=="--help" goto :usage
shift
goto :parse_args

:args_done
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "$p='%CONFIG_PATH%'; if (Test-Path $p) { $j=Get-Content $p -Raw | ConvertFrom-Json; if($j.edition.runtime){$j.edition.runtime}else{'docker'} } else {'docker'}"`) do set "EDITION_RUNTIME=%%i"
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "$p='%CONFIG_PATH%'; if (Test-Path $p) { $j=Get-Content $p -Raw | ConvertFrom-Json; if($j.edition.type){$j.edition.type}else{'standard'} } else {'standard'}"`) do set "EDITION_TYPE=%%i"

if /i "%EDITION_RUNTIME%"=="native" (
  if /i "%ACTION%"=="start" (
    echo runtime=native，不支持通过 start-gateway.bat 启动 Docker 入口。
    echo 请改用 scripts\dev-restart.sh，或在 /settings 中切换 runtime 为 docker。
    exit /b 1
  )
  if /i "%ACTION%"=="restart" (
    echo runtime=native，不支持通过 start-gateway.bat 启动 Docker 入口。
    echo 请改用 scripts\dev-restart.sh，或在 /settings 中切换 runtime 为 docker。
    exit /b 1
  )
)

where docker >nul 2>&1
if %errorlevel% neq 0 (
  echo Docker is not installed.
  exit /b 1
)

docker info >nul 2>&1
if %errorlevel% neq 0 (
  echo Docker daemon is not running.
  exit /b 1
)

set "SERVICES=gateway web redis"
if /i "%EDITION_TYPE%"=="standard" set "SERVICES=gateway web redis ollama"
if /i "%EDITION_TYPE%"=="enterprise" set "SERVICES=gateway web redis ollama qdrant"

if /i "%ACTION%"=="stop" goto :stop
if /i "%ACTION%"=="logs" goto :logs
if /i "%ACTION%"=="restart" goto :restart
goto :start

:start
if "%WITH_MONITORING%"=="true" (
  docker compose --profile monitoring up -d --build %SERVICES% prometheus grafana alertmanager
) else (
  docker compose up -d --build %SERVICES%
)
call :stop_extra
echo Services started with edition=%EDITION_TYPE% runtime=%EDITION_RUNTIME%
exit /b 0

:stop
docker compose down
echo Services stopped
exit /b 0

:restart
docker compose down
timeout /t 1 /nobreak >nul
goto :start

:logs
docker compose logs -f
exit /b 0

:stop_extra
if /i "%EDITION_TYPE%"=="basic" (
  docker compose stop ollama qdrant >nul 2>&1
)
if /i "%EDITION_TYPE%"=="standard" (
  docker compose stop qdrant >nul 2>&1
)
exit /b 0

:usage
echo Usage: %~nx0 [OPTIONS]
echo   -m, --monitoring    Start with monitoring stack
echo   -s, --stop          Stop services
echo   -r, --restart       Restart services
echo   -l, --logs          Show logs
echo   -h, --help          Show this help
exit /b 0
