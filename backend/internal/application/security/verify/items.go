package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/engine"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/security"
)

// items — реакции, покрытие и проекции по каждому изделию (AD-9): журнал
// переигрывается той же свёрткой, что у воркера; каждая пачка реакций
// проверяется свёрткой префикса входа до её basis_seq; проекции состояния
// сравниваются с пересвёрткой.
func (v *run) items(ctx context.Context) error {
	ids := []string{}
	for _, r := range v.idx["main"] {
		if r.itemID != "" && (v.in.RunID == "" || r.runID == v.in.RunID) {
			ids = append(ids, r.itemID)
		}
	}
	fold := v.in.Fold
	if fold == nil {
		fold = engine.Fold
	}
	for _, id := range uniq(ids) {
		if err := v.item(ctx, id, fold); err != nil {
			return err
		}
	}
	return nil
}

// safeFold — свёртка с перехватом паники доменного кода.
func safeFold(fold engine.Folder, b engine.Bundle, in []kernel.Record) (s engine.Snapshot, rs []kernel.Reaction, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("паника свёртки: %v", p)
		}
	}()
	s, rs = fold(b, in)
	for i := range rs {
		// Как у воркера: режим 0 записывается как 1 «только сообщить» (FR-50).
		if rs[i].AutomationMode == 0 {
			rs[i].AutomationMode = 1
		}
	}
	return s, rs, nil
}

func (v *run) item(ctx context.Context, id string, fold engine.Folder) error {
	creac, ccov, cproj := v.checks["reactions"], v.checks["coverage"], v.checks["projections"]
	part := kernel.PartitionOf(id, max(v.in.Partitions, 1))
	cursor, stored, in, err := v.settle(ctx, id, part)
	if err != nil {
		var pe *engineapp.ProcessingError
		if errorsAs(err, &pe) {
			creac.unverifiable("reactions.decode", fmt.Sprintf("изделие %s: запись seq %d не разбирается — %v", id, pe.Seq, pe.Err))
			return nil
		}
		return err
	}
	if in.Stopped {
		ccov.unverifiable("coverage.stopped", fmt.Sprintf("изделие %s: «обработка остановлена» — реакции и проекции не сверяются до повтора", id))
		return nil
	}
	if len(in.Input) == 0 {
		return nil
	}
	// Реакции по basis_seq: группы по основанию, по порядку записи.
	// Реакции, записанные после головы, до которой прочитаны цепочки, в
	// отчёт этой проверки не входят (их basis_seq в индексе ещё нет) — их
	// сверит следующая проверка.
	head := v.lastSeq("main")
	groups := map[int64][]engine.Recorded{}
	for _, r := range in.Recorded {
		if r.Seq > head {
			continue
		}
		b := v.byIDSeq(r.EventID)
		groups[b] = append(groups[b], r)
	}
	for _, basis := range slices.Sorted(maps.Keys(groups)) {
		g := groups[basis]
		creac.checked += len(g)
		if basis == 0 {
			creac.unverifiable("reactions.no_basis", fmt.Sprintf("изделие %s: у %d реакций нет basis_seq", id, len(g)))
			continue
		}
		first := slices.MinFunc(g, func(a, b engine.Recorded) int { return int(a.Seq - b.Seq) }).Seq
		var prefix []kernel.Record
		for _, r := range in.Input {
			if r.Seq <= basis {
				prefix = append(prefix, r)
			}
		}
		var before []engine.Recorded
		for _, r := range in.Recorded {
			if r.Seq < first {
				before = append(before, r)
			}
		}
		b, _, err := v.in.Bundles.Bundle(ctx, id, prefix)
		if err != nil {
			creac.unverifiable("reactions.bundle", fmt.Sprintf("изделие %s: нормативный слой на basis_seq %d не собран — %v", id, basis, err))
			continue
		}
		_, rs, err := safeFold(fold, b, prefix)
		if err != nil {
			creac.reject("reactions.fold", fmt.Sprintf("изделие %s: свёртка входа до basis_seq %d падает — %v", id, basis, err), "main", basis, "")
			continue
		}
		var trig kernel.Record
		if len(prefix) > 0 {
			trig = prefix[len(prefix)-1]
		}
		plan, err := engine.Diff(rs, before, engine.Trigger{EventID: trig.EventID, OccurredAt: trig.OccurredAt})
		if err != nil {
			creac.reject("reactions.diff", fmt.Sprintf("изделие %s, basis_seq %d: %v", id, basis, err), "main", basis, "")
			continue
		}
		for _, r := range g {
			i := slices.IndexFunc(plan, func(p engine.Planned) bool { return p.EventID == r.EventID })
			if i < 0 || plan[i].Reaction.Type != r.Type {
				creac.reject("reactions.unexpected", fmt.Sprintf("реакция seq %d (%s, изделие %s) не следует из журнала: свёртка входа до basis_seq %d её не даёт — вход изменён в обход системы или реакция вписана",
					r.Seq, r.Type, id, basis), "main", r.Seq, "")
				continue
			}
			if plan[i].Change == engine.ChangeWithdrawn {
				continue
			}
			fc, err1 := engine.Fingerprint(plan[i].Reaction)
			fr, err2 := r.Fingerprint()
			if err1 != nil || err2 != nil || fc != fr {
				creac.reject("reactions.content", fmt.Sprintf("реакция seq %d (%s, изделие %s) расходится со свёрткой входа до basis_seq %d — содержимое реакции или её входа изменено",
					r.Seq, r.Type, id, basis), "main", r.Seq, "")
			}
		}
	}
	// Покрытие и проекции — только для изделий, чей вход обработан воркером.
	ccov.checked++
	if cursor < in.Last.Seq {
		ccov.unverifiable("coverage.pending", fmt.Sprintf("изделие %s: вход до seq %d ещё не обработан воркером (курсор %d) — реакции и проекции сверяются после обработки", id, in.Last.Seq, cursor))
		return nil
	}
	b, _, err := v.in.Bundles.Bundle(ctx, id, in.Input)
	if err != nil {
		ccov.unverifiable("coverage.bundle", fmt.Sprintf("изделие %s: нормативный слой не собран — %v", id, err))
		return nil
	}
	snap, rs, err := safeFold(fold, b, in.Input)
	if err != nil {
		ccov.reject("coverage.fold", fmt.Sprintf("изделие %s: свёртка падает — %v", id, err), "main", in.Last.Seq, "")
		return nil
	}
	plan, err := engine.Diff(rs, in.Recorded, engine.Trigger{EventID: in.Last.EventID, OccurredAt: in.Last.OccurredAt})
	if err == nil {
		for _, p := range plan {
			ccov.reject("coverage.missing", fmt.Sprintf("изделие %s: реакция %s (%s) следует из журнала до seq %d, но не записана", id, p.Reaction.Type, p.Change, in.Last.Seq), "main", in.Last.Seq, "")
		}
	}
	return v.projections(id, cproj, snap, rs, in, stored)
}

