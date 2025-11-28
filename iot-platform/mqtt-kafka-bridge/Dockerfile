# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制依赖文件和源代码
COPY go.mod ./
COPY cmd/ ./cmd/

# 下载依赖并生成 go.sum
RUN go mod tidy
RUN go mod download
RUN go mod verify

# 编译 (静态链接)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o bridge \
    ./cmd/bridge

# Stage 2: Runtime
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 bridge && \
    adduser -D -u 1000 -G bridge bridge

# 从 builder 复制二进制文件
COPY --from=builder /build/bridge /app/bridge

# 切换到非 root 用户
USER bridge

# 健康检查 (进程存活即可)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep bridge || exit 1

ENTRYPOINT ["/app/bridge"]
