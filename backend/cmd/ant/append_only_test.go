package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Д-62, AD-28: попытка правки журнала или журнала критических действий через
// API — 405 (api.method_not_allowed), чтение и прочие пути — не задеты.
func TestAppendOnlyPaths(t *testing.T) {
	for _, c := range []struct {
		method, path string
		want         bool
	}{
		{http.MethodPut, "/api/v1/journal/1", true},
		{http.MethodDelete, "/api/v1/journal/1", true},
		{http.MethodPost, "/api/v1/critical-actions", true},
		{http.MethodDelete, "/api/v1/critical-actions/CA-1", true},
		{http.MethodGet, "/api/v1/journal/1", false},
		{http.MethodPut, "/api/v1/journalx", false},
		{http.MethodPost, "/api/v1/unknown", false},
	} {
		if _, got := appendOnly(httptest.NewRequest(c.method, c.path, nil)); got != c.want {
			t.Errorf("%s %s: %v, ожидалось %v", c.method, c.path, got, c.want)
		}
	}
}
