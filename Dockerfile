# 1. 使用官方 Golang 1.24.4 镜像作为构建环境
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# 1. 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN go env -w GO111MODULE=on &&\
    go env -w GOPROXY=https://goproxy.cn,direct

# 2. 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 3. 复制项目源码
COPY . .

# 4. 编译 Go 可执行文件（静态编译）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app main.go

# 5. 使用更小的基础镜像运行
FROM alpine:latest

WORKDIR /app

# 6. 复制编译好的二进制文件
COPY --from=builder /app/app .

# 7. 配置端口
EXPOSE 8080

# 8. 启动服务
CMD ["./app"] 