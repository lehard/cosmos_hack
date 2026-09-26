package security

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	sim "ant/internal/domain/simulation"
	"ant/internal/infrastructure/security/atrest"
)

// Tamperer — подделка в обход системы (AD-28 «make tamper», S09, UJ-5):
// отдельное подключение суперпользователя БД правит таблицы напрямую —
// так действует администратор БД/ОС (AD-34). Три атаки:
//
//  1. update_in_place — правка содержимого записи журнала (и её commit в
//     заголовке, как сделал бы аккуратный злоумышленник), звено не тронуто →
//     верификатор: «звено не сходится» с номером записи;
//  2. update_and_rechain — та же правка и пересчёт всех звеньев до головы
//     суперпользователем → цепочка внутри сходится, но расходится с
//     контрольными точками и звеньями хранителя;
//  3. projection_update — правка проекции сдерживания («заблокировано» →
//     «разрешено»), журнал не тронут → «проекция расходится с журналом … (CA-…)».
//
// Только профили fixtures и demo (демо-инструмент, AD-26); реализует порт
// application/simulation.Tamperer для кнопки подделки. С KEK (администратор
// ОС в демо его видит, AD-23) содержимое перешифровывается тем же DEK.
type Tamperer struct {
	// Conn — подключение суперпользователя БД (не роль ant_app).
	Conn *pgx.Conn
	// KEK — ключ шифрования ключей; nil — блоки записей открыты.
	KEK *atrest.KEK
	// Profile — профиль стенда: подделка только в fixtures и demo.
	Profile string
}

// Result — что подделано.
type Result struct {
	Attack  string
	Chain   string
	Seq     int64
	EventID string
	Type    string
	ItemID  string
	Detail  string
	// Relinked — сколько звеньев пересчитано (атака 2).
	Relinked int
}

// ErrProfile — подделка вне демо-профилей.
var ErrProfile = errors.New("tamper: только профили fixtures и demo (AD-26)")

func (t *Tamperer) allowed() error {
	if t.Profile != "fixtures" && t.Profile != "demo" {
		return fmt.Errorf("%w: профиль %q", ErrProfile, t.Profile)
	}
	return nil
}

