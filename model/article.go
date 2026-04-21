package model

type Article struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`

	// 新增字段
	CategoryID int      `json:"category_id,omitempty"`
	Category   string   `json:"category"` // 连表查询出来的分类名称
	Tags       []string `json:"tags"`     // 将查出来的标签逗号字符串切分为数组
}
