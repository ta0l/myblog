package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTKey 是用于加密和解密 Token 的盐值（极其重要，生产环境中绝不能写死在代码里！）
var JWTKey = []byte("my_super_secret_blog_key_1024")

// LoginAPI 处理管理员登录请求
func LoginAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	// 【简易防线】因为是个人博客，我们暂时将账号密码硬编码写死
	// 生产环境中，这里应该去查询数据库，并且对比密码的 Hash 值
	if req.Username != "admin" || req.Password != "123456" {
		respondWithError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 账号密码正确，开始生成 JWT
	// 包含两部分信息：载荷 (Claims) 比如用户名和过期时间，以及签名方法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token 24 小时后过期
	})

	// 使用我们的专属密钥给 Token 签名
	tokenString, err := token.SignedString(JWTKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	// 将生成的 Token 返回给前端
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "登录成功",
		"token":   tokenString,
	})
}
