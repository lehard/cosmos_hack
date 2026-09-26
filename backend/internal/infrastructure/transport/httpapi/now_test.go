package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ant/internal/application/platform"
)

// Ant-Now (AD-37): «сейчас» сервера из Config.Now — на успешном ответе и на
// ответе-ошибке; без Config.Now — реальное время.
func TestNowHeader(t *testing.T) {
	world := time.Date(2026, 9, 23, 10, 45, 0, 0, time.FixedZone("MSK", 3*3600))
	mux := http.NewServeMux()
	api := New(mux, Config{Now: func(context.Context) time.Time { return world }})
	act := platform.Action{ID: "security.integrity.read", Owner: "security"}
	Read(api, Get("/integrity-test", "t", "t"), act, func(_ context.Context, _ *struct{}, _ platform.Moment) (string, error) {
		return "ok", nil
	})
	Read(api, Get("/integrity-fail", "t", "t"), platform.Action{ID: "security.fail.read", Owner: "security"},
		func(_ context.Context, _ *struct{}, _ platform.Moment) (string, error) {
			return "", platform.NotImplemented("security.fail.read")
		})
	for _, path := range []string{"/integrity-test", "/integrity-fail"} {
		rw := httptest.NewRecorder()
		mux.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, Prefix+path, nil))
		if got := rw.Header().Get(NowHeader); got != "2026-09-23T07:45:00.000Z" {
			t.Fatalf("%s (%d): Ant-Now = %q", path, rw.Code, got)
		}
	}
	mux = http.NewServeMux()
	api = New(mux, Config{})
	Read(api, Get("/integrity-test", "t", "t"), act, func(_ context.Context, _ *struct{}, _ platform.Moment) (string, error) {
		return "ok", nil
	})
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, Prefix+"/integrity-test", nil))
	got, err := time.Parse(time.RFC3339, rw.Header().Get(NowHeader))
	if err != nil || time.Since(got) > time.Minute {
		t.Fatalf("реальное время: %q %v", rw.Header().Get(NowHeader), err)
	}
}
