package handler

import (
	"encoding/json"
	"log/slog"
	"my_blog/config"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LoginAPI 处理管理员登录请求
func LoginAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("登录请求格式解析失败", slog.String("ip", r.RemoteAddr), slog.Any("error", err))
		respondWithError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	if req.Username != config.Cfg.Auth.AdminUser || req.Password != config.Cfg.Auth.AdminPass {
		slog.Warn("登录失败：账号或密码错误",
			slog.String("attempt_user", req.Username),
			slog.String("ip", r.RemoteAddr))
		respondWithError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(config.Cfg.Auth.JWTSecret))
	if err != nil {
		slog.Error("生成 JWT Token 失败", slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	slog.Info("管理员登录成功", slog.String("user", req.Username), slog.String("ip", r.RemoteAddr))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "登录成功",
		"token":   tokenString,
	})
}

// VerifyAuthAPI 用于前端静默校验 Token 是否过期
func VerifyAuthAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   true,
		"message": "Token is valid",
	})
}
