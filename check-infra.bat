@echo off
setlocal

echo ========================================
echo  Checking Middleware Status
echo ========================================
echo.

REM Check MongoDB
echo [MongoDB]
docker ps --filter "name=^mongo$" --filter "status=running" --format "{{.Names}}" | findstr "mongo" >nul 2>&1
if errorlevel 1 (
    docker ps -a --filter "name=^mongo$" --format "{{.Names}}" | findstr "mongo" >nul 2>&1
    if errorlevel 1 (
        echo   Status: Not exists
        echo   Creating MongoDB...
        docker compose -p mongo -f docker-compose.mongo.yml up -d
    ) else (
        echo   Status: Stopped
        echo   Starting MongoDB...
        docker start mongo
    )
    echo   [OK] MongoDB started
) else (
    echo   Status: Running (skipped)
)

echo.

REM Check Redis
echo [Redis]
docker ps --filter "name=^redis$" --filter "status=running" --format "{{.Names}}" | findstr "redis" >nul 2>&1
if errorlevel 1 (
    docker ps -a --filter "name=^redis$" --format "{{.Names}}" | findstr "redis" >nul 2>&1
    if errorlevel 1 (
        echo   Status: Not exists
        echo   Creating Redis...
        docker compose -p redis -f docker-compose.redis.yml up -d
    ) else (
        echo   Status: Stopped
        echo   Starting Redis...
        docker start redis
    )
    echo   [OK] Redis started
) else (
    echo   Status: Running (skipped)
)

echo.

REM Check NATS
echo [NATS]
docker ps --filter "name=^nats$" --filter "status=running" --format "{{.Names}}" | findstr "nats" >nul 2>&1
if errorlevel 1 (
    docker ps -a --filter "name=^nats$" --format "{{.Names}}" | findstr "nats" >nul 2>&1
    if errorlevel 1 (
        echo   Status: Not exists
        echo   Creating NATS...
        docker compose -p nats -f docker-compose.nats.yml up -d
    ) else (
        echo   Status: Stopped
        echo   Starting NATS...
        docker start nats
    )
    echo   [OK] NATS started
) else (
    echo   Status: Running (skipped)
)

echo.
echo ========================================
echo  All middleware checked
echo ========================================

endlocal
