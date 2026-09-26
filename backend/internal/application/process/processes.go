package process

import (
	"context"
	"slices"
	"sync"

	"ant/internal/application/platform"
	dp "ant/internal/domain/process"
)

// Процессы (UI-11, FR-22): версии группируются по процессу — id и имя
// главного исполняемого bpmn:process версии (Definition.Main). Отдельного
// реестра процессов нет: черновик из редактора наследует процесс своей
// основы, потому что редактор не меняет id главного процесса. Основной
// процесс — процесс стартовой версии (SeedVersionID); без параметра процесса
// живая карта и список версий показывают его, как до появления выбора.

// ProcessInfo — процесс версии: id и имя главного bpmn:process.
type ProcessInfo struct {
	ID   string
	Name string
}

var (
	procMu    sync.Mutex
	procCache = map[string]ProcessInfo{}
)

// ProcessOf — процесс версии по её XML (кеш по хешу версии: байты версии
// неизменны, разбор XML на каждый запрос списка не нужен). Не разобралась —
// процесс из одной этой версии (id версии).
func ProcessOf(v VersionRecord) ProcessInfo {
	key := v.Hash
	if key != "" {
		procMu.Lock()
		p, ok := procCache[key]
		procMu.Unlock()
		if ok {
			return p
		}
	}
	p := ProcessInfo{ID: v.ID, Name: v.Label}
	if d, _ := dp.Parse(v.XML); d != nil && d.Main != "" {
		p.ID = d.Main
		if sc := d.Scopes[d.Main]; sc != nil && sc.Name != "" {
			p.Name = sc.Name
		} else {
			p.Name = d.Main
		}
	}
	if key != "" {
		procMu.Lock()
		procCache[key] = p
		procMu.Unlock()
	}
	return p
}

// DefaultProcess — основной процесс: процесс стартовой версии, иначе —
// процесс самой ранней версии. Пусто — версий нет.
func DefaultProcess(vs []VersionRecord) string {
	for _, v := range vs {
		if v.ID == SeedVersionID {
			return ProcessOf(v).ID
		}
	}
	var first *VersionRecord
	for i := range vs {
		if first == nil || vs[i].CreatedAt.Before(first.CreatedAt) {
			first = &vs[i]
		}
	}
	if first == nil {
		return ""
	}
	return ProcessOf(*first).ID
}

// VersionsOf — версии процесса processID в порядке хранилища.
func VersionsOf(vs []VersionRecord, processID string) []VersionRecord {
	out := make([]VersionRecord, 0, len(vs))
	for _, v := range vs {
		if ProcessOf(v).ID == processID {
			out = append(out, v)
		}
	}
	return out
}

// latest — самая поздняя по созданию версия (процесс без действующей).
func latest(vs []VersionRecord) (VersionRecord, bool) {
	if len(vs) == 0 {
		return VersionRecord{}, false
	}
	best := vs[0]
	for _, v := range vs[1:] {
		if v.CreatedAt.After(best.CreatedAt) {
			best = v
		}
	}
	return best, true
}

// processVersions — версии процесса processID (пусто — основной процесс) и его id.
func (s *LiveService) processVersions(ctx context.Context, processID string) ([]VersionRecord, string, error) {
	all, err := s.Library.List(ctx)
	if err != nil {
		return nil, "", err
	}
	if processID == "" {
		processID = DefaultProcess(all)
	}
	vs := VersionsOf(all, processID)
	if len(vs) == 0 {
		return nil, processID, notFound("Процесс", processID)
	}
	return vs, processID, nil
}

// resolve — показываемая версия: заданная versionID (и принадлежащая
// processID, если он задан) или действующая версия процесса, а если
// действующей нет — самая поздняя.
func (s *LiveService) resolve(ctx context.Context, processID, versionID string) (VersionRecord, error) {
	if versionID != "" {
		v, err := s.version(ctx, versionID)
		if err != nil {
			return v, err
		}
		if processID != "" && ProcessOf(v).ID != processID {
			return VersionRecord{}, notFound("Версия процесса "+processID, versionID)
		}
		return v, nil
	}
	vs, pid, err := s.processVersions(ctx, processID)
	if err != nil {
		return VersionRecord{}, err
	}
	if v, ok := Active(vs); ok {
		return v, nil
	}
	if v, ok := latest(vs); ok {
		return v, nil
	}
	return VersionRecord{}, notFound("Процесс", pid)
}

// processStatus — состояние процесса по статусам версий: есть действующая —
// active; ни одна не вводилась (черновики, на утверждении) — draft; все
// введённые выведены — retired.
func processStatus(vs []VersionRecord) string {
	retired := false
	for _, v := range vs {
		switch v.Status {
		case dp.StatusActive:
			return "active"
		case dp.StatusRetired:
			retired = true
		}
	}
	if retired {
		return "retired"
	}
	return "draft"
}

// Processes — список процессов для выбора (process.process.list, UI-11,
// FR-22): действующая версия, число версий, состояние, изделий в работе.
func (s *LiveService) Processes(ctx context.Context, m platform.Moment) (ProcessList, error) {
	if s.Library == nil {
		return ProcessList{}, platform.NotImplemented("process.process.list")
	}
	all, err := s.Library.List(ctx)
	if err != nil {
		return ProcessList{}, err
	}
	inWork := map[string]int{}
	if s.Store != nil {
		views, err := s.items(ctx, m)
		if err != nil {
			return ProcessList{}, err
		}
		for _, v := range views {
			if !v.Completed {
				inWork[v.VersionID]++
			}
		}
	}
	def := DefaultProcess(all)
	var order []string
	byProc := map[string][]VersionRecord{}
	names := map[string]string{}
	for _, v := range all {
		p := ProcessOf(v)
		if _, ok := byProc[p.ID]; !ok {
			order = append(order, p.ID)
		}
		byProc[p.ID] = append(byProc[p.ID], v)
		names[p.ID] = p.Name
	}
	// Основной процесс — первым, остальные — в порядке появления.
	if i := slices.Index(order, def); i > 0 {
		order = append([]string{def}, slices.Delete(order, i, i+1)...)
	}
	out := ProcessList{Items: []ProcessSummary{}}
	for _, pid := range order {
		vs := byProc[pid]
		ps := ProcessSummary{ProcessID: pid, Name: names[pid], Versions: len(vs), Status: processStatus(vs), IsDefault: pid == def}
		if a, ok := Active(vs); ok {
			ps.ActiveVersion = &ProcessVersionRef{VersionID: a.ID, Label: a.Label}
			// Имя — по действующей версии (черновик мог переименовать процесс).
			ps.Name = ProcessOf(a).Name
		}
		for _, v := range vs {
			ps.ItemsInWork += inWork[v.ID]
		}
		out.Items = append(out.Items, ps)
	}
	return out, nil
}
