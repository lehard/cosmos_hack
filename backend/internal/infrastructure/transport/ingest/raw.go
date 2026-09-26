package ingest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	app "ant/internal/application/ingest"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/infrastructure/transport/httpapi"
)

// Пути тонкого обработчика приёма — до объявления операций Huma эпиком 02.
const (
	PathEvents  = "/api/v1/ingest/events"
	PathBatches = "/api/v1/ingest/batches"
)

// MaxRawBody — наибольшее тело сообщения или пачки (16 МиБ).
const MaxRawBody = 16 << 20

// RawHandler — тонкий обработчик приёма сырых тел (AD-20: приём берёт тело как
// сырой JSON и проверяет его теми же схемами) поверх ведущего порта Commands:
//
//	POST /api/v1/ingest/events  — одно сообщение (DSSE или событие в профиле demo);
//	POST /api/v1/ingest/batches — пачка edge-агента {source_id, sent_at, messages[]} (FR-39).
//
// Это не объявление операции Huma: операции ingest.event.submit и
// ingest.batch.submit с x-ant-action объявляет эпик 02 в register.go и
// вызывает те же методы порта; до этого обработчик монтируется в mux роли api
// (mux.Handle(PathEvents, h); mux.Handle(PathBatches, h)) и нужен edge-агенту.
// Ответ на сообщение: 202 — принято (в том числе с флагом), 200 — повтор с
// прежним ответом, problem+json с кодом из contracts/errors.yaml — карантин
// (422/403) и конфликт (409). Ответ на пачку — 200 и итог по каждому сообщению.
func RawHandler(c app.Commands) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+PathEvents, func(w http.ResponseWriter, r *http.Request) {
		body, ok := readBody(w, r)
		if !ok {
			return
		}
		res, err := c.Ingest(r.Context(), body)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		switch res.Outcome {
		case app.OutcomeQuarantined, app.OutcomeConflict:
			writeProblem(w, problemOf(res, r.URL.Path))
		default:
			writeJSON(w, res.Status, res)
		}
	})
	mux.HandleFunc("POST "+PathBatches, func(w http.ResponseWriter, r *http.Request) {
		body, ok := readBody(w, r)
		if !ok {
			return
		}
		res, err := c.IngestBatch(r.Context(), body)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
	return mux
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRawBody))
	if err != nil {
		p := problem(errcodes.ApiValidationFailed, "тело запроса не прочитано: "+err.Error())
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			p.Status = http.StatusRequestEntityTooLarge
		}
		writeProblem(w, p)
		return nil, false
	}
	return b, true
}

// problemOf — problem+json итога карантина или конфликта (FR-28, FR-30).
func problemOf(res app.Result, instance string) *httpapi.Problem {
	p := problem(res.Code, res.Detail)
	p.Status = res.Status
	p.Instance = instance
	p.QuarantineID = res.QuarantineID
	if res.Field != "" {
		p.Violations = []httpapi.Violation{{Code: string(res.Code), Field: res.Field, Message: res.Detail}}
		p.Params = map[string]string{"field": res.Field, "value": res.Value, "event_type": res.EventType, "source_id": res.SourceID, "event_id": res.EventID}
	}
	return p
}

func problem(code errcodes.Code, detail string) *httpapi.Problem {
	info, ok := errcodes.Lookup(code)
	if !ok {
		info = errcodes.Info{Code: code, Status: http.StatusInternalServerError, Title: string(code)}
	}
	return &httpapi.Problem{Type: errcodes.ProblemTypePrefix + string(code), Title: info.Title, Status: info.Status, Detail: detail, Code: string(code)}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := platform.AsError(err); ok {
		p := problem(e.Code, e.Detail)
		p.Params, p.Instance = e.Params, r.URL.Path
		writeProblem(w, p)
		return
	}
	writeProblem(w, problem(errcodes.ApiInternalError, "Внутренняя ошибка приёма"))
}

func writeProblem(w http.ResponseWriter, p *httpapi.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
