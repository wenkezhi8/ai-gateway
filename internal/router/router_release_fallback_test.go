package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-gateway/internal/config"

	"github.com/gin-gonic/gin"
)

func prepareStaticDirForRouterTest(t *testing.T) string {
	t.Helper()
	staticDir := t.TempDir()
	assetsDir := filepath.Join(staticDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!doctype html><title>spa</title>"), 0o644); err != nil {
		t.Fatalf("write index.html failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "index-test.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write asset failed: %v", err)
	}
	return staticDir
}

func withRouterTestEnv(t *testing.T, staticDir string) {
	t.Helper()
	prevStaticDir, hadStaticDir := os.LookupEnv("STATIC_DIR")
	if err := os.Setenv("STATIC_DIR", staticDir); err != nil {
		t.Fatalf("set STATIC_DIR failed: %v", err)
	}
	prevPprof, hadPprof := os.LookupEnv("PPROF_ENABLED")
	_ = os.Unsetenv("PPROF_ENABLED")
	deferFn := func() {
		if hadStaticDir {
			_ = os.Setenv("STATIC_DIR", prevStaticDir)
		} else {
			_ = os.Unsetenv("STATIC_DIR")
		}
		if hadPprof {
			_ = os.Setenv("PPROF_ENABLED", prevPprof)
		} else {
			_ = os.Unsetenv("PPROF_ENABLED")
		}
	}
	t.Cleanup(deferFn)
}

func newReleaseRouterForTest(t *testing.T) *gin.Engine {
	t.Helper()
	return NewFullWithConfig(&config.Config{
		Server: config.ServerConfig{Mode: "release", Port: "8566"},
	}, &RouterConfig{EnableSwagger: false}, nil, nil, nil)
}

func TestNewFullWithConfig_ReleaseMode_DebugRoutesShouldNotFallBackToSPA(t *testing.T) {
	staticDir := prepareStaticDirForRouterTest(t)
	withRouterTestEnv(t, staticDir)

	r := NewFullWithConfig(&config.Config{
		Server: config.ServerConfig{Mode: "release", Port: "8566"},
	}, &RouterConfig{EnableSwagger: false}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if rec.Body.String() == "<!doctype html><title>spa</title>" {
		t.Fatal("debug route should not fall back to SPA index")
	}
}

func TestNewFullWithConfig_ReleaseMode_DocsShouldServeSPACenter(t *testing.T) {
	staticDir := prepareStaticDirForRouterTest(t)
	withRouterTestEnv(t, staticDir)

	r := NewFullWithConfig(&config.Config{
		Server: config.ServerConfig{Mode: "release", Port: "8566"},
	}, &RouterConfig{EnableSwagger: true}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/docs", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Header().Get("Location") != "" {
		t.Fatalf("docs should not redirect, location=%q", rec.Header().Get("Location"))
	}
	if rec.Body.String() != "<!doctype html><title>spa</title>" {
		t.Fatalf("docs should serve SPA index, body=%s", rec.Body.String())
	}
}

func TestNewFullWithConfig_ReleaseMode_DocsTrailingSlashShouldServeSPACenter(t *testing.T) {
	staticDir := prepareStaticDirForRouterTest(t)
	withRouterTestEnv(t, staticDir)

	r := NewFullWithConfig(&config.Config{
		Server: config.ServerConfig{Mode: "release", Port: "8566"},
	}, &RouterConfig{EnableSwagger: true}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/docs/", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Header().Get("Location") != "" {
		t.Fatalf("docs trailing slash should not redirect, location=%q", rec.Header().Get("Location"))
	}
	if rec.Body.String() != "<!doctype html><title>spa</title>" {
		t.Fatalf("docs trailing slash should serve SPA index, body=%s", rec.Body.String())
	}
}

func TestNewFullWithConfig_ReleaseMode_SwaggerRootRedirectsToIndex(t *testing.T) {
	staticDir := prepareStaticDirForRouterTest(t)
	withRouterTestEnv(t, staticDir)

	r := NewFullWithConfig(&config.Config{
		Server: config.ServerConfig{Mode: "release", Port: "8566"},
	}, &RouterConfig{EnableSwagger: true}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/swagger", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusFound, rec.Body.String())
	}
	if rec.Header().Get("Location") != "/swagger/index.html" {
		t.Fatalf("swagger root location=%q want=%q", rec.Header().Get("Location"), "/swagger/index.html")
	}
}

func TestNewFullWithConfig_ReleaseMode_SwaggerTrailingSlashRedirectsToIndex(t *testing.T) {
	staticDir := prepareStaticDirForRouterTest(t)
	withRouterTestEnv(t, staticDir)

	r := NewFullWithConfig(&config.Config{
		Server: config.ServerConfig{Mode: "release", Port: "8566"},
	}, &RouterConfig{EnableSwagger: true}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/swagger/", http.NoBody)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusFound, rec.Body.String())
	}
	if rec.Header().Get("Location") != "/swagger/index.html" {
		t.Fatalf("swagger trailing slash location=%q want=%q", rec.Header().Get("Location"), "/swagger/index.html")
	}
}
