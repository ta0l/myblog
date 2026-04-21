package storage

import (
	"database/sql"
	"log"

	// 引入纯 Go 版 SQLite 驱动。前面的下划线 "_" 表示：
	// 我们只执行它内部的 init() 函数来注册驱动，而不直接调用它的 API
	_ "modernc.org/sqlite"
)

// DB 是一个全局的数据库连接池对象，方便其他包调用
var DB *sql.DB

// InitDB 初始化数据库连接并创建表
func InitDB() {
	var err error
	// 连接本地名为 blog.db 的数据库文件 (如果不存在会自动创建)
	DB, err = sql.Open("sqlite", "./blog.db")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
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

	`

	// 执行建表语句
	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatal("创建数据表失败:", err)
	}

	log.Println("数据库初始化成功！")
}