// settleAttempts — сколько раз снимок изделия перечитывается, пока воркер
// не догонит его вход (изделие в потоке прогона); не догнал — «вход ещё не
// обработан» (не проверяемо, не нарушение).
const settleAttempts = 5

// defaultSettleWait — пауза между попытками снимка по умолчанию.
const defaultSettleWait = 150 * time.Millisecond

// settle — согласованный снимок изделия без общей транзакции (AD-9, AD-45):
// курсор воркера партиции → проекции изделия → вход изделия, строго в этом
// порядке. Воркер пишет проекции и курсор в транзакциях Append (курсор — в
// последней транзакции пачки), поэтому:
//   - проекции прочитаны после курсора — в них есть обработка всего входа
//     до курсора;
//   - вход прочитан после проекций — в нём есть всё, что успели отразить
//     проекции (они не бывают «новее» входа).
//
// Значит, при последней записи входа не дальше курсора проекции ровно
// отражают прочитанный вход. Иначе (изделие обрабатывается прямо сейчас) —
// снимок перечитывается. Прежний порядок «вход → курсор → проекции» во время
// живого прогона давал ложное «проекция расходится с журналом»: воркер
// успевал свернуть новую запись между чтением входа и проекций (basis_seq в
// проекции новее, чем во входе).
func (v *run) settle(ctx context.Context, id string, part int) (cursor int64, stored map[string][]byte, in engineapp.ItemInput, err error) {
	wait := v.in.SettleWait
	if wait <= 0 {
		wait = defaultSettleWait
	}
	for attempt := 1; ; attempt++ {
		if cursor, err = v.in.Projections.WorkerCursor(ctx, part); err != nil {
			return
		}
		if stored, err = v.in.Projections.ItemRows(ctx, id); err != nil {
			return
		}
		if in, err = v.in.Codec.LoadItem(ctx, id, 0); err != nil {
			return
		}
		if in.Stopped || len(in.Input) == 0 || in.Last.Seq <= cursor || attempt >= settleAttempts {
			return
		}
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			err = ctx.Err()
			return
		case <-t.C:
		}
	}
}

