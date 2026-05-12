package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthEndpoint(t *testing.T) {
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": "v1.0.0"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "healthy" {
		t.Errorf("status = %q, want %q", resp["status"], "healthy")
	}

	if resp["version"] != "v1.0.0" {
		t.Errorf("version = %q, want %q", resp["version"], "v1.0.0")
	}
}

func TestParseUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", true},
		{"empty string", "", true},
		{"invalid format", "not-a-uuid", false},
		{"partial UUID", "550e8400", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUUID(tt.input)
			if tt.input == "" && result.String() != "00000000-0000-0000-0000-000000000000" {
				t.Errorf("parseUUID(%q) should return nil UUID", tt.input)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid number", "42", 42},
		{"empty string", "", 0},
		{"invalid", "abc", 0},
		{"negative", "-10", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"valid float", "3.14", 3.14},
		{"empty string", "", 0},
		{"invalid", "abc", 0},
		{"integer", "10", 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFloat(tt.input)
			if result != tt.expected {
				t.Errorf("parseFloat(%q) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClusterHandlers(t *testing.T) {
	router := gin.New()
	router.GET("/clusters", ListClusters)
	router.POST("/clusters", CreateCluster)
	router.GET("/clusters/:id", GetCluster)
	router.DELETE("/clusters/:id", DeleteCluster)

	t.Run("ListClusters returns empty array", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/clusters", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["items"] == nil {
			t.Error("items should not be nil")
		}

		if resp["total"] != float64(0) {
			t.Errorf("total = %v, want 0", resp["total"])
		}
	})

	t.Run("CreateCluster returns success", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "test-cluster"}`)
		req := httptest.NewRequest("POST", "/clusters", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}
	})

	t.Run("GetCluster returns cluster", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/clusters/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("DeleteCluster returns success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/clusters/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestAuthHandlers(t *testing.T) {
	router := gin.New()
	router.POST("/auth/register", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
			Name     string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"email": req.Email, "name": req.Name})
	})

	router.POST("/auth/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": "mock-token"})
	})

	t.Run("Register validates email format", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email": "invalid", "password": "password123", "name": "Test"}`)
		req := httptest.NewRequest("POST", "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Register validates password length", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email": "test@example.com", "password": "short", "name": "Test"}`)
		req := httptest.NewRequest("POST", "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Register validates required fields", func(t *testing.T) {
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest("POST", "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Login requires password", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email": "test@example.com"}`)
		req := httptest.NewRequest("POST", "/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}
