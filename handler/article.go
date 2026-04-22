package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"my_blog/storage"

	"github.com/go-chi/chi/v5"
)

// 一个辅助函数，用于统一返回 JSON 格式的错误信息
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	// 返回形如 {"error": "错误信息"} 的 JSON
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// GetArticleAPI 处理单篇文章请求，返回 JSON 数据
func GetArticleAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "articleID")
	article, err := storage.GetArticleByID(id)

	if err != nil {
		slog.Error("查询单篇文章失败", slog.String("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	if article == nil {
		respondWithError(w, http.StatusNotFound, "文章不存在")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article)
}

// GetArticlesAPI 处理获取文章列表的请求
func GetArticlesAPI(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")
	keyword := r.URL.Query().Get("q") // q 是标准的搜索缩写

	// 解析页码，默认第 1 页
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// 设置每页显示的数量 (测试时你可以设为 2 或 3，方便看分页效果，生产环境一般 10)
	pageSize := 5

	// 调用底层查询
	result, err := storage.GetAllArticles(category, tag, keyword, page, pageSize)
	if err != nil {
		slog.Error("获取文章列表查询失败",
			slog.String("category", category),
			slog.String("tag", tag),
			slog.String("keyword", keyword),
			slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "获取文章列表失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CreateArticleAPI 处理发布新文章的 POST 请求
func CreateArticleAPI(w http.ResponseWriter, r *http.Request) {
	// 升级请求结构体，增加 Category 和 Tags 字段
	var req struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"` // 接收前端传来的字符串数组
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	if req.Title == "" || req.Content == "" {
		respondWithError(w, http.StatusBadRequest, "标题和内容不能为空")
		return
	}

	id, err := storage.CreateArticleWithTags(req.Title, req.Content, req.Category, req.Tags)
	if err != nil {
		slog.Error("保存文章及关联数据失败", slog.String("title", req.Title), slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "保存文章及关联数据失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "文章发布成功！",
		"id":      id,
	})
}

// UpdateArticleAPI 处理更新文章的 PUT 请求
func UpdateArticleAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "articleID")

	// 增加 Category 和 Tags
	var req struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	// 调用带有事务的更新函数
	err := storage.UpdateArticleWithTags(id, req.Title, req.Content, req.Category, req.Tags)
	if err != nil {
		slog.Error("更新文章数据失败", slog.String("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "更新失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "文章更新成功！"})
}

// DeleteArticleAPI 处理删除文章的 DELETE 请求
func DeleteArticleAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "articleID")

	// 调用数据层删除
	err := storage.DeleteArticle(id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("尝试删除不存在的文章", slog.String("id", id))
			respondWithError(w, http.StatusNotFound, "要删除的文章不存在")
		} else {
			slog.Error("删除文章数据库执行失败", slog.String("id", id), slog.Any("error", err))
			respondWithError(w, http.StatusInternalServerError, "删除失败")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "文章已永久删除！"})
}

func GetSidebarAPI(w http.ResponseWriter, r *http.Request) {
	meta, err := storage.GetSidebarMeta()
	if err != nil {
		slog.Error("获取侧边栏数据统计失败", slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "获取侧边栏数据失败")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}