// Apply — порт application/simulation.Tamperer: шаг tamper сценария (S09).
func (t *Tamperer) Apply(ctx context.Context, _ string, tp sim.Tamper, eventID string) error {
	var err error
	switch tp.Kind {
	case "update_in_place":
		_, err = t.UpdateInPlace(ctx, eventID, tp.Change)
	case "update_and_rechain":
		_, err = t.UpdateAndRechain(ctx, eventID, tp.Change)
	case "projection_update":
		item := strings.TrimPrefix(cmpOr(eventID, tp.Target), "item:")
		_, err = t.ProjectionUpdate(ctx, item, tp.Change)
	default:
		err = fmt.Errorf("tamper: неизвестная атака %q", tp.Kind)
	}
	return err
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// row — запись журнала как её видит суперпользователь.
type row struct {
	seq      int64
	header   string
	commit   []byte
	link     []byte
	salt     []byte
	envelope []byte
	dekID    *string
	nonce    []byte
}

// Target — выбор записи: event_id, «seq:N» или пусто — последний результат
// контроля с признаками дефекта (правка «брак → годно», UJ-5), иначе
// последний факт изделия.
func (t *Tamperer) target(ctx context.Context, tx pgx.Tx, target string) (int64, error) {
	var seq int64
	var err error
	switch {
	case strings.HasPrefix(target, "seq:"):
		seq, err = strconv.ParseInt(strings.TrimPrefix(target, "seq:"), 10, 64)
	case target != "":
		err = tx.QueryRow(ctx, "SELECT seq FROM journal.entries WHERE chain = 'main' AND event_id = $1", target).Scan(&seq)
	default:
		err = tx.QueryRow(ctx, `SELECT seq FROM journal.entries WHERE chain = 'main' AND item_id IS NOT NULL AND entry_kind = 'fact'
ORDER BY (event_type = 'inspection.result.recorded') DESC, seq DESC LIMIT 1`).Scan(&seq)
	}
	if err != nil {
		return 0, fmt.Errorf("tamper: запись %q не найдена: %w", target, err)
	}
	return seq, nil
}

func (t *Tamperer) read(ctx context.Context, tx pgx.Tx, seq int64) (row, error) {
	r := row{seq: seq}
	err := tx.QueryRow(ctx, "SELECT header, commit, link, salt, envelope, dek_id, nonce FROM journal.entries WHERE chain = 'main' AND seq = $1", seq).
		Scan(&r.header, &r.commit, &r.link, &r.salt, &r.envelope, &r.dekID, &r.nonce)
	return r, err
}

// block — соль и конверт записи (с KEK — расшифровка).
func (t *Tamperer) block(ctx context.Context, tx pgx.Tx, r row) (salt, env, dek []byte, err error) {
	if r.dekID == nil {
		return r.salt, r.envelope, nil, nil
	}
	if t.KEK == nil {
		return nil, nil, nil, errors.New("tamper: запись зашифрована, KEK не указан (-kek): без него правится только заголовок")
	}
	var wrapped []byte
	if err := tx.QueryRow(ctx, "SELECT wrapped FROM journal.dek_wraps WHERE dek_id = $1 AND kek_id = $2", *r.dekID, t.KEK.KEKID()).Scan(&wrapped); err != nil {
		return nil, nil, nil, err
	}
	if dek, err = t.KEK.Unwrap(*r.dekID, wrapped); err != nil {
		return nil, nil, nil, err
	}
	commit, _ := dj.DigestFromBytes(r.commit)
	pb, err := t.KEK.Open(dek, r.nonce, r.envelope, dj.AAD("main", r.seq, commit))
	if err != nil {
		return nil, nil, nil, err
	}
	salt, env, err = dj.ParsePlainBlock(pb)
	return salt, env, dek, err
}

// edit — правка payload события по путям change («data/outcome»: …); без
// change — результат контроля становится «признаков дефекта нет».
func edit(env []byte, change map[string]any) ([]byte, string, error) {
	var d map[string]any
	if err := json.Unmarshal(env, &d); err != nil {
		return nil, "", err
	}
	pl, _ := d["payload"].(string)
	raw, err := base64.StdEncoding.DecodeString(pl)
	if err != nil {
		return nil, "", err
	}
	var ev map[string]any
	if err := json.Unmarshal(raw, &ev); err != nil {
		return nil, "", err
	}
	if len(change) == 0 {
		if ev["event_type"] == "inspection.result.recorded" {
			change = map[string]any{"data/outcome": "no_defect_indicated", "data/defects": []any{}}
		} else {
			change = map[string]any{"data/tampered": true}
		}
	}
	var what []string
	for path, val := range change {
		m := ev
		parts := strings.Split(path, "/")
		for _, p := range parts[:len(parts)-1] {
			next, ok := m[p].(map[string]any)
			if !ok {
				next = map[string]any{}
				m[p] = next
			}
			m = next
		}
		was, _ := json.Marshal(m[parts[len(parts)-1]])
		m[parts[len(parts)-1]] = val
		now, _ := json.Marshal(val)
		what = append(what, fmt.Sprintf("%s: %s → %s", path, was, now))
	}
	raw, err = json.Marshal(ev)
	if err != nil {
		return nil, "", err
	}
	if raw, err = dj.Canonical(raw); err != nil {
		return nil, "", err
	}
	d["payload"] = base64.StdEncoding.EncodeToString(raw)
	out, err := json.Marshal(d)
	if err != nil {
		return nil, "", err
	}
	out, err = dj.Canonical(out)
	return out, strings.Join(what, "; "), err
}

// unlock — суперпользователь выключает триггеры запрета изменения журнала
// на время своей транзакции (ALWAYS-триггеры выключает только владелец или
// суперпользователь — это и есть обход системы, AD-2).
func unlock(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "ALTER TABLE journal.entries DISABLE TRIGGER entries_forbid_change")
	return err
}

