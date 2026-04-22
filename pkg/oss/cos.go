package oss

import (
	"context"
	"fmt"
	"mime/multipart"
	"my_blog/config"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// UploadToCOS 接收上传的文件并推送到腾讯云，返回文件的访问 URL
func UploadToCOS(file multipart.File, header *multipart.FileHeader) (string, error) {
	u, _ := url.Parse(config.Cfg.COS.BucketURL)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.Cfg.COS.SecretID,
			SecretKey: config.Cfg.COS.SecretKey,
		},
	})

	// 为了避免文件名冲突，通常会使用 UUID 或 时间戳+原文件名 重新命名
	// 这里简单演示直接使用 header.Filename，并在前面加上 moments/ 目录
	cosPath := fmt.Sprintf("moments/%s", header.Filename)

	// 执行上传
	_, err := client.Object.Put(context.Background(), cosPath, file, nil)
	if err != nil {
		return "", err
	}

	// 拼接最终的公网访问地址
	fileURL := fmt.Sprintf("%s/%s", u, cosPath)
	return fileURL, nil
}

// GeneratePresignedURL 生成用于客户端直传的预签名 URL
// filename: 前端准备上传的原始文件名（或者前端生成的随机后缀名）
func GeneratePresignedURL(filename string) (string, string, error) {
	u, _ := url.Parse(config.Cfg.COS.BucketURL)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.Cfg.COS.SecretID,
			SecretKey: config.Cfg.COS.SecretKey,
		},
	})

	// 1. 构造云端的文件路径 (加入时间戳防止重名覆盖)
	cosPath := fmt.Sprintf("moments/%d_%s", time.Now().Unix(), filename)

	// 2. 生成预签名 URL
	// 参数说明：上下文, 请求方法(PUT表示上传), 文件路径, ak, sk, 有效期, 可选配置项
	presignedURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodPut,
		cosPath,
		config.Cfg.COS.SecretID,
		config.Cfg.COS.SecretKey,
		10*time.Minute, // 这个 URL 10 分钟内有效
		nil,
	)
	if err != nil {
		return "", "", err
	}

	// 3. 拼接最终图片在公网的只读访问地址（供前端发布动态时存入数据库使用）
	finalReadURL := fmt.Sprintf("%s/%s", u, cosPath)

	return presignedURL.String(), finalReadURL, nil
}

// DeleteObjects 从 COS 中批量删除文件
func DeleteObjects(imageURLs []string) error {
	u, _ := url.Parse(config.Cfg.COS.BucketURL)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.Cfg.COS.SecretID,
			SecretKey: config.Cfg.COS.SecretKey,
		},
	})

	for _, rawURL := range imageURLs {
		// 1. 解析 URL 提取 Key (例如从 https://.../moments/1.png 提取出 moments/1.png)
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		// 去掉路径开头的 /
		key := strings.TrimPrefix(u.Path, "/")

		// 2. 调用删除接口
		_, err = client.Object.Delete(context.Background(), key)
		if err != nil {
			// 这里记录日志即可，单张图片删除失败通常不应中断整个流程
			continue
		}
	}
	return nil
}
