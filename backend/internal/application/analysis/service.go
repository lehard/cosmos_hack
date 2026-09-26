package analysis

import (
	"context"
	"encoding/json"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/analysis"
)

// Config — зависимости живой реализации (AD-36, режим live): проекции
// analysis.*, запись решений и доменные часы. Оборудование — из профилей
// выполнения machinelogs в проекции изделия (эпик 23).
type Config struct {
	// Projections — чтение проекций движка (кэш входа гардов, AD-39).
	Projections engineapp.ProjectionStore
	// Decisions — запись решений человека (journal.Append, AD-44).
	Decisions DecisionWriter
	// Clock — доменное «сейчас» при приёме команды (AD-37); nil — системное.
	Clock appjournal.DomainClock
	// Generators — генераторы предложений (FR-63); nil — встроенные на правилах
	// (RuleGenerators) плюс подключённые RegisterGenerator.
	Generators []Generator
	// Line — ограничение линии для генератора «bottleneck»; nil — не подключено.
	Line LineSource
	// Names — названия из справочников (имена людей, значения общих факторов);
	// nil — без названий (author_name, common_factor.label не заполняются).
	Names Names
}

// Service — реализация live ведущих портов модуля analysis (AD-36): чтение —
// над проекциями analysis.* и доменными функциями разбора; команды — гард над
// состоянием инцидента на basis_seq, затем решение в журнал. Без зависимостей
// (NewService) — заглушка: операции отвечают 501.
type Service struct {
	Unimplemented
	cfg Config
}

// NewService создаёт заглушку live (все операции — 501); живую реализацию
// собирает NewLive.
func NewService() *Service { return &Service{} }

// NewLive создаёт живую реализацию над проекциями и журналом.
func NewLive(cfg Config) *Service { return &Service{cfg: cfg} }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

func (s *Service) live() bool { return s.cfg.Projections != nil }

// now — доменное «сейчас» команды (AD-37).
func (s *Service) now(ctx context.Context) (time.Time, error) {
	if s.cfg.Clock == nil {
		return time.Now().UTC(), nil
	}
	return s.cfg.Clock.Now(ctx)
}

func (s *Service) get(ctx context.Context, name, key string, v any) (bool, error) {
	raw, ok, err := s.cfg.Projections.Get(ctx, name, key)
	if err != nil || !ok {
		return false, err
	}
	return true, json.Unmarshal(raw, v)
}

func notFound(object, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", object, "id", id)
	e.Detail = object + " «" + id + "» не найден"
	return e
}

// nc — несоответствие из проекции analysis.nc.
func (s *Service) nc(ctx context.Context, id string) (dom.NCRecord, error) {
	var v dom.NCRecord
	ok, err := s.get(ctx, ProjectionNC, id, &v)
	if err != nil {
		return v, err
	}
	if !ok || v.ItemID == "" {
		return v, notFound("Несоответствие", id)
	}
	return v, nil
}

// item — состояние разбора изделия из проекции analysis.circumstances.
func (s *Service) item(ctx context.Context, id string) (ItemView, error) {
	v := ItemView{ItemID: id}
	_, err := s.get(ctx, ProjectionCircumstances, id, &v)
	return v, err
}

// incident — инцидент из проекции analysis.incident.
func (s *Service) incident(ctx context.Context, id string) (dom.IncidentRecord, error) {
	var v dom.IncidentRecord
	ok, err := s.get(ctx, ProjectionIncident, id, &v)
	if err != nil {
		return v, err
	}
	if !ok || v.IncidentID == "" {
		return v, notFound("Инцидент", id)
	}
	return v, nil
}

func (s *Service) list(ctx context.Context, name string) ([]string, error) {
	var l dom.ListRecord
	_, err := s.get(ctx, name, dom.ListKey, &l)
	return l.IDs, err
}

// analyze — разбор несоответствия: состояние изделия и события оборудования
// из профилей выполнения (FR-58, FR-148, FR-153).
func (s *Service) analyze(ctx context.Context, n dom.NCRecord) (dom.Analysis, ItemView, error) {
	iv, err := s.item(ctx, n.ItemID)
	if err != nil {
		return dom.Analysis{}, iv, err
	}
	a, ok := dom.Analyze(iv.State, n.ItemID, n.NCID, iv.Equipment)
	if !ok {
		return dom.Analysis{}, iv, notFound("Несоответствие", n.NCID)
	}
	return a, iv, nil
}

// profiles — профили всех несоответствий (для групп, общих факторов и
// похожих случаев).
func (s *Service) profiles(ctx context.Context) ([]dom.Profile, map[string]dom.NCRecord, error) {
	ids, err := s.list(ctx, ProjectionNC)
	if err != nil {
		return nil, nil, err
	}
	items := map[string]ItemView{}
	ncs := map[string]dom.NCRecord{}
	var out []dom.Profile
	for _, id := range ids {
		n, err := s.nc(ctx, id)
		if err != nil {
			continue
		}
		iv, ok := items[n.ItemID]
		if !ok {
			if iv, err = s.item(ctx, n.ItemID); err != nil {
				return nil, nil, err
			}
			items[n.ItemID] = iv
		}
		a, ok := dom.Analyze(iv.State, n.ItemID, id, iv.Equipment)
		if !ok {
			continue
		}
		p := a.Profile
		if p.DefectType == "" {
			p.DefectType = n.DefectType
		}
		ncs[id] = n
		out = append(out, p)
	}
	return out, ncs, nil
}
