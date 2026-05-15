package coordination

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func predictHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req PredictRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	prediction := make([]LoadPrediction, req.Hours)
	for i := 0; i < req.Hours; i++ {
		prediction[i] = LoadPrediction{
			Hour:       i,
			Load:       50 + float64(i)*2 + float64(i%6)*5,
			Price:      0.5 + float64(i%24)/100 + float64(i)/500,
			Carbon:     0.8 - float64(i%24)*0.02,
		}
	}

	history := ScheduleHistory{
		UserID: userID,
		Type:   "predict",
		Status: "completed",
	}
	db.Create(&history)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": PredictResponse{
		Hours:      req.Hours,
		Region:     req.Region,
		Prediction: prediction,
	}})
}

func optimizeHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req OptimizeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	bestOption := OptimizationOption{
		ID:          "opt-001",
		StartTime:   2,
		Duration:    req.Duration,
		Cost:        100.0,
		Carbon:      50.0,
		Score:       0.95,
		Description: "最优方案：选择电价低谷时段执行",
	}

	alternatives := []OptimizationOption{
		{
			ID:          "opt-002",
			StartTime:   8,
			Duration:    req.Duration,
			Cost:        150.0,
			Carbon:      30.0,
			Score:       0.88,
			Description: "低碳方案：选择绿电比例最高时段",
		},
		{
			ID:          "opt-003",
			StartTime:   0,
			Duration:    req.Duration,
			Cost:        80.0,
			Carbon:      80.0,
			Score:       0.75,
			Description: "低成本方案：选择最便宜时段",
		},
	}

	history := ScheduleHistory{
		UserID:        userID,
		Type:          "optimize",
		Status:        "completed",
		CostSaved:     20.0,
		CarbonReduced: 10.0,
	}
	db.Create(&history)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": OptimizeResponse{
		BestOption:    bestOption,
		Alternatives:  alternatives,
		TotalCost:     bestOption.Cost,
		CarbonEmission: bestOption.Carbon,
	}})
}

func executeHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req ExecuteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var history ScheduleHistory
	if err := db.Where("id = ? AND user_id = ?", req.ScheduleID, userID).First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "调度记录不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	history.Status = "executing"
	db.Save(&history)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "调度执行中", "data": history.ToResponse()})
}

func getHistoryHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var histories []ScheduleHistory

	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]ScheduleHistoryResponse, len(histories))
	for i, history := range histories {
		response[i] = history.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getHistoryDetailHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	historyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var history ScheduleHistory
	if err := db.Where("id = ? AND user_id = ?", historyID, userID).First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "调度记录不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": history.ToResponse()})
}