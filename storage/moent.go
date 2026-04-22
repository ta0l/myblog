package storage

import (
	"my_blog/model"
)

// CreateMoment 插入一条新动态，增加 images 参数
func CreateMoment(content string, images string) (int, error) {
	// SQL 语句中新增 images
	stmt, err := DB.Prepare("INSERT INTO moments (content, images) VALUES (?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(content, images)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	return int(id), err
}

// GetMoments 获取动态列表
func GetMoments() ([]model.Moment, error) {
	// SQL 语句中新增 images 字段的查询
	rows, err := DB.Query("SELECT id, content, images, created_at FROM moments ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moments []model.Moment
	for rows.Next() {
		var m model.Moment
		// Scan 必须按 SELECT 的字段顺序一一对应
		if err := rows.Scan(&m.ID, &m.Content, &m.Images, &m.CreatedAt); err != nil {
			return nil, err
		}
		moments = append(moments, m)
	}
	return moments, nil
}

// DeleteMoment 删除动态
func DeleteMoment(id int) error {
	_, err := DB.Exec("DELETE FROM moments WHERE id = ?", id)
	return err
}

// UpdateMoment 更新动态内容
func UpdateMoment(id int, content string, images string) error {
	_, err := DB.Exec("UPDATE moments SET content = ?, images = ? WHERE id = ?", content, images, id)
	return err
}

// GetMomentByID 根据 ID 获取单条动态
func GetMomentByID(id int) (*model.Moment, error) {
	var m model.Moment
	err := DB.QueryRow("SELECT id, content, images FROM moments WHERE id = ?", id).
		Scan(&m.ID, &m.Content, &m.Images)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
