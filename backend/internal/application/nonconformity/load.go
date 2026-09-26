package nonconformity

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
)

// recorded — записанная реакция воркера с метаданными конверта (версия,
// «пересмотрен из-за записи», правило) и временем записи — для блока «анализ
// системы» карточки (все версии вывода, AD-3, FR-32).
type recorded struct {
	engineapp.Decoded
	RecordedAt time.Time
}

// itemView — изделие на момент: вход свёртки, записанные реакции и итог
// свёртки (AD-22: та же свёртка, что у воркера; ничего не пишет).
type itemView struct {
	ItemID string
	RunID  string
	// Input — вход свёртки на момент по порядку журнала.
	Input []kernel.Record
	// Reactions — записанные реакции воркера на момент.
	Reactions []recorded
	Snap      engine.Snapshot
	Computed  []kernel.Reaction
	// BasisSeq — seq последней записи изделия (вход и реакции): basis_seq
	// команд, чтобы проверка AD-39 не считала устаревшим то, что клиент видел.
	BasisSeq int64
	Env      dom.Env
}

// State — состояние модуля nonconformity изделия.
func (v *itemView) State() dom.State { return v.Snap.Nonconformity }

// Upstream — состояния модулей раньше nonconformity (для гарда, AD-40).
func (v *itemView) Upstream() dom.Upstream {
	s := &v.Snap
	return dom.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision, Quality: &s.Quality, Machinelogs: &s.Machinelogs, Documents: &s.Documents}
}

// Record — запись входа по event_id.
func (v *itemView) Record(id string) (kernel.Record, bool) {
	i := slices.IndexFunc(v.Input, func(r kernel.Record) bool { return r.EventID == id })
	if i < 0 {
		return kernel.Record{}, false
	}
	return v.Input[i], true
}

// readAll — все записи по запросу постранично.
func (s *Service) readAll(ctx context.Context, q appjournal.ReadQuery) ([]jc.JournalEntry, error) {
	var out []jc.JournalEntry
	q.Limit = 1000
	for {
		page, err := s.d.Journal.Read(ctx, q)
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
		if len(page) < q.Limit {
			return out, nil
		}
		q.AfterSeq = int64(page[len(page)-1].Seq)
	}
}

// within — запись видна на момент m (AD-22): «как было» — occurred_at ≤ T,
// «что мы знали» — recorded_at ≤ T.
func within(m platform.Moment, occurred, recordedAt time.Time) bool {
	if m.AsOf == nil {
		return true
	}
	t := occurred
	if m.Axis == platform.AxisRecorded {
		t = recordedAt
	}
	return !t.After(*m.AsOf)
}

// loadItem — изделие на момент m: вход из журнала, свёртка движка (AD-5, AD-22).
func (s *Service) loadItem(ctx context.Context, itemID string, m platform.Moment) (*itemView, error) {
	entries, err := s.readAll(ctx, appjournal.ReadQuery{ItemID: itemID})
	if err != nil {
		return nil, err
	}
	v := &itemView{ItemID: itemID}
	for _, e := range entries {
		if e.Chain != "" && e.Chain != jc.JournalEntryChainMain {
			continue
		}
		d, err := s.d.Codec.Decode(ctx, e)
		if err != nil {
			return nil, err
		}
		rec, _ := time.Parse(time.RFC3339Nano, e.RecordedAt)
		if !within(m, d.Record.OccurredAt, rec) {
			continue
		}
		if int64(e.Seq) > v.BasisSeq {
			v.BasisSeq = int64(e.Seq)
		}
		if v.RunID == "" && d.Record.RunID != "" {
			v.RunID = d.Record.RunID
		}
		switch {
		case d.Info.Type == catalog.OpsProcessingFailed || d.Info.Type == catalog.OpsProcessingRetried:
			continue
		case d.Info.Role == engineapp.RoleWorker:
			v.Reactions = append(v.Reactions, recorded{Decoded: d, RecordedAt: rec.UTC()})
			continue
		}
		v.Input = append(v.Input, d.Record)
	}
	if len(v.Input) == 0 {
		e := platform.Fail(errcodes.ApiNotFound, "object", "Изделие", "id", itemID)
		e.Detail = "Изделие " + itemID + " не найдено в журнале"
		return nil, e
	}
	b, _, err := s.d.Bundles.Bundle(ctx, itemID, v.Input)
	if err != nil {
		return nil, err
	}
	if isZeroEnv(b.Nonconformity) {
		b.Nonconformity = s.cfg.Env
	}
	b.Nonconformity.Quality = b.Quality
	b.Nonconformity.Process = b.Process
	v.Env = b.Nonconformity
	if err := func() (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("свёртка изделия %s: %v", itemID, p)
			}
		}()
		v.Snap, v.Computed = s.d.Fold(b, v.Input)
		return nil
	}(); err != nil {
		return nil, err
	}
	return v, nil
}

func isZeroEnv(e dom.Env) bool { return len(e.ClosingPoints) == 0 && !e.DelegatedIncidentRelease }

// indexTypes — записи, по которым изделие попадает в очередь и список
// несоответствий: черновики и регистрации несоответствий, изоляция,
// сдерживание человеком. Предъявления — отдельно (только нерешённые).
var indexTypes = []catalog.Type{
	catalog.DecisionNonconformityDrafted, catalog.DecisionNonconformityRegistered, catalog.DecisionItemIsolated,
	catalog.DecisionContainmentSet, catalog.DecisionNonconformityConfirmed,
}

