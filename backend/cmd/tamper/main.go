// Команда tamper — демо-инструмент подделки в обход системы (AD-28 «make
// tamper», FR-74, UJ-5, сценарий S09): отдельным подключением
// суперпользователя БД правит таблицы напрямую, как администратор БД/ОС.
// Только профили fixtures и demo (AD-26); в prod отказывает.
//
//	tamper -attack 1  — правка записи журнала → верификатор: «звено не сходится», номер записи;
//	tamper -attack 2  — правка и пересчёт цепочки → расхождение с контрольной точкой хранителя;
//	tamper -attack 3  — правка проекции сдерживания → «проекция расходится с журналом … (CA-…)»;
//	tamper -api URL   — та же попытка через API: операций правки записи и CA нет — отказ.
//
// Обнаруживает подделку не этот инструмент, а независимый верификатор
// (make tamper запускает его следом); индикатор на столах загорается, когда
// ant заберёт отчёт у хранителя.
//
// Слой: точка входа (cmd/*). Владелец: эпик 29 (доверие).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"ant/cmd/internal/config"
	"ant/cmd/internal/db"
	"ant/internal/infrastructure/security/atrest"
	storagesecurity "ant/internal/infrastructure/storage/security"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tamper", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", envOr("ANT_CONFIG", "/etc/ant/ant.yaml"), "ant.yaml: подключение к БД и профиль")
	attack := fs.String("attack", "", "атака: 1 | 2 | 3 | all (пусто — только -api)")
	target := fs.String("target", "", "атаки 1, 2: event_id или seq:N (пусто — последний результат контроля)")
	item := fs.String("item", "", "атака 3: изделие (пусто — первое заблокированное)")
	change := fs.String("change", "", "правка путь=значение через запятую (data/outcome=no_defect_indicated)")
	kekFile := fs.String("kek", "", "KEK (по умолчанию security.kek_file): перешифровать правленый блок")
	api := fs.String("api", "", "адрес ant (http://ant:8080): та же попытка через API")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load(*cfgPath, os.LookupEnv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if cfg.Profile != config.ProfileFixtures && cfg.Profile != config.ProfileDemo {
		_, _ = fmt.Fprintf(stderr, "tamper: профиль %s — демо-инструмента нет (только fixtures и demo, AD-26)\n", cfg.Profile)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	code := 0
	if *attack != "" {
		if err := attacks(ctx, cfg, *attack, *target, *item, *change, *kekFile, stdout); err != nil {
			_, _ = fmt.Fprintln(stderr, "tamper:", err)
			code = 1
		}
	}
	if *api != "" {
		viaAPI(ctx, strings.TrimRight(*api, "/"), stdout)
	}
	return code
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func parseChange(s string) map[string]any {
	if s == "" {
		return nil
	}
	out := map[string]any{}
	for kv := range strings.SplitSeq(s, ",") {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case v == "true" || v == "false":
			out[k] = v == "true"
		case v == "[]":
			out[k] = []any{}
		default:
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				out[k] = n
			} else {
				out[k] = v
			}
		}
	}
	return out
}

func attacks(ctx context.Context, cfg *config.Config, which, target, item, change, kekFile string, out io.Writer) error {
	pc, err := db.Config(cfg.DB, "ant-tamper")
	if err != nil {
		return err
	}
	// Суперпользователь БД без SET ROLE: так действует администратор БД (AD-34).
	conn, err := pgx.ConnectConfig(ctx, pc.ConnConfig)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(context.Background()) }()
	if kekFile == "" {
		kekFile = cfg.Security.KEKFile
	}
	t := &storagesecurity.Tamperer{Conn: conn, Profile: cfg.Profile}
	if k, err := atrest.Load(kekFile); err == nil {
		t.KEK = k
	}
	ch := parseChange(change)
	list := []string{which}
	t1, t2 := target, target
	if which == "all" {
		// Атака 2 — по более ранней записи, затем атака 1 — по более поздней:
		// пересчёт цепочки после атаки 1 «залечил» бы её звено.
		list = []string{"2", "1", "3"}
		if target == "" {
			ts, err := t.Targets(ctx, 2)
			if err != nil {
				return err
			}
			if len(ts) < 2 {
				return fmt.Errorf("в журнале меньше двух фактов изделий — сначала прогон сценария или приём")
			}
			t1, t2 = ts[0], ts[1]
		}
	}
	for _, a := range list {
		var r storagesecurity.Result
		var err error
		switch a {
		case "1":
			r, err = t.UpdateInPlace(ctx, t1, ch)
		case "2":
			r, err = t.UpdateAndRechain(ctx, t2, ch)
		case "3":
			r, err = t.ProjectionUpdate(ctx, item, ch)
		default:
			return fmt.Errorf("атака %q: допустимы 1, 2, 3, all", a)
		}
		if err != nil {
			return fmt.Errorf("атака %s: %w", a, err)
		}
		report(out, r)
	}
	return nil
}

func report(out io.Writer, r storagesecurity.Result) {
	switch r.Attack {
	case "update_in_place":
		_, _ = fmt.Fprintf(out, "Атака 1 — правка записи журнала в обход системы: seq %d (%s, изделие %s): %s. Звено не пересчитано.\n", r.Seq, r.Type, r.ItemID, r.Detail)
	case "update_and_rechain":
		_, _ = fmt.Fprintf(out, "Атака 2 — правка записи и пересчёт цепочки суперпользователем: seq %d (%s, изделие %s): %s; пересчитано звеньев: %d.\n", r.Seq, r.Type, r.ItemID, r.Detail, r.Relinked)
	case "projection_update":
		_, _ = fmt.Fprintf(out, "Атака 3 — правка проекции сдерживания изделия %s: %s. Журнал не тронут.\n", r.ItemID, r.Detail)
	}
}

// viaAPI — та же попытка через API: правки и удаления записей журнала и
// записи в журнал критических действий в контракте нет (AD-2, AD-28) — отказ.
func viaAPI(ctx context.Context, base string, out io.Writer) {
	c := http.Client{Timeout: 10 * time.Second}
	tries := []struct{ method, path, what string }{
		{http.MethodPut, "/api/v1/journal/1", "переписать запись журнала"},
		{http.MethodDelete, "/api/v1/journal/1", "удалить запись журнала"},
		{http.MethodPost, "/api/v1/critical-actions", "вписать запись в журнал критических действий"},
		{http.MethodDelete, "/api/v1/critical-actions/CA-1", "удалить критическое действие"},
	}
	for _, tr := range tries {
		rq, _ := http.NewRequestWithContext(ctx, tr.method, base+tr.path, strings.NewReader(`{"containment":"none"}`))
		rq.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(rq)
		if err != nil {
			_, _ = fmt.Fprintf(out, "API: %s %s — нет связи: %v\n", tr.method, tr.path, err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		_ = resp.Body.Close()
		verdict := "ОТКАЗ"
		if resp.StatusCode < 400 {
			verdict = "ПРИНЯТО — так быть не должно"
		}
		_, _ = fmt.Fprintf(out, "API: попытка «%s» (%s %s) — %s: %d %s\n", tr.what, tr.method, tr.path, verdict, resp.StatusCode, oneLine(string(body)))
	}
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 160 {
		return string(r[:160]) + "…"
	}
	return s
}