func relock(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "ALTER TABLE journal.entries ENABLE ALWAYS TRIGGER entries_forbid_change")
	return err
}

// rewrite — правка содержимого записи seq: новый конверт, commit заголовка
// и столбца (с той же солью), блок перешифрован. Возвращает новый commit и
// заголовок.
func (t *Tamperer) rewrite(ctx context.Context, tx pgx.Tx, seq int64, change map[string]any) (Result, jc.JournalEntry, error) {
	r, err := t.read(ctx, tx, seq)
	if err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	salt, env, dek, err := t.block(ctx, tx, r)
	if err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	newEnv, what, err := edit(env, change)
	if err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	var e jc.JournalEntry
	if err := json.Unmarshal([]byte(r.header), &e); err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	commit, err := dj.Commit(salt, newEnv)
	if err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	e.Commit = commit.String()
	header, err := dj.OpenFields(e)
	if err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	stored, nonce := newEnv, r.nonce
	if r.dekID != nil {
		pb, err := dj.PlainBlock(salt, newEnv)
		if err != nil {
			return Result{}, jc.JournalEntry{}, err
		}
		if nonce, stored, err = t.KEK.Seal(dek, pb, dj.AAD("main", seq, commit)); err != nil {
			return Result{}, jc.JournalEntry{}, err
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE journal.entries SET header = $1, commit = $2, envelope = $3, nonce = $4 WHERE chain = 'main' AND seq = $5",
		string(header), commit.Bytes(), stored, nonce, seq); err != nil {
		return Result{}, jc.JournalEntry{}, err
	}
	res := Result{Chain: "main", Seq: seq, EventID: e.EventID, Type: e.EventType, Detail: what}
	if e.ItemID != nil {
		res.ItemID = *e.ItemID
	}
	return res, e, nil
}

// UpdateInPlace — атака 1: правка записи журнала в обход системы.
func (t *Tamperer) UpdateInPlace(ctx context.Context, target string, change map[string]any) (Result, error) {
	if err := t.allowed(); err != nil {
		return Result{}, err
	}
	var res Result
	err := pgx.BeginTxFunc(ctx, t.Conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		seq, err := t.target(ctx, tx, target)
		if err != nil {
			return err
		}
		if err := unlock(ctx, tx); err != nil {
			return err
		}
		if res, _, err = t.rewrite(ctx, tx, seq, change); err != nil {
			return err
		}
		return relock(ctx, tx)
	})
	res.Attack = "update_in_place"
	return res, err
}

// UpdateAndRechain — атака 2: правка записи и пересчёт всех звеньев основной
// цепочки до головы («администратор БД»).
func (t *Tamperer) UpdateAndRechain(ctx context.Context, target string, change map[string]any) (Result, error) {
	if err := t.allowed(); err != nil {
		return Result{}, err
	}
	var res Result
	err := pgx.BeginTxFunc(ctx, t.Conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		seq, err := t.target(ctx, tx, target)
		if err != nil {
			return err
		}
		if err := unlock(ctx, tx); err != nil {
			return err
		}
		if res, _, err = t.rewrite(ctx, tx, seq, change); err != nil {
			return err
		}
		prev := dj.ZeroLink
		if seq > 1 {
			var l []byte
			if err := tx.QueryRow(ctx, "SELECT link FROM journal.entries WHERE chain = 'main' AND seq = $1", seq-1).Scan(&l); err != nil {
				return err
			}
			prev, _ = dj.DigestFromBytes(l)
		}
		rows, err := tx.Query(ctx, "SELECT seq, header, commit FROM journal.entries WHERE chain = 'main' AND seq >= $1 ORDER BY seq", seq)
		if err != nil {
			return err
		}
		type upd struct {
			seq  int64
			link []byte
		}
		var ups []upd
		for rows.Next() {
			var s int64
			var h string
			var c []byte
			if err := rows.Scan(&s, &h, &c); err != nil {
				rows.Close()
				return err
			}
			cd, _ := dj.DigestFromBytes(c)
			l := dj.Link(prev, cd, dj.H([]byte(h)))
			ups = append(ups, upd{s, l.Bytes()})
			prev = l
		}
		rows.Close()
		for _, u := range ups {
			if _, err := tx.Exec(ctx, "UPDATE journal.entries SET link = $1 WHERE chain = 'main' AND seq = $2", u.link, u.seq); err != nil {
				return err
			}
		}
		res.Relinked = len(ups)
		return relock(ctx, tx)
	})
	res.Attack = "update_and_rechain"
	return res, err
}

