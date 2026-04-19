package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mojocn/base64Captcha"
)

// 获取配置化的 JWT Secret，如果环境变量未配置，则使用回退默认值（开发使用）
var jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "skopeo-sync-super-secret-key-123456"))

// 获取管理员账密配置
var adminUsername = getEnvOrDefault("ADMIN_USER", "admin")
var adminPassword = getEnvOrDefault("ADMIN_PASSWORD", "admin123")

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// 验证码存储，这里使用内存存储
var store = base64Captcha.DefaultMemStore

// LoginReq 登录请求体
type LoginReq struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	CaptchaID    string `json:"captcha_id"`
	CaptchaValue string `json:"captcha_value"`
}

// GenerateCaptcha 生成图形验证码
func GenerateCaptcha(c *gin.Context) {
	// 创建数字验证码配置
	driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
	captcha := base64Captcha.NewCaptcha(driver, store)
	
	id, b64s, _, err := captcha.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证码失败"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"captcha_id": id,
		"captcha_img": b64s,
	})
}

// Login 管理员登录接口
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	// 校验验证码
	if !store.Verify(req.CaptchaID, req.CaptchaValue, true) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误或已过期"})
		return
	}

	// 使用环境变量配置的管理员账号密码验证
	if req.Username == adminUsername && req.Password == adminPassword {
		// 生成真实的 JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": req.Username,
			"exp":      time.Now().Add(24 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成Token失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": tokenString,
			"msg":   "登录成功",
		})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
}
