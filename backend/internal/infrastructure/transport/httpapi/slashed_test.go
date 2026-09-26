package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSlashedIDs — id изделия прогона с «/» доходит до всех операций изделия
// и в сыром виде (клиент подставил id как есть), и с %2F; путь без операции —
// общий ответ «нет в контракте».
func TestSlashedIDs(t *testing.T) {
	mux := http.NewServeMux()
	for _, p := range []string{"GET /api/v1/items/{item_id}/passport", "POST /api/v1/items/{item_id}/movements",
		"POST /api/v1/items/{item_id}/movements/receive", "POST /api/v1/items/{item_id}/interventions/{intervention_id}/close",
		"GET /api/v1/items/lookup", "POST /api/v1/ops/stopped-items/{item_id}/retry"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(p + "|" + r.PathValue("item_id") + "|" + r.PathValue("intervention_id")))
		})
	}
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	h := SlashedIDs(mux)
	const id = "ENT01:show-is2-20260921-1/I-3CDF7159"
	cases := []struct{ method, path, want string }{
		{"GET", "/api/v1/items/ENT01:show-is2-20260921-1/I-3CDF7159/passport", "GET /api/v1/items/{item_id}/passport|" + id + "|"},
		{"GET", "/api/v1/items/ENT01:show-is2-20260921-1%2FI-3CDF7159/passport", "GET /api/v1/items/{item_id}/passport|" + id + "|"},
		{"POST", "/api/v1/items/ENT01:show-is2-20260921-1/I-3CDF7159/movements/receive", "POST /api/v1/items/{item_id}/movements/receive|" + id + "|"},
		{"POST", "/api/v1/items/ENT01:show-is2-20260921-1/I-3CDF7159/movements", "POST /api/v1/items/{item_id}/movements|" + id + "|"},
		{"POST", "/api/v1/items/ENT01:a/b/c/interventions/IV-1/close", "POST /api/v1/items/{item_id}/interventions/{intervention_id}/close|ENT01:a/b/c|IV-1"},
		{"POST", "/api/v1/ops/stopped-items/ENT01:run-1/I-1/retry", "POST /api/v1/ops/stopped-items/{item_id}/retry|ENT01:run-1/I-1|"},
		{"GET", "/api/v1/items/lookup", "GET /api/v1/items/lookup||"},
		{"GET", "/api/v1/items/ENT01:show-is2-20260921-1/I-3CDF7159", ""},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if c.want == "" {
			if rec.Code != http.StatusNotFound {
				t.Errorf("%s %s: %d, ждали 404", c.method, c.path, rec.Code)
			}
			continue
		}
		if rec.Code != http.StatusOK || rec.Body.String() != c.want {
			t.Errorf("%s %s: %d %q, ждали %q", c.method, c.path, rec.Code, rec.Body.String(), c.want)
		}
	}
}
