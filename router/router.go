package router

import (
	"my_blog/handler"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// corsMiddleware 跨域中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // 允许所有域名访问
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")

		// 浏览器在发送跨域 POST/PUT 请求前，会先发一个 OPTIONS 请求试探，我们直接放行
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetupRouter 初始化并配置 API 路由
func SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware) // 挂载跨域中间件

	//RESTful API
	r.Route("/api", func(r chi.Router) {
		// 【公共区】任何人都可以访问
		r.Get("/articles", handler.GetArticlesAPI)
		r.Get("/articles/{articleID}", handler.GetArticleAPI)
		r.Post("/login", handler.LoginAPI)
		r.Get("/sidebar", handler.GetSidebarAPI)

		// 【受保护区】只有带了合法 Token 的请求才能进入
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/articles", handler.CreateArticleAPI)
			r.Put("/articles/{articleID}", handler.UpdateArticleAPI)
			r.Delete("/articles/{articleID}", handler.DeleteArticleAPI)
		})
	})

	return r
}

// authMiddleware 拦截器：查验请求头中是否带有合法的 JWT
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 获取 Authorization 请求头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "未提供认证 Token"}`, http.StatusUnauthorized)
			return
		}

		// 2. 标准的 Token 格式为 "Bearer xxxxx.yyyyy.zzzzz"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Token 格式错误"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// 3. 解析并验证 Token 是否被篡改或过期
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 这里必须提供和签发时一模一样的密钥
			return handler.JWTKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"error": "Token 无效或已过期"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