// ProjectionUpdate — атака 3: правка проекции в обход системы, журнал не
// тронут. Без change — сдерживание изделия «заблокировано» → «разрешено»
// (проекция nonconformity.item); изделие пусто — первое заблокированное.
func (t *Tamperer) ProjectionUpdate(ctx context.Context, itemID string, change map[string]any) (Result, error) {
	if err := t.allowed(); err != nil {
		return Result{}, err
	}
	if itemID == "" {
		err := t.Conn.QueryRow(ctx, `SELECT item_id FROM engine.projections WHERE name = 'nonconformity.item'
AND value->>'containment' IN ('item_hold', 'lot_hold') ORDER BY item_id LIMIT 1`).Scan(&itemID)
		if err != nil {
			return Result{}, fmt.Errorf("tamper: заблокированного изделия в проекции сдерживания нет: %w", err)
		}
	}
	res := Result{Attack: "projection_update", ItemID: itemID}
	if len(change) == 0 {
		var was string
		if err := t.Conn.QueryRow(ctx, "SELECT value->>'containment' FROM engine.projections WHERE name = 'nonconformity.item' AND key = $1", itemID).Scan(&was); err != nil {
			return res, fmt.Errorf("tamper: проекции сдерживания изделия %s нет: %w", itemID, err)
		}
		if _, err := t.Conn.Exec(ctx, `UPDATE engine.projections SET value = jsonb_set(value, '{containment}', '"none"')
WHERE name = 'nonconformity.item' AND key = $1`, itemID); err != nil {
			return res, err
		}
		res.Detail = fmt.Sprintf("nonconformity.item/containment: %s → none («разрешено»)", was)
		return res, nil
	}
	var done []string
	for path, val := range change {
		b, _ := json.Marshal(val)
		keys := "{" + strings.ReplaceAll(path, "/", ",") + "}"
		tag, err := t.Conn.Exec(ctx, "UPDATE engine.projections SET value = jsonb_set(value, $1::text[], $2::jsonb) WHERE item_id = $3 AND value #> $1::text[] IS NOT NULL",
			keys, string(b), itemID)
		if err != nil {
			return res, err
		}
		done = append(done, fmt.Sprintf("%s = %s (%d проекций)", path, b, tag.RowsAffected()))
	}
	res.Detail = strings.Join(done, "; ")
	return res, nil
}

// Targets — n последних фактов изделий (результаты контроля первыми), от
// новых к старым: make tamper бьёт атакой 2 по более ранней записи, атакой 1
// — по более поздней (пересчёт цепочки иначе «залечил» бы звено атаки 1).
func (t *Tamperer) Targets(ctx context.Context, n int) ([]string, error) {
	rows, err := t.Conn.Query(ctx, `SELECT seq FROM journal.entries WHERE chain = 'main' AND item_id IS NOT NULL AND entry_kind = 'fact'
ORDER BY (event_type = 'inspection.result.recorded') DESC, seq DESC LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s int64
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, "seq:"+strconv.FormatInt(s, 10))
	}
	return out, rows.Err()
}
