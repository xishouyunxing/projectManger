package router

import (
	"crane-system/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupRouterServesFrontendIndexForSPARoutes(t *testing.T) {
	ginMode := os.Getenv("GIN_MODE")
	t.Cleanup(func() {
		if ginMode == "" {
			os.Unsetenv("GIN_MODE")
			return
		}
		os.Setenv("GIN_MODE", ginMode)
	})
	os.Setenv("GIN_MODE", "test")

	frontendDist := t.TempDir()
	indexPath := filepath.Join(frontendDist, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>app</body></html>"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	config.AppConfig = &config.Config{
		App:     config.AppSection{FrontendDist: frontendDist},
		Storage: config.StorageSection{UploadsDir: t.TempDir()},
		CORS:    config.CORSSection{AllowedOrigins: []string{"http://localhost:3000"}},
		Auth: config.AuthSection{
			JWTSecret:       "12345678901234567890123456789012",
			DefaultPassword: "admin123456",
		},
	}

	r := SetupRouter()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); body != "<html><body>app</body></html>" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestSetupRouterKeepsUnknownAPIRoutesAs404(t *testing.T) {
	ginMode := os.Getenv("GIN_MODE")
	t.Cleanup(func() {
		if ginMode == "" {
			os.Unsetenv("GIN_MODE")
			return
		}
		os.Setenv("GIN_MODE", ginMode)
	})
	os.Setenv("GIN_MODE", "test")

	frontendDist := t.TempDir()
	if err := os.WriteFile(filepath.Join(frontendDist, "index.html"), []byte("<html><body>app</body></html>"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	config.AppConfig = &config.Config{
		App:     config.AppSection{FrontendDist: frontendDist},
		Storage: config.StorageSection{UploadsDir: t.TempDir()},
		CORS:    config.CORSSection{AllowedOrigins: []string{"http://localhost:3000"}},
		Auth: config.AuthSection{
			JWTSecret:       "12345678901234567890123456789012",
			DefaultPassword: "admin123456",
		},
	}

	r := SetupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/not-found", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
	}
}
