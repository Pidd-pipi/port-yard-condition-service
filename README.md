# Backend

Go 1.22 模块 example.com/port-yard-condition-service，入口为 main.go，启动后提供 /healthz 健康检查。

## Standard Commands

在本目录执行：

    go test ./...
    go build ./...
    go run .

服务端口由 PORT 环境变量控制，默认 8080。backend/web/assets.go 使用 go:embed 嵌入同目录下的 index.html 和 app.js。

