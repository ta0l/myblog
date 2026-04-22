package storage

import (
	"database/sql"
	"log/slog" // 仅保留 slog
	"my_blog/config"
	"os"

	// 引入纯 Go 版 SQLite 驱动。前面的下划线 "_" 表示：
	// 我们只执行它内部的 init() 函数来注册驱动，而不直接调用它的 API
	_ "modernc.org/sqlite"
)

// DB 是一个全局的数据库连接池对象，方便其他包调用
var DB *sql.DB

// InitDB 初始化数据库连接并创建表
func InitDB() {
	var err error
	// 连接本地名为 blog.db 的数据库文件
	DB, err = sql.Open("sqlite", config.Cfg.Database.DBFile)
	if err != nil {
		// 🚨 使用 slog.Error 记录结构化错误，并手动退出程序
		slog.Error("数据库连接失败", slog.Any("error", err), slog.String("db_file", config.Cfg.Database.DBFile))
		os.Exit(1)
	}

	// 测试数据库连接是否真正有效 (sql.Open 只是验证参数，Ping 才是真正的连接测试)
	if err = DB.Ping(); err != nil {
		slog.Error("数据库 Ping 失败", slog.Any("error", err))
		os.Exit(1)
	}

	// 编写建表 SQL 语句
	createTableSQL := `
        -- 1. 分类表
        CREATE TABLE IF NOT EXISTS categories (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE
        );

        -- 2. 标签表
        CREATE TABLE IF NOT EXISTS tags (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE
        );

        -- 3. 文章表 (新增 category_id)
        CREATE TABLE IF NOT EXISTS articles (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            category_id INTEGER -- 指向 categories 表的外键
        );

        -- 4. 文章-标签关联表 (解决多对多关系的核心)
        CREATE TABLE IF NOT EXISTS article_tags (
            article_id INTEGER,
            tag_id INTEGER,
            PRIMARY KEY (article_id, tag_id) -- 联合主键，确保一篇文章不会有两个相同的标签
        );

        -- 5. 动态表 (Moments)
        CREATE TABLE IF NOT EXISTS moments (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            content TEXT NOT NULL,
            images TEXT DEFAULT '[]', -- 存储 JSON 格式的图片 URL 数组，例如 '["url1", "url2"]'
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );

    `

	// 执行建表语句
	_, err = DB.Exec(createTableSQL)
	if err != nil {
		// 🚨 使用 slog.Error 记录建表失败的具体原因
		slog.Error("创建数据表失败", slog.Any("error", err))
		os.Exit(1)
	}

	// ✅ 优化：不仅记录成功，还把最终使用的数据库驱动和路径打印出来，方便核对
	slog.Info("数据库初始化成功", slog.String("driver", "sqlite"), slog.String("db_file", config.Cfg.Database.DBFile))
}
