FROM golang:1.21-alpine AS builder

WORKDIR /build

# 复制 go.mod
COPY go.mod ./

# 添加依赖
RUN go get github.com/lib/pq@v1.10.9 && \
    go get github.com/segmentio/kafka-go@v0.4.47 && \
    go get go.uber.org/zap@v1.27.0

# 复制源代码
COPY cmd/ ./cmd/

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -o consumer cmd/consumer/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/consumer .

CMD ["./consumer"]
