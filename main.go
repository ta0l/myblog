package main

import (
	"log/slog"
	"my_blog/config" // 引入你的 config 包
	"my_blog/pkg/logger"
	"my_blog/router"
	"my_blog/storage"
	"net/http"
	"os"
)

func main() {
	// 1. 最先初始化配置
	config.InitConfig()
	logger.InitLogger()

	// 2. 初始化数据库 (稍后需修改 sqlite.go 使用 config)
	storage.InitDB()

	// 3. 注册路由
	r := router.SetupRouter()

	// 4. 启动服务 (使用 config 里的端口)
	port := ":" + config.Cfg.Server.Port
	slog.Info("博客后端服务已启动", slog.String("port", config.Cfg.Server.Port))
	if err := http.ListenAndServe(port, r); err != nil {
		// 记录致命错误并退出
		slog.Error("服务器异常退出", slog.Any("error", err))
		os.Exit(1)
	}
}
