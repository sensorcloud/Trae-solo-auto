package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORS(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("allows CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
		}

		expectedMethods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
		if got := w.Header().Get("Access-Control-Allow-Methods"); got != expectedMethods {
			t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, expectedMethods)
		}
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("OPTIONS status = %d, want %d", w.Code, http.StatusNoContent)
		}
	})
}

func TestRequestID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		if requestID == "" {
			t.Error("request_id should not be empty")
		}
		c.String(http.StatusOK, "ok")
	})

	t.Run("generates request ID if not provided", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Request-ID") == "" {
			t.Error("X-Request-ID header should be set")
		}
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		expectedID := "test-request-id-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", expectedID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if got := w.Header().Get("X-Request-ID"); got != expectedID {
			t.Errorf("X-Request-ID = %q, want %q", got, expectedID)
		}
	})
}

func TestMetrics(t *testing.T) {
	router := gin.New()
	router.Use(Metrics())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("records request metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestTracing(t *testing.T) {
	router := gin.New()
	router.Use(Tracing())
	router.GET("/test", func(c *gin.Context) {
		traceID := c.GetString("trace_id")
		if traceID == "" {
			t.Error("trace_id should not be empty")
		}
		c.String(http.StatusOK, "ok")
	})

	t.Run("sets trace ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Trace-ID") == "" {
			t.Error("X-Trace-ID header should be set")
		}
	})
}

func TestAuthenticate(t *testing.T) {
	secret := "test-secret-key"

	t.Run("rejects missing auth header", func(t *testing.T) {
		router := gin.New()
		router.Use(Authenticate(secret))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("rejects invalid auth format", func(t *testing.T) {
		router := gin.New()
		router.Use(Authenticate(secret))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("accepts valid Bearer token", func(t *testing.T) {
		router := gin.New()
		router.Use(Authenticate(secret))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestRateLimit(t *testing.T) {
	router := gin.New()
	router.Use(RateLimit("localhost:6379", 100, 0))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("allows requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestTenantMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(TenantMiddleware())
	router.GET("/test", func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		c.String(http.StatusOK, tenantID)
	})

	t.Run("sets tenant ID from header", func(t *testing.T) {
		expectedTenant := "test-tenant-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", expectedTenant)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Body.String() != expectedTenant {
			t.Errorf("body = %q, want %q", w.Body.String(), expectedTenant)
		}
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("allows matching role", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "admin")
			c.Next()
		})
		router.Use(RequireRole("admin", "superuser"))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("rejects non-matching role", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "user")
			c.Next()
		})
		router.Use(RequireRole("admin", "superuser"))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects missing role", func(t *testing.T) {
		router := gin.New()
		router.Use(RequireRole("admin"))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}
