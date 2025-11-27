package regression

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"httpserver/internal/api/routes"
)

func TestLegacyRoutesSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	routes.RegisterLegacyRoutes(router, testLegacyBinder{})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"BenchmarksImport", http.MethodPost, "/api/legacy/benchmarks/manufacturers/import", http.StatusOK},
		{"SimilarityCompare", http.MethodPost, "/api/legacy/similarity/compare", http.StatusCreated},
		{"SimilarityWeightsGet", http.MethodGet, "/api/legacy/similarity/weights", http.StatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("Expected %d, got %d (body=%s)", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

type testLegacyBinder struct{}

func (testLegacyBinder) RegisterLegacy(group *gin.RouterGroup) {
	benchmarks := group.Group("/benchmarks")
	benchmarks.POST("/manufacturers/import", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	similarity := group.Group("/similarity")
	similarity.POST("/compare", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	similarity.GET("/weights", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})
}

