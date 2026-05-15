package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

type PagedResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"items"`
	Total     int64       `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Code      int               `json:"code"`
	Message   string            `json:"message"`
	Errors    map[string]string `json:"errors,omitempty"`
	Timestamp int64             `json:"timestamp"`
	RequestID string            `json:"request_id,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func Created(c *gin.Context, data interface{}) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusCreated, Response{
		Code:      0,
		Message:   "created",
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func PagedSuccess(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusOK, PagedResponse{
		Code:      0,
		Message:   "success",
		Data:      items,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func BadRequest(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:      http.StatusBadRequest,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func BadRequestWithErrors(c *gin.Context, message string, errors map[string]string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:      http.StatusBadRequest,
		Message:   message,
		Errors:    errors,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func Unauthorized(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Code:      http.StatusUnauthorized,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func Forbidden(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusForbidden, ErrorResponse{
		Code:      http.StatusForbidden,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func NotFound(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusNotFound, ErrorResponse{
		Code:      http.StatusNotFound,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func Conflict(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusConflict, ErrorResponse{
		Code:      http.StatusConflict,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func InternalError(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:      http.StatusInternalServerError,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func ServiceUnavailable(c *gin.Context, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusServiceUnavailable, ErrorResponse{
		Code:      http.StatusServiceUnavailable,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: getStringOrEmpty(requestID),
	})
}

func getStringOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
