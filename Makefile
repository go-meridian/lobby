.PHONY: help build up down restart logs clean infra-up infra-down network

# 默认目标
help:
	@echo "可用命令："
	@echo "  make network     - 创建共享网络（首次使用前执行）"
	@echo "  make build       - 构建 Docker 镜像"
	@echo "  make up          - 启动 Lobby 服务"
	@echo "  make down        - 停止 Lobby 服务"
	@echo "  make restart     - 重启 Lobby 服务"
	@echo "  make logs        - 查看日志"
	@echo "  make infra-up    - 启动所有中间件（Mongo/Redis/NATS）"
	@echo "  make infra-down  - 停止所有中间件"
	@echo "  make mongo-up    - 启动 MongoDB"
	@echo "  make mongo-down  - 停止 MongoDB"
	@echo "  make redis-up    - 启动 Redis"
	@echo "  make redis-down  - 停止 Redis"
	@echo "  make nats-up     - 启动 NATS"
	@echo "  make nats-down   - 停止 NATS"
	@echo "  make clean       - 清理所有容器和网络"
	@echo "  make all-up      - 启动全部（网络+中间件+Lobby）"

# 创建共享网络
network:
	docker network create lobby-network 2>/dev/null || true

# 构建镜像
build:
	docker compose build

# 启动 Lobby
up: network
	docker compose up -d

# 停止 Lobby
down:
	docker compose down

# 重启 Lobby
restart:
	docker compose restart

# 查看日志
logs:
	docker compose logs -f

# 启动所有中间件
infra-up: network
	docker compose -f docker-compose.mongo.yml up -d
	docker compose -f docker-compose.redis.yml up -d
	docker compose -f docker-compose.nats.yml up -d

# 停止所有中间件
infra-down:
	docker compose -f docker-compose.mongo.yml down
	docker compose -f docker-compose.redis.yml down
	docker compose -f docker-compose.nats.yml down

# MongoDB
mongo-up: network
	docker compose -f docker-compose.mongo.yml up -d

mongo-down:
	docker compose -f docker-compose.mongo.yml down

# Redis
redis-up: network
	docker compose -f docker-compose.redis.yml up -d

redis-down:
	docker compose -f docker-compose.redis.yml down

# NATS
nats-up: network
	docker compose -f docker-compose.nats.yml up -d

nats-down:
	docker compose -f docker-compose.nats.yml down

# 启动全部
all-up: infra-up up

# 清理
clean:
	docker compose down -v --rmi local 2>/dev/null || true
	docker compose -f docker-compose.mongo.yml down -v 2>/dev/null || true
	docker compose -f docker-compose.redis.yml down -v 2>/dev/null || true
	docker compose -f docker-compose.nats.yml down 2>/dev/null || true
	docker network rm lobby-network 2>/dev/null || true
