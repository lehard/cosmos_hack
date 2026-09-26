// Пакет webui — раздача собранного фронтенда, встроенного в бинарник ant
// (AD-25: «фронтенд встроен в бинарник»; NFR-SEC-1: все ресурсы локальные).
//
// Слой: infrastructure/transport. Не импортирует другие зоны; подключается в
// cmd/ant как обработчик «всё, что не API».
//
// Каталог dist заполняет сборка образа (deploy/Dockerfile копирует туда
// frontend/dist). В репозитории лежит только .gitkeep — тогда обработчик
// отвечает 503 с пояснением, а API продолжает работать.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// contentSecurityPolicy — только собственные ресурсы, без внешних адресов
// (закрытый контур). 'unsafe-inline' для стилей нужен Naive UI (CSS-in-JS).
// 'wasm-unsafe-eval' — пакет подписи signer.wasm для ключа в хранилище
// страницы (Д-72, вариант page: планшет, телефон без расширений); сам код
// WASM — только свой (/signer/signer.wasm из той же сборки).
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' 'wasm-unsafe-eval'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'none'; " +
	"form-action 'self'"

// Handler раздаёт статические файлы; неизвестные пути получают index.html
// (история маршрутов SPA). Пути /api/ сюда не должны попадать — их
// регистрирует транспорт API раньше.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // каталог встроен при компиляции — ошибки быть не может
	}
	return handler(sub)
}

func handler(files fs.FS) http.Handler {
	fileServer := http.FileServerFS(files)
	_, statErr := fs.Stat(files, "index.html")
	built := statErr == nil

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			h.Set("Allow", "GET, HEAD")
			http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}
		if !built {
			h.Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Интерфейс не встроен в эту сборку: соберите образ (make build) — API работает.\n"))
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if st, err := fs.Stat(files, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Имена файлов Vite содержат хеш содержимого.
					h.Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		h.Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, files, "index.html")
	})
}
