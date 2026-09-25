package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPAFallbackAndAssets(t *testing.T) {
	files := fstest.MapFS{
		"index.html":      {Data: []byte("<!doctype html><title>ant</title>")},
		"assets/app-1.js": {Data: []byte("console.log(1)")},
		"favicon.svg":     {Data: []byte("<svg/>")},
	}
	h := handler(files)

	cases := []struct {
		path, wantBody string
		wantCode       int
	}{
		{"/", "<!doctype html><title>ant</title>", 200},
		{"/desk/quality", "<!doctype html><title>ant</title>", 200},
		{"/assets/app-1.js", "console.log(1)", 200},
		{"/favicon.svg", "<svg/>", 200},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.wantCode || rec.Body.String() != c.wantBody {
			t.Errorf("%s: код %d, тело %q", c.path, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("%s: нет Content-Security-Policy", c.path)
		}
	}
}

func TestNotBuilt(t *testing.T) {
	h := handler(fstest.MapFS{".gitkeep": {}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("код %d, ждали 503", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := handler(fstest.MapFS{"index.html": {Data: []byte("x")}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("код %d, ждали 405", rec.Code)
	}
}
