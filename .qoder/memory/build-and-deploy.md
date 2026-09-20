# 构建与部署

## 构建验证

```bash
go build ./...
go vet ./...
```

## Docker 部署

### 相关文件

| 文件 | 用途 |
|------|------|
| `Dockerfile` | 多阶段构建镜像 |
| `config.docker.yaml` | Docker 环境配置 |
| `docker-compose.yml` | 完整服务（Lobby + 中间件） |
| `docker-compose.mongo.yml` | 仅 MongoDB |
| `docker-compose.redis.yml` | 仅 Redis |
| `docker-compose.nats.yml` | 仅 NATS |
| `docker-compose.infra.yml` | 所有中间件 |
| `docker-compose.local.yml` | 仅 Lobby（连接宿主机中间件） |
| `Dockerfile.local` | 本地模式构建 |
| `config.local.yaml` | 本地模式配置 |
| `check-infra.bat` | 检查并启动中间件脚本 |
| `Makefile` | Linux/Mac 快捷命令 |
| `make.bat` | Windows 快捷命令 |

### 部署模式

**模式1：全容器化（推荐新环境）**
```cmd
.\make.bat all-up
```
启动 Lobby + MongoDB + Redis + NATS，所有服务独立容器。

**模式2：本地模式（连接已有中间件）**
```cmd
.\make.bat check-infra   # 检查中间件状态，未运行则自动启动
.\make.bat local-up       # 启动 Lobby（连接宿主机中间件）
```
仅启动 Lobby 容器，连接宿主机已运行的 MongoDB/Redis/NATS。

### 常用命令

**Windows (PowerShell/cmd):**
```cmd
.\make.bat all-up        # 全容器化启动
.\make.bat local-up      # 本地模式启动
.\make.bat local-down    # 本地模式停止
.\make.bat check-infra   # 检查并启动中间件
.\make.bat infra-up      # 仅启动中间件
.\make.bat mongo-up      # 启动单个中间件
.\make.bat redis-up
.\make.bat nats-up
.\make.bat down          # 停止服务
.\make.bat infra-down
```

**Linux/Mac (bash):**
```bash
make all-up
make infra-up
make down
```

### 网络配置

Docker 容器使用 `lobby-network` 桥接网络，服务间通过容器名访问：
- MongoDB: `mongo:27017`
- Redis: `redis:6379`
- NATS: `nats:4222`
