# 构建阶段
FROM golang:1.27-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装依赖（使用 vendor 模式不需要下载）
# COPY go.mod go.sum ./
# RUN go mod download

# 复制源码和 vendor
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o lobby .

# 运行阶段
FROM alpine:3.18

# 安装必要工具
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/lobby .

# 复制配置文件
COPY config.docker.yaml config.yaml

# 创建日志目录
RUN mkdir -p logs

# 暴露端口
EXPOSE 9001 9002

# 启动命令
CMD ["./lobby"]
