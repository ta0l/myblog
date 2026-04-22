package handler

import (
	"encoding/json"
	"log/slog"
	"my_blog/pkg/oss"
	"net/http"
)

// GetPresignedURLAPI 返回客户端直传所需的临时链接
func GetPresignedURLAPI(w http.ResponseWriter, r *http.Request) {
	// 假设前端通过查询参数传递准备上传的文件名：?filename=avatar.png
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		slog.Warn("请求上传链接被拒绝：未提供文件名", slog.String("ip", r.RemoteAddr))
		respondWithError(w, http.StatusBadRequest, "请提供文件名")
		return
	}

	// 调用服务生成 URL
	uploadURL, readURL, err := oss.GeneratePresignedURL(filename)
	if err != nil {
		slog.Error("生成 COS 上传签名失败", slog.String("filename", filename), slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "生成上传链接失败")
		return
	}

	slog.Info("成功下发 COS 直传链接", slog.String("filename", filename), slog.String("ip", r.RemoteAddr))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"upload_url": uploadURL, // 前端用这个地址执行 PUT 请求上传文件实体
		"read_url":   readURL,   // 前端在发布动态时，把这个地址提交给后端的 /moments 接口
	})
}
