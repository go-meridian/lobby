@echo off
setlocal

if "%1"=="" goto help
if "%1"=="help" goto help
if "%1"=="build" goto build
if "%1"=="up" goto up
if "%1"=="down" goto down
if "%1"=="restart" goto restart
if "%1"=="logs" goto logs
if "%1"=="infra-up" goto infra-up
if "%1"=="infra-down" goto infra-down
if "%1"=="mongo-up" goto mongo-up
if "%1"=="mongo-down" goto mongo-down
if "%1"=="redis-up" goto redis-up
if "%1"=="redis-down" goto redis-down
if "%1"=="nats-up" goto nats-up
if "%1"=="nats-down" goto nats-down
if "%1"=="all-up" goto all-up
if "%1"=="local-up" goto local-up
if "%1"=="local-down" goto local-down
if "%1"=="check-infra" goto check-infra
if "%1"=="clean" goto clean
echo Unknown command: %1
goto help

:help
echo Available commands:
echo   make build       - Build Docker image
echo   make up          - Start Lobby service
echo   make down        - Stop Lobby service
echo   make restart     - Restart Lobby service
echo   make logs        - View logs
echo   make infra-up    - Start all middleware (Mongo/Redis/NATS)
echo   make infra-down  - Stop all middleware
echo   make mongo-up    - Start MongoDB
echo   make mongo-down  - Stop MongoDB
echo   make redis-up    - Start Redis
echo   make redis-down  - Stop Redis
echo   make nats-up     - Start NATS
echo   make nats-down   - Stop NATS
echo   make all-up      - Start everything (middleware + Lobby)
echo   make local-up    - Start Lobby only (connect to host middleware)
echo   make local-down  - Stop local Lobby
echo   make check-infra - Check and start middleware if needed
echo   make clean       - Clean all containers and networks
goto end

:build
docker compose build
goto end

:up
docker compose up -d
goto end

:down
docker compose down
goto end

:restart
docker compose restart
goto end

:logs
docker compose logs -f
goto end

:infra-up
docker compose -p mongo -f docker-compose.mongo.yml up -d
docker compose -p redis -f docker-compose.redis.yml up -d
docker compose -p nats -f docker-compose.nats.yml up -d
goto end

:infra-down
docker compose -p mongo -f docker-compose.mongo.yml down
docker compose -p redis -f docker-compose.redis.yml down
docker compose -p nats -f docker-compose.nats.yml down
goto end

:mongo-up
docker compose -p mongo -f docker-compose.mongo.yml up -d
goto end

:mongo-down
docker compose -p mongo -f docker-compose.mongo.yml down
goto end

:redis-up
docker compose -p redis -f docker-compose.redis.yml up -d
goto end

:redis-down
docker compose -p redis -f docker-compose.redis.yml down
goto end

:nats-up
docker compose -p nats -f docker-compose.nats.yml up -d
goto end

:nats-down
docker compose -p nats -f docker-compose.nats.yml down
goto end

:all-up
call :infra-up
call :up
goto end

:local-up
docker compose -f docker-compose.local.yml up -d --build
goto end

:local-down
docker compose -f docker-compose.local.yml down
goto end

:check-infra
call check-infra.bat
goto end

:clean
docker compose down -v --rmi local >nul 2>&1
docker compose -p mongo -f docker-compose.mongo.yml down -v >nul 2>&1
docker compose -p redis -f docker-compose.redis.yml down -v >nul 2>&1
docker compose -p nats -f docker-compose.nats.yml down >nul 2>&1
docker network rm mongo-net redis-net nats-net lobby-net >nul 2>&1
echo Cleanup completed
goto end

:end
