package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger 初始化企业级日志
func InitLogger() {
	// 1. 配置日志轮转 (Lumberjack)
	fileWriter := &lumberjack.Logger{
		Filename:   "logs/app.log", // 日志文件路径
		MaxSize:    10,             // 每个日志文件最大 10 MB
		MaxBackups: 5,              // 保留最近的 5 个日志文件
		MaxAge:     30,             // 最多保留 30 天
		Compress:   true,           // 是否压缩旧日志 (gzip)
	}

	// 2. 配置多路输出：同时输出到控制台和文件
	// 控制台为了方便人看，依然留着；文件用来持久化
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	// 3. 配置 Slog 处理器 (这里使用 JSON 格式输出，方便后续接入分析系统)
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // 设置最低输出级别：Debug < Info < Warn < Error
		// AddSource: true,   // 如果设为 true，会打印输出日志的文件名和行号
	}

	// 使用 JSON 格式处理器
	handler := slog.NewJSONHandler(multiWriter, opts)

	// 4. 将配置好的 logger 设为全局默认
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("结构化日志初始化成功")
}
