package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/middleware"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB(database *gorm.DB) {
	db = database
}

func registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var existingUser User
	if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 2002, "message": "邮箱已被注册"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	user := User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Phone:    req.Phone,
		Nickname: req.Nickname,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "注册成功", "data": user.ToResponse()})
}

func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	var user User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "邮箱或密码错误"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1004, "message": "账户已被禁用"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "邮箱或密码错误"})
		return
	}

	accessToken, refreshToken, expireAt, err := generateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "登录成功", "data": TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpireAt:     expireAt,
	}})
}

func logoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "退出成功"})
}

func refreshHandler(c *gin.Context) {
	refreshToken := c.GetHeader("Refresh-Token")
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "缺少Refresh Token"})
		return
	}

	claims, err := middleware.ValidateRefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "Refresh Token无效"})
		return
	}

	var user User
	if err := db.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "用户不存在"})
		return
	}

	newAccessToken, err := middleware.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	newRefreshToken, err := middleware.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Token刷新成功", "data": TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpireAt:     0,
	}})
}

func generateTokens(userID uint, email, role string) (string, string, int64, error) {
	accessToken, err := middleware.GenerateAccessToken(userID, email, role)
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, err := middleware.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, 0, nil
}

func getUserHandler(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": user.ToResponse()})
}

func updateUserHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "用户不存在"})
		return
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": user.ToResponse()})
}

func deleteUserHandler(c *gin.Context) {
	userID := c.GetUint("user_id")

	if err := db.Delete(&User{}, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}