package handler

import (
	"encoding/json"
	"log/slog"
	"my_blog/pkg/oss"
	"my_blog/storage"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// GetMomentsAPI 获取所有动态
func GetMomentsAPI(w http.ResponseWriter, r *http.Request) {
	moments, err := storage.GetMoments()
	if err != nil {
		slog.Error("获取动态列表失败", slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "获取动态失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(moments)
}

// CreateMomentAPI 创建新动态
func CreateMomentAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string   `json:"content"`
		Images  []string `json:"images"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("创建动态请求解析失败", slog.String("ip", r.RemoteAddr), slog.Any("error", err))
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	if req.Content == "" && len(req.Images) == 0 {
		slog.Warn("创建动态被拦截：内容和图片均为空")
		respondWithError(w, http.StatusBadRequest, "内容和图片不能同时为空")
		return
	}

	imagesJSON, err := json.Marshal(req.Images)
	if err != nil {
		slog.Warn("动态图片列表 JSON 序列化失败，退化为空数组", slog.Any("error", err))
		imagesJSON = []byte("[]")
	}

	id, err := storage.CreateMoment(req.Content, string(imagesJSON))
	if err != nil {
		slog.Error("保存新动态到数据库失败", slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "发布动态失败")
		return
	}

	slog.Info("新动态发布成功", slog.Int("id", int(id)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "发布成功",
		"id":      id,
	})
}

// DeleteMomentAPI 处理删除请求
func DeleteMomentAPI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "momentID")
	id, _ := strconv.Atoi(idStr)

	moment, err := storage.GetMomentByID(id)
	if err != nil {
		slog.Warn("尝试删除不存在的动态", slog.Int("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusNotFound, "动态不存在")
		return
	}

	var images []string
	if err := json.Unmarshal([]byte(moment.Images), &images); err != nil {
		slog.Warn("解析待删除动态的图片列表失败", slog.Int("id", id), slog.Any("error", err))
		images = []string{}
	}

	if len(images) > 0 {
		if err := oss.DeleteObjects(images); err != nil {
			slog.Warn("同步清理云端动态图片存在异常", slog.Int("moment_id", id), slog.Any("error", err))
		} else {
			slog.Info("同步清理云端动态图片成功", slog.Int("moment_id", id), slog.Int("count", len(images)))
		}
	}

	if err := storage.DeleteMoment(id); err != nil {
		slog.Error("从数据库删除动态失败", slog.Int("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "数据库删除失败")
		return
	}

	slog.Info("动态删除成功", slog.Int("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// UpdateMomentAPI 处理更新请求
func UpdateMomentAPI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "momentID")
	id, _ := strconv.Atoi(idStr)

	oldMoment, err := storage.GetMomentByID(id)
	if err != nil {
		slog.Warn("尝试更新不存在的动态", slog.Int("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusNotFound, "动态不存在")
		return
	}

	var oldImages []string
	if err := json.Unmarshal([]byte(oldMoment.Images), &oldImages); err != nil {
		slog.Warn("解析旧动态图片列表失败", slog.Int("id", id), slog.Any("error", err))
	}

	var req struct {
		Content string   `json:"content"`
		Images  []string `json:"images"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("更新动态请求解析失败", slog.Int("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusBadRequest, "无效的请求参数")
		return
	}

	var toDelete []string
	newImagesMap := make(map[string]bool)
	for _, img := range req.Images {
		newImagesMap[img] = true
	}

	for _, oldImg := range oldImages {
		if !newImagesMap[oldImg] {
			toDelete = append(toDelete, oldImg)
		}
	}

	if len(toDelete) > 0 {
		go func() {
			if err := oss.DeleteObjects(toDelete); err != nil {
				slog.Warn("异步清理动态过期图片存在异常", slog.Any("error", err))
			} else {
				slog.Info("异步清理动态过期图片成功", slog.Int("count", len(toDelete)))
			}
		}()
	}

	imagesJSON, _ := json.Marshal(req.Images)
	if err := storage.UpdateMoment(id, req.Content, string(imagesJSON)); err != nil {
		slog.Error("更新动态到数据库失败", slog.Int("id", id), slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "数据库更新失败")
		return
	}

	slog.Info("动态更新成功", slog.Int("id", id))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "修改成功，已同步清理过期资源"})
}