// indexItems — изделия с несоответствиями, изоляцией или нерешённым
// предъявлением на момент m в прогоне m.RunID (AD-38), отсортированы.
// Предъявления сверяются с решениями по ним без свёртки изделия: в очередь
// попадают только изделия, ждущие решения на точке.
func (s *Service) indexItems(ctx context.Context, m platform.Moment) ([]string, error) {
	set := map[string]bool{}
	for _, t := range indexTypes {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t), RunID: m.RunID, Moment: m})
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			if e.ItemID != nil && *e.ItemID != "" {
				set[*e.ItemID] = true
			}
		}
	}
	type key struct {
		item, step string
		no         int
	}
	pending := map[key]bool{}
	for _, t := range []catalog.Type{catalog.ItemPresentationRecorded, catalog.DecisionPresentationResolved} {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t), RunID: m.RunID, Moment: m})
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			if e.ItemID == nil || *e.ItemID == "" {
				continue
			}
			d, err := s.d.Codec.Decode(ctx, e)
			if err != nil {
				continue
			}
			var x struct {
				StepKey        string `json:"step_key"`
				PresentationNo int    `json:"presentation_no"`
			}
			if json.Unmarshal(d.Record.Data, &x) != nil {
				continue
			}
			k := key{*e.ItemID, x.StepKey, x.PresentationNo}
			if t == catalog.ItemPresentationRecorded {
				if _, done := pending[k]; !done {
					pending[k] = true
				}
			} else {
				pending[k] = false
			}
		}
	}
	for k, open := range pending {
		if open {
			set[k.item] = true
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	slices.Sort(out)
	return out, nil
}

// ncItem — изделие несоответствия: по записям черновика и регистрации
// (nc_id в data), иначе — свёрткой изделий индекса (черновик уже в
// свёртке, реакция воркера ещё не записана).
func (s *Service) ncItem(ctx context.Context, ncID string) (string, error) {
	if v, ok := s.ncItems.Load(ncID); ok {
		return v.(string), nil
	}
	for _, t := range []catalog.Type{catalog.DecisionNonconformityDrafted, catalog.DecisionNonconformityRegistered} {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t)})
		if err != nil {
			return "", err
		}
		for _, e := range es {
			if e.ItemID == nil {
				continue
			}
			d, err := s.d.Codec.Decode(ctx, e)
			if err != nil {
				continue
			}
			var x struct {
				NCID string `json:"nc_id"`
			}
			if json.Unmarshal(d.Record.Data, &x) == nil && x.NCID != "" {
				s.ncItems.Store(x.NCID, *e.ItemID)
			}
		}
	}
	if v, ok := s.ncItems.Load(ncID); ok {
		return v.(string), nil
	}
	items, err := s.indexItems(ctx, platform.Moment{})
	if err != nil {
		return "", err
	}
	for _, it := range items {
		v, err := s.loadItem(ctx, it, platform.Moment{})
		if err != nil {
			continue
		}
		for _, n := range v.State().NCs {
			s.ncItems.Store(n.ID, it)
		}
	}
	if v, ok := s.ncItems.Load(ncID); ok {
		return v.(string), nil
	}
	e := platform.Fail(errcodes.ApiNotFound, "object", "Несоответствие", "id", ncID)
	e.Detail = "Несоответствие " + ncID + " не найдено"
	return "", e
}

// loadNC — изделие несоответствия на момент и само несоответствие.
func (s *Service) loadNC(ctx context.Context, ncID string, m platform.Moment) (*itemView, dom.NC, error) {
	itemID, err := s.ncItem(ctx, ncID)
	if err != nil {
		return nil, dom.NC{}, err
	}
	v, err := s.loadItem(ctx, itemID, m)
	if err != nil {
		return nil, dom.NC{}, err
	}
	n, ok := v.State().NC(ncID)
	if !ok {
		e := platform.Fail(errcodes.ApiNotFound, "object", "Несоответствие", "id", ncID)
		e.Detail = "Несоответствие " + ncID + " на этот момент ещё не зарегистрировано"
		return nil, dom.NC{}, e
	}
	return v, n, nil
}

// concessionBook — книга разрешений на отклонение из журнала на момент m (FR-54).
func (s *Service) concessionBook(ctx context.Context, m platform.Moment) (*dom.ConcessionBook, error) {
	var all []jc.JournalEntry
	for _, t := range []catalog.Type{catalog.DecisionConcessionGranted, catalog.DecisionConcessionRevoked,
		catalog.DecisionDispositionSet, catalog.DecisionPresentationResolved} {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t), Moment: m})
		if err != nil {
			return nil, err
		}
		all = append(all, es...)
	}
	slices.SortFunc(all, func(a, b jc.JournalEntry) int { return a.Seq - b.Seq })
	b := dom.NewConcessionBook()
	for _, e := range all {
		d, err := s.d.Codec.Decode(ctx, e)
		if err != nil {
			return nil, err
		}
		b.Apply(d.Record)
	}
	return b, nil
}

// holdBook — остановки точек процесса из журнала (FR-49): та же функция
// стадии над записями остановок.
func (s *Service) holdBook(ctx context.Context) (dom.StageState, error) {
	var all []jc.JournalEntry
	for _, t := range []catalog.Type{catalog.DecisionProcessHoldSet, catalog.DecisionProcessHoldReleased} {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t)})
		if err != nil {
			return dom.StageState{}, err
		}
		all = append(all, es...)
	}
	slices.SortFunc(all, func(a, b jc.JournalEntry) int { return a.Seq - b.Seq })
	var st dom.StageState
	for _, e := range all {
		d, err := s.d.Codec.Decode(ctx, e)
		if err != nil {
			return st, err
		}
		st, _ = dom.Stage(st, d.Record)
	}
	return st, nil
}
