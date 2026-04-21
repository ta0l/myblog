package storage

import (
	"database/sql"
	"my_blog/model"
	"strings"
)

// PaginatedArticles 分页返回结构体
type PaginatedArticles struct {
	Articles   []model.Article `json:"articles"`
	Total      int             `json:"total"`       // 符合条件的总文章数
	Page       int             `json:"page"`        // 当前页码
	PageSize   int             `json:"page_size"`   // 每页数量
	TotalPages int             `json:"total_pages"` // 总页数
}

// CreateArticle 插入一篇新文章，并返回新文章的 ID
func CreateArticleWithTags(title, content, categoryName string, tags []string) (int64, error) {
	// 1. 开启数据库事务
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}

	// 极其关键的防御机制：如果函数 return 前没有执行 Commit，就会自动 Rollback（回滚撤销所有操作）
	defer tx.Rollback()

	// 2. 处理分类 (如果存在就忽略，不存在就插入)
	var categoryID int64
	if categoryName != "" {
		// INSERT OR IGNORE 是 SQLite 的神器，防止重复名称报错
		_, err = tx.Exec(`INSERT OR IGNORE INTO categories (name) VALUES (?)`, categoryName)
		if err != nil {
			return 0, err
		}
		// 把刚才插入（或已存在）的分类 ID 查出来
		err = tx.QueryRow(`SELECT id FROM categories WHERE name = ?`, categoryName).Scan(&categoryID)
		if err != nil {
			return 0, err
		}
	}

	// 3. 插入文章本体，并关联分类 ID
	// 如果 categoryID 是 0（用户没填），可以存 NULL 或者默认值，这里简化处理存 0
	res, err := tx.Exec(`INSERT INTO articles (title, content, category_id) VALUES (?, ?, ?)`, title, content, categoryID)
	if err != nil {
		return 0, err
	}

	articleID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// 4. 处理标签数组并建立多对多关联
	for _, tagName := range tags {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		var tagID int64
		// 插入新标签或忽略已存在的标签
		_, err = tx.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tagName)
		if err != nil {
			return 0, err
		}

		// 查询当前标签的 ID
		err = tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, tagName).Scan(&tagID)
		if err != nil {
			return 0, err
		}

		// 将文章 ID 和标签 ID 写入多对多关联表
		_, err = tx.Exec(`INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)`, articleID, tagID)
		if err != nil {
			return 0, err
		}
	}

	// 5. 走到这里说明所有操作都成功了，提交事务，让数据真正落盘！
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return articleID, nil
}

// GetArticleByID 连表查询单篇文章的分类和所有标签
func GetArticleByID(id string) (*model.Article, error) {
	// 使用 LEFT JOIN 确保即使文章没有分类或标签，也能查出文章主体
	stmt := `
		SELECT 
			a.id, a.title, a.content, a.created_at,
			IFNULL(c.name, '未分类') as category_name,
			GROUP_CONCAT(t.name) as tags_string
		FROM articles a
		LEFT JOIN categories c ON a.category_id = c.id
		LEFT JOIN article_tags at ON a.id = at.article_id
		LEFT JOIN tags t ON at.tag_id = t.id
		WHERE a.id = ?
		GROUP BY a.id; -- 必须根据文章 ID 分组，才能让 GROUP_CONCAT 生效
	`

	row := DB.QueryRow(stmt, id)
	var a model.Article
	var tagsString sql.NullString // 处理 tags_string 可能为 NULL 的情况

	err := row.Scan(&a.ID, &a.Title, &a.Content, &a.CreatedAt, &a.Category, &tagsString)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 极其优雅的后处理：将数据库返回的 "Go,后端,数据库" 字符串，切分成 Go 数组
	if tagsString.Valid && tagsString.String != "" {
		a.Tags = strings.Split(tagsString.String, ",")
	} else {
		a.Tags = []string{} // 确保返回空数组而不是 null
	}

	return &a, nil
}

