package billing

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ywtech/edgeagent-hub/internal/user"
	"github.com/ywtech/edgeagent-hub/pkg/utils"
)

var db *gorm.DB

func InitDB(database *gorm.DB) {
	db = database
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "未授权，请先登录"})
			return
		}
		c.Set("user_id", userID.(uint))
		c.Next()
	}
}

func listBillsHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var bills []Bill

	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&bills).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]BillResponse, len(bills))
	for i, bill := range bills {
		response[i] = bill.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getBillHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	billID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var bill Bill
	if err := db.Where("id = ? AND user_id = ?", billID, userID).First(&bill).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "账单不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": bill.ToResponse()})
}

func payBillHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req PayBillRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var bill Bill
	if err := db.Where("id = ? AND user_id = ?", req.BillID, userID).First(&bill).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "账单不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if bill.Status != "unpaid" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2003, "message": "账单状态错误"})
		return
	}

	var u user.User
	if err := db.First(&u, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	if u.Balance < bill.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2004, "message": "余额不足"})
		return
	}

	tx := db.Begin()

	u.Balance -= bill.Amount
	if err := tx.Save(&u).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	bill.Status = "paid"
	if err := tx.Save(&bill).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "支付成功", "data": bill.ToResponse()})
}

func getUsageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var usages []UsageRecord

	if err := db.Where("user_id = ?", userID).Order("timestamp DESC").Limit(100).Find(&usages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]UsageResponse, len(usages))
	for i, usage := range usages {
		response[i] = usage.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func recordUsageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req RecordUsageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	usage := UsageRecord{
		UserID:       userID,
		ResourceID:   req.ResourceID,
		ResourceType: req.ResourceType,
		Usage:        req.Usage,
		Unit:         req.Unit,
		Timestamp:    utils.GetCurrentTimestamp(),
	}

	if err := db.Create(&usage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "记录成功", "data": usage.ToResponse()})
}