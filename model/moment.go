package model

type Moment struct {
	ID        int    `json:"id"`
	Content   string `json:"content"`
	Images    string `json:"images"`
	CreatedAt string `json:"created_at"`
}