// GetAllArticles 获取文章列表，支持按分类、标签、关键字搜索及分页
func GetAllArticles(category, tag, keyword string, page, pageSize int) (*PaginatedArticles, error) {
	var args []interface{}
	whereClause := "WHERE 1=1"

	// 1. 动态拼接过滤条件
	if category != "" {
		whereClause += ` AND c.name = ?`
		args = append(args, category)
	}
	if tag != "" {
		whereClause += ` AND a.id IN (
			SELECT article_id FROM article_tags at2 
			JOIN tags t2 ON at2.tag_id = t2.id 
			WHERE t2.name = ?
		)`
		args = append(args, tag)
	}
	if keyword != "" {
		// 使用 LIKE 进行模糊搜索，匹配标题或正文
		whereClause += ` AND (a.title LIKE ? OR a.content LIKE ?)`
		searchStr := "%" + keyword + "%"
		args = append(args, searchStr, searchStr)
	}

	// 2. 极其关键：先查出符合条件的总记录数 (用于前端计算页码)
	// 注意这里必须用 COUNT(DISTINCT a.id)，因为 LEFT JOIN 标签会产生重复记录
	countQuery := `
		SELECT COUNT(DISTINCT a.id) 
		FROM articles a
		LEFT JOIN categories c ON a.category_id = c.id
		LEFT JOIN article_tags at ON a.id = at.article_id
		LEFT JOIN tags t ON at.tag_id = t.id
	` + whereClause

	var total int
	err := DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := (total + pageSize - 1) / pageSize

	// 3. 核心数据查询 (加入 LIMIT 和 OFFSET 实现分页)
	dataQuery := `
		SELECT 
			a.id, a.title, a.created_at,
			IFNULL(c.name, '未分类') as category_name,
			GROUP_CONCAT(t.name) as tags_string
		FROM articles a
		LEFT JOIN categories c ON a.category_id = c.id
		LEFT JOIN article_tags at ON a.id = at.article_id
		LEFT JOIN tags t ON at.tag_id = t.id
	` + whereClause + `
		GROUP BY a.id ORDER BY a.created_at DESC
		LIMIT ? OFFSET ?
	`

	// 追加分页参数：限制查 pageSize 条，跳过 (page-1)*pageSize 条
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	rows, err := DB.Query(dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make([]model.Article, 0)
	for rows.Next() {
		var a model.Article
		var tagsString sql.NullString
		if err := rows.Scan(&a.ID, &a.Title, &a.CreatedAt, &a.Category, &tagsString); err == nil {
			if tagsString.Valid && tagsString.String != "" {
				a.Tags = strings.Split(tagsString.String, ",")
			} else {
				a.Tags = []string{}
			}
			articles = append(articles, a)
		}
	}

	// 4. 打包返回分页结构
	return &PaginatedArticles{
		Articles:   articles,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateArticle 根据 ID 修改文章的标题和内容
func UpdateArticleWithTags(id string, title, content, categoryName string, tags []string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // 发生错误时自动回滚

	// 1. 处理分类
	var categoryID sql.NullInt64 // 使用 NullInt64 防止分类为空
	if categoryName != "" {
		_, err = tx.Exec(`INSERT OR IGNORE INTO categories (name) VALUES (?)`, categoryName)
		if err != nil {
			return err
		}
		err = tx.QueryRow(`SELECT id FROM categories WHERE name = ?`, categoryName).Scan(&categoryID)
		if err != nil {
			return err
		}
	}

	// 2. 更新文章基础信息和分类ID
	_, err = tx.Exec(`UPDATE articles SET title = ?, content = ?, category_id = ? WHERE id = ?`,
		title, content, categoryID, id)
	if err != nil {
		return err
	}

	// 3. 【核心策略】先彻底删除这篇文章与旧标签的所有关联！
	_, err = tx.Exec(`DELETE FROM article_tags WHERE article_id = ?`, id)
	if err != nil {
		return err
	}

	// 4. 重新插入前端传来的所有新标签
	for _, tagName := range tags {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		var tagID int64
		// 确保标签在 tags 表中存在
		_, err = tx.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tagName)
		if err != nil {
			return err
		}

		err = tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, tagName).Scan(&tagID)
		if err != nil {
			return err
		}

		// 重新建立新的多对多关联
		_, err = tx.Exec(`INSERT INTO article_tags (article_id, tag_id) VALUES (?, ?)`, id, tagID)
		if err != nil {
			return err
		}
	}

	// 5. 提交事务
	return tx.Commit()
}

// DeleteArticle 彻底删除文章，并清理所有相关的多对多关联和孤儿数据
func DeleteArticle(id string) error {
	// 开启事务，保证删除操作的原子性（要么全删，要么全不删）
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM article_tags WHERE article_id = ?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM tags WHERE id NOT IN (SELECT DISTINCT tag_id FROM article_tags)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM categories WHERE id NOT IN (SELECT DISTINCT category_id FROM articles WHERE category_id IS NOT NULL)`)
	if err != nil {
		return err
	}

	// 提交事务，让上述四个毁灭性操作真正生效！
	return tx.Commit()
}

// MetaItem 用于统一返回带有数量的分类或标签
type MetaItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetSidebarMeta 获取所有分类和标签及其对应的文章数量
func GetSidebarMeta() (map[string]interface{}, error) {
	// 1. 获取分类统计 (只查询有文章的分类)
	catStmt := `
		SELECT c.name, COUNT(a.id) 
		FROM categories c 
		JOIN articles a ON c.id = a.category_id 
		GROUP BY c.id ORDER BY COUNT(a.id) DESC
	`
	catRows, err := DB.Query(catStmt)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()

	categories := make([]MetaItem, 0)
	for catRows.Next() {
		var item MetaItem
		if err := catRows.Scan(&item.Name, &item.Count); err == nil {
			categories = append(categories, item)
		}
	}

	// 2. 获取标签统计
	tagStmt := `
		SELECT t.name, COUNT(at.article_id) 
		FROM tags t 
		JOIN article_tags at ON t.id = at.tag_id 
		GROUP BY t.id ORDER BY COUNT(at.article_id) DESC
	`
	tagRows, err := DB.Query(tagStmt)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()

	tags := make([]MetaItem, 0)
	for tagRows.Next() {
		var item MetaItem
		if err := tagRows.Scan(&item.Name, &item.Count); err == nil {
			tags = append(tags, item)
		}
	}

	// 返回组合数据
	return map[string]interface{}{
		"categories": categories,
		"tags":       tags,
	}, nil
}
