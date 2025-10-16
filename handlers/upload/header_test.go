package upload

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/storage"
)

func TestHeaderEnabledSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Helper to create a request with given header values map and call headerEnabled
	call := func(hdrVals map[string][]string, key string) bool {
		// Build a minimal gin context using httptest
		req := httptest.NewRequest("POST", "/", nil)
		for k, vals := range hdrVals {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		return headerEnabled(c, key)
	}

	// 1) absent header -> false
	if call(nil, "X-Base64") {
		t.Fatal("expected headerEnabled to be false when header is absent")
	}

	// 2) present with empty value -> true
	if !call(map[string][]string{"X-Base64": {""}}, "X-Base64") {
		t.Fatal("expected headerEnabled to be true for empty value (presence-enabled)")
	}

	// 3) explicit false values (case-insensitive) -> false
	for _, v := range []string{"false", "False", "FALSE", "0", "no", "NO"} {
		if call(map[string][]string{"X-Base64": {v}}, "X-Base64") {
			t.Fatalf("expected headerEnabled to be false for disabling value %q", v)
		}
	}

	// 4) explicit true-ish values -> true
	for _, v := range []string{"true", "1", "yes", "enabled"} {
		if !call(map[string][]string{"X-Base64": {v}}, "X-Base64") {
			t.Fatalf("expected headerEnabled to be true for value %q", v)
		}
	}

	// 5) multiple values: if any value enables -> true
	if !call(map[string][]string{"X-Base64": {"false", "", "0"}}, "X-Base64") {
		t.Fatal("expected headerEnabled to be true when at least one value is empty/presence-enabled")
	}

	// 6) multiple values all disabling -> false
	if call(map[string][]string{"X-Base64": {"false", "0"}}, "X-Base64") {
		t.Fatal("expected headerEnabled to be false when all values are explicit disabling tokens")
	}
}

func TestXBurnHeaderSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	call := func(hdrVals map[string][]string, key string) bool {
		req := httptest.NewRequest("POST", "/", nil)
		for k, vals := range hdrVals {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		return headerEnabled(c, key)
	}

	// Absent -> false
	if call(nil, "X-Burn") {
		t.Fatal("expected X-Burn to be false when absent")
	}

	// Empty (presence) -> true
	if !call(map[string][]string{"X-Burn": {""}}, "X-Burn") {
		t.Fatal("expected X-Burn to be true for empty value")
	}

	// Explicit disable tokens
	for _, v := range []string{"false", "0", "no"} {
		if call(map[string][]string{"X-Burn": {v}}, "X-Burn") {
			t.Fatalf("expected X-Burn to be false for %q", v)
		}
	}

	// Explicit enable tokens
	for _, v := range []string{"true", "1", "yes"} {
		if !call(map[string][]string{"X-Burn": {v}}, "X-Burn") {
			t.Fatalf("expected X-Burn to be true for %q", v)
		}
	}
}

func TestParseTTLHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	cfg := &config.Config{
		BufferSize: 1024,
		DefaultTTL: 24 * time.Hour,
	}
	svc := services.NewPasteService(store, cfg)
	h := NewHandler(svc, cfg)

	// helper to call parseTTL with given header
	call := func(val string) (time.Time, error) {
		req := httptest.NewRequest("POST", "/", nil)
		if val != "" {
			req.Header.Set("X-TTL", val)
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		return h.parseTTL(c)
	}

	// valid TTL within range
	tt, err := call("2h")
	if err != nil {
		t.Fatalf("expected valid TTL, got error: %v", err)
	}
	if time.Until(tt) < time.Hour || time.Until(tt) > 3*time.Hour {
		t.Fatalf("unexpected TTL computed: %v", tt)
	}

	// invalid short TTL
	if _, err := call("30m"); err == nil {
		t.Fatalf("expected error for TTL below min, got nil")
	}

	// invalid long TTL
	if _, err := call("8d"); err == nil {
		t.Fatalf("expected error for TTL above max, got nil")
	}

	// invalid format
	if _, err := call("notaduration"); err == nil {
		t.Fatalf("expected error for invalid duration format, got nil")
	}
}

func TestCustomSlugHeaderValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	cfg := &config.Config{
		BufferSize: 1024 * 1024,
		DefaultTTL: 24 * time.Hour,
	}
	svc := services.NewPasteService(store, cfg)
	h := NewHandler(svc, cfg)

	router := gin.New()
	router.POST("/", h.Upload)

	// invalid slug should return 400
	req := httptest.NewRequest("POST", "/", strings.NewReader("hello"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Slug", "BAD!!")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid slug, got %d: %s", w.Code, w.Body.String())
	}
}
