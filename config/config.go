package config

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// 定义配置映射的结构体
type AppConfig struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	COS      COSConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	DBFile string `mapstructure:"db_file"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
	AdminUser string `mapstructure:"admin_user"`
	AdminPass string `mapstructure:"admin_pass"`
}

type COSConfig struct {
	BucketURL string `mapstructure:"bucket_url"`
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
}

// Cfg 是全局配置实例
var Cfg *AppConfig

// InitConfig 初始化配置
func InitConfig() {
	viper.SetConfigName("config") // 配置文件名称(无扩展名)
	viper.SetConfigType("yaml")   // 配置文件类型
	viper.AddConfigPath(".")      // 查找配置文件的路径（当前目录）

	if err := viper.ReadInConfig(); err != nil {
		// 1. 彻底移除旧版 log，使用 slog 记录致命错误并手动退出程序
		slog.Error("读取配置文件失败", slog.Any("error", err))
		os.Exit(1)
	}

	Cfg = &AppConfig{}
	if err := viper.Unmarshal(Cfg); err != nil {
		// 2. 修复 slog 的传参格式，使用键值对记录 err
		slog.Error("解析配置到结构体失败", slog.Any("error", err))
		os.Exit(1)
	}

	// 3. 优化点：打印出最终读取的是哪个路径的配置文件，这在多环境部署时非常有用
	slog.Info("配置加载成功", slog.String("file", viper.ConfigFileUsed()))
}
