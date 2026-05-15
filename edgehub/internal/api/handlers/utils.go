package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaginationParams struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
	Offset   int `form:"offset" json:"offset"`
	Limit    int `form:"limit" json:"limit"`
}

type SortParams struct {
	SortBy string `form:"sort_by" json:"sort_by"`
	Order  string `form:"order" json:"order"`
}

type TimeRangeParams struct {
	StartTime *time.Time `form:"start_time" json:"start_time"`
	EndTime   *time.Time `form:"end_time" json:"end_time"`
}

func GetPagination(c *gin.Context) PaginationParams {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := parseIntDefault(c.Query("offset"), 0)
	limit := parseIntDefault(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
		Limit:    limit,
	}
}

func GetSortParams(c *gin.Context) SortParams {
	sortBy := c.Query("sort_by")
	order := c.Query("order")
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	return SortParams{
		SortBy: sortBy,
		Order:  order,
	}
}

func GetTimeRange(c *gin.Context) TimeRangeParams {
	var start, end *time.Time

	startStr := c.Query("start_time")
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = &t
		}
	}

	endStr := c.Query("end_time")
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = &t
		}
	}

	return TimeRangeParams{
		StartTime: start,
		EndTime:   end,
	}
}

func ParseUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func ParseUUIDs(s string) []uuid.UUID {
	if s == "" {
		return nil
	}
	var ids []uuid.UUID
	for _, v := range splitString(s, ",") {
		if id := ParseUUID(v); id != uuid.Nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func ParseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}

func parseIntDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

func parseFloatDefault(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func GetTenantID(c *gin.Context) uuid.UUID {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		if v, exists := c.Get("tenant_id"); exists {
			if id, ok := v.(string); ok {
				return ParseUUID(id)
			}
		}
		return uuid.Nil
	}
	return ParseUUID(tenantID)
}

func GetUserID(c *gin.Context) uuid.UUID {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		if v, exists := c.Get("user_id"); exists {
			if id, ok := v.(string); ok {
				return ParseUUID(id)
			}
		}
		return uuid.Nil
	}
	return ParseUUID(userID)
}

func GetRole(c *gin.Context) string {
	role := c.GetHeader("X-Role")
	if role == "" {
		if v, exists := c.Get("role"); exists {
			if r, ok := v.(string); ok {
				return r
			}
		}
		return ""
	}
	return role
}
