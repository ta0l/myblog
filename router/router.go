package router

import (
	"fmt"
	"log/slog"
	"my_blog/config"
	"my_blog/handler"
	"net/http"
	"strings"
	"time"

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

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以便获取状态码
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		// 请求结束后打印结构化日志
		slog.Info("HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.String("latency", time.Since(start).String()),
			slog.String("ip", r.RemoteAddr),
		)
	})
}

// SetupRouter 初始化并配置 API 路由
func SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware) // 挂载跨域中间件

	//RESTful API
	r.Route("/api", func(r chi.Router) {
		// 【公共区】任何人都可以访问
		r.Get("/articles", handler.GetArticlesAPI)
		r.Get("/articles/{articleID}", handler.GetArticleAPI)
		r.Post("/login", handler.LoginAPI)
		r.Get("/sidebar", handler.GetSidebarAPI)
		r.Get("/moments", handler.GetMomentsAPI)

		// 【受保护区】只有带了合法 Token 的请求才能进入
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/articles", handler.CreateArticleAPI)
			r.Put("/articles/{articleID}", handler.UpdateArticleAPI)
			r.Delete("/articles/{articleID}", handler.DeleteArticleAPI)
			r.Post("/moments", handler.CreateMomentAPI)

			r.Get("/upload-url", handler.GetPresignedURLAPI)
			// Token 校验接口
			r.Get("/auth/verify", handler.VerifyAuthAPI)
			r.Delete("/moments/{momentID}", handler.DeleteMomentAPI)
			r.Put("/moments/{momentID}", handler.UpdateMomentAPI)
		})
	})

	return r
}

// authMiddleware 拦截器：查验请求头中是否带有合法的 JWT
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// 💡 记录未授权的访问尝试
			slog.Warn("未授权访问拦截：缺少 Authorization 请求头", slog.String("path", r.URL.Path), slog.String("ip", r.RemoteAddr))
			http.Error(w, `{"error": "未提供 Token"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			slog.Warn("未授权访问拦截：Token 格式错误", slog.String("path", r.URL.Path), slog.String("ip", r.RemoteAddr))
			http.Error(w, `{"error": "Token 格式错误"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// 🚨 关键修复：把原来的 handler.JWTKey 替换为从配置中读取
			return []byte(config.Cfg.Auth.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			// 💡 记录伪造或过期的 Token，这是非常重要的安全日志
			slog.Warn("未授权访问拦截：Token 无效或已过期", slog.String("path", r.URL.Path), slog.String("ip", r.RemoteAddr), slog.Any("error", err))
			http.Error(w, `{"error": "无效的或已过期的 Token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
