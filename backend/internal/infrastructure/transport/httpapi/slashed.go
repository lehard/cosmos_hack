package httpapi

import (
	"net/http"
	"strings"
)

// SlashedIDs — маршрутизация id с «/» (id изделий прогона сценария:
// ENT01:show-is2-20260921-1/I-3CDF7159, AD-38). Клиент подставляет id в путь
// как есть, и «/» внутри id делит его на сегменты — ServeMux не находит
// операцию (/items/{item_id}/passport не совпадает с …/show-…/I-…/passport).
// Если путь не совпал ни с одной операцией, соседние сегменты склеиваются
// в один (через %2F) — первый вариант, который совпал с операцией
// контракта, и обслуживается; PathValue отдаёт id с «/». Запрос, уже
// совпавший с операцией (в том числе с %2F в id), не трогается.
func SlashedIDs(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, Prefix+"/") || matched(mux, r) {
			mux.ServeHTTP(w, r)
			return
		}
		segs := strings.Split(strings.TrimPrefix(r.URL.EscapedPath(), Prefix+"/"), "/")
		// segs[0] — коллекция; id — не короче двух сегментов, после него хотя бы
		// ничего или хвост операции. Сначала — самый короткий хвост id слева.
		for s := 1; s < len(segs)-1; s++ {
			for e := len(segs) - 1; e > s; e-- {
				parts := append(append(append([]string{}, segs[:s]...), strings.Join(segs[s:e+1], "%2F")), segs[e+1:]...)
				r2 := r.Clone(r.Context())
				u := *r.URL
				u.RawPath = Prefix + "/" + strings.Join(parts, "/")
				r2.URL = &u
				if matched(mux, r2) {
					mux.ServeHTTP(w, r2)
					return
				}
			}
		}
		mux.ServeHTTP(w, r)
	})
}

// matched — путь совпал с операцией API, а не с общим ответом «нет в контракте».
func matched(mux *http.ServeMux, r *http.Request) bool {
	_, pattern := mux.Handler(r)
	return pattern != "" && pattern != "/api/" && pattern != "/"
}