func (v *run) byIDSeq(eventID string) int64 {
	if r, ok := v.byID[eventID]; ok {
		return r.basisSeq
	}
	return 0
}

// projections — проекции состояния изделия против пересвёртки (AD-9, AD-28).
// stored — проекции из того же согласованного снимка, что и вход (settle).
func (v *run) projections(id string, c *check, snap engine.Snapshot, rs []kernel.Reaction, in engineapp.ItemInput, stored map[string][]byte) error {
	effects, err := v.in.Registry.ItemEffects(id, snap, rs, in.Input)
	if err != nil {
		c.unverifiable("projections.view", fmt.Sprintf("изделие %s: %v", id, err))
		return nil
	}
	for _, e := range effects {
		switch x := e.(type) {
		case engineapp.ProjectionPut:
			c.checked++
			got, ok := stored[x.Name]
			if !ok {
				c.reject("projection_mismatch.missing", fmt.Sprintf("проекция «%s» изделия %s отсутствует, а журнал её даёт", x.Name, id), "", 0, "")
				continue
			}
			if !sameJSON(got, x.Value) {
				v.mismatch(c, id, x.Name, x.Value, got)
			}
		case engineapp.ProjectionReset:
			if _, ok := stored[x.Name]; ok {
				c.checked++
				c.reject("projection_mismatch.extra", fmt.Sprintf("проекция «%s» изделия %s есть, хотя журнал её не даёт — вписана в обход системы", x.Name, id), "", 0, "")
			}
		}
	}
	return nil
}

func sameJSON(a, b []byte) bool {
	ca, err1 := dj.Canonical(a)
	cb, err2 := dj.Canonical(b)
	return err1 == nil && err2 == nil && bytes.Equal(ca, cb)
}

// containmentTypes — записи, меняющие ось «сдерживание» (для ссылки на CA).
var containmentTypes = []string{string(catalog.DecisionContainmentSet), string(catalog.DecisionContainmentApplied),
	string(catalog.DecisionContainmentReleased), string(catalog.DecisionItemIsolated)}

// mismatch — находка «проекция расходится с журналом»; для сдерживания —
// как в AD-28: «изделие X в журнале „…“ (CA-…), в проекции „…“».
func (v *run) mismatch(c *check, id, name string, want, got []byte) {
	var w, g map[string]any
	_ = json.Unmarshal(want, &w)
	_ = json.Unmarshal(got, &g)
	if cw, ok := w["containment"].(string); ok {
		if cg, _ := g["containment"].(string); cg != cw {
			ref := v.lastCA("item:"+id, containmentTypes)
			refText := ""
			if ref != "" {
				refText = " (" + ref + ")"
			}
			c.reject("projection_mismatch", fmt.Sprintf("проекция расходится с журналом: изделие %s в журнале «%s»%s, в проекции «%s»",
				id, containmentLabel(cw), refText, containmentLabel(cg)), "", 0, ref)
			return
		}
	}
	field := ""
	for _, k := range slices.Sorted(maps.Keys(w)) {
		a, _ := json.Marshal(w[k])
		b, _ := json.Marshal(g[k])
		if !bytes.Equal(a, b) {
			field = fmt.Sprintf(": поле %s — в журнале %s, в проекции %s", k, short(a), short(b))
			break
		}
	}
	c.reject("projection_mismatch", fmt.Sprintf("проекция расходится с журналом: «%s» изделия %s%s", name, id, field), "", 0, "")
}

func short(b []byte) string {
	r := []rune(string(b))
	if len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return string(r)
}

// containmentLabel — «заблокировано» / «разрешено» с уровнем по словарю статусов.
func containmentLabel(code string) string {
	l := code
	for _, d := range statuses.Axes {
		if d.Name == "containment" {
			for _, x := range d.Values {
				if x.Code == code {
					l = x.Label
				}
			}
		}
	}
	switch code {
	case "item_hold", "lot_hold":
		return "заблокировано: " + l
	case "none", "":
		return "разрешено"
	}
	return l
}

// lastCA — последняя запись CA по объекту (и типам основной записи).
func (v *run) lastCA(object string, types []string) string {
	for i := len(v.ca) - 1; i >= 0; i-- {
		a := v.ca[i]
		if a.r.ObjectRef == object && (types == nil || slices.Contains(types, a.r.ActionType)) {
			return dom.Ref(a.seq)
		}
	}
	return ""
}
