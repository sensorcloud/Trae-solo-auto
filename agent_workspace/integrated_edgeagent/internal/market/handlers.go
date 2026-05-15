package market

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ywtech/edgeagent-hub/internal/user"
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

func createAssetHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateAssetRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	asset := Asset{
		UserID:        userID,
		Name:          req.Name,
		Description:   req.Description,
		GPUModel:      req.GPUModel,
		GPUNumber:     req.GPUNumber,
		Memory:        req.Memory,
		Region:        req.Region,
		Price:         req.Price,
		Performance:   req.Performance,
	}

	if err := db.Create(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "创建成功", "data": asset.ToResponse()})
}

func listAssetsHandler(c *gin.Context) {
	var assets []Asset

	region := c.Query("region")
	gpuModel := c.Query("gpu_model")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")

	query := db.Where("status = ?", "available")

	if region != "" {
		query = query.Where("region = ?", region)
	}
	if gpuModel != "" {
		query = query.Where("gpu_model LIKE ?", "%"+gpuModel+"%")
	}
	if minPrice != "" {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice != "" {
		query = query.Where("price <= ?", maxPrice)
	}

	if err := query.Find(&assets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]AssetResponse, len(assets))
	for i, asset := range assets {
		response[i] = asset.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getAssetHandler(c *gin.Context) {
	assetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var asset Asset
	if err := db.First(&asset, assetID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "资源不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": asset.ToResponse()})
}

func updateAssetHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	assetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var asset Asset
	if err := db.Where("id = ? AND user_id = ?", assetID, userID).First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "资源不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	asset.Name = req.Name
	asset.Description = req.Description
	asset.GPUModel = req.GPUModel
	asset.GPUNumber = req.GPUNumber
	asset.Memory = req.Memory
	asset.Region = req.Region
	asset.Price = req.Price
	asset.Performance = req.Performance

	if err := db.Save(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": asset.ToResponse()})
}

func deleteAssetHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	assetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", assetID, userID).Delete(&Asset{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func createOrderHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var asset Asset
	if err := db.Where("id = ? AND status = ?", req.AssetID, "available").First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "资源不存在或不可用"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	totalPrice := asset.Price * float64(req.Quantity) * float64(req.Duration)

	var u user.User
	if err := db.First(&u, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	if u.Balance < totalPrice {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2004, "message": "余额不足"})
		return
	}

	order := Order{
		UserID:        userID,
		AssetID:       req.AssetID,
		Quantity:      req.Quantity,
		TotalPrice:    totalPrice,
		PaymentMethod: req.PaymentMethod,
		Duration:      req.Duration,
	}

	tx := db.Begin()
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	u.Balance -= totalPrice
	if err := tx.Save(&u).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	asset.Status = "sold"
	if err := tx.Save(&asset).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "订单创建成功", "data": order.ToResponse()})
}

func listOrdersHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var orders []Order

	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]OrderResponse, len(orders))
	for i, order := range orders {
		response[i] = order.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getOrderHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var order Order
	if err := db.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "订单不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": order.ToResponse()})
}

func payOrderHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var order Order
	if err := db.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "订单不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2003, "message": "订单状态错误"})
		return
	}

	var u user.User
	if err := db.First(&u, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	if u.Balance < order.TotalPrice {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2004, "message": "余额不足"})
		return
	}

	tx := db.Begin()

	u.Balance -= order.TotalPrice
	if err := tx.Save(&u).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	order.Status = "paid"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "支付成功", "data": order.ToResponse()})
}

func cancelOrderHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var order Order
	if err := db.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "订单不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if order.Status != "pending" && order.Status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2003, "message": "订单状态错误，无法取消"})
		return
	}

	tx := db.Begin()

	if order.Status == "paid" {
		var u user.User
		if err := tx.First(&u, userID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
			return
		}
		u.Balance += order.TotalPrice
		if err := tx.Save(&u).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
			return
		}
	}

	order.Status = "cancelled"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "取消成功", "data": order.ToResponse()})
}