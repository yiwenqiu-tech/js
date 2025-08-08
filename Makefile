.PHONY: test test-verbose test-coverage test-race clean build

# 运行所有测试
test:
	go test ./...

# 运行测试并显示详细信息
test-verbose:
	go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 运行测试并检测竞态条件
test-race:
	go test -race ./...

# 清理测试文件
clean:
	rm -f coverage.out coverage.html

# 构建项目
build:
	go build -o jieyou-backend main.go

# 运行项目
run:
	go run main.go

# 安装依赖
deps:
	go mod tidy
	go mod download

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 完整测试流程
test-all: deps fmt test test-coverage 