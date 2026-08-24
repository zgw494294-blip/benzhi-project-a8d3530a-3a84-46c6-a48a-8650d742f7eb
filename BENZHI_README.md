# BENZHI_README

基于 Go 实现的海岸带修复证据验收台 Web 项目，一款后端服务，已完整实现海岸带修复证据验收台，包括浏览器工作台、全部案件流程、哈希事件账本、离线恢复、并发与幂等控制、独立验收及放行凭据。全部回归测试和默认地址全链路自检均已通过。

## 项目说明
- 项目：benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb
- 项目用途：已完整实现海岸带修复证据验收台，包括浏览器工作台、全部案件流程、哈希事件账本、离线恢复、并发与幂等控制、独立验收及放行凭据。全部回归测试和默认地址全链路自检均已通过。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run . -selfcheck
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb-arm64 linux/arm64
docker run -it benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run . -selfcheck`
