package main

import (
	"fmt"
	"my_blog/router" // 引入路由包
	"my_blog/storage"
	"net/http"
)

func main() {
	// 1. 初始化数据库
	storage.InitDB()

	// 3. 获取配置好的路由引擎
	r := router.SetupRouter()

	// 4. 启动服务
	fmt.Println("🚀 博客服务器已启动")
	fmt.Println("👉 测试你的数据库: http://localhost:8888/post/1")

	err := http.ListenAndServe(":8888", r)
	if err != nil {
		fmt.Println("服务器启动失败:", err)
	}
}
