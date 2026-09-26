package item

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// Config — зависимости живой реализации (AD-36, режим live).
type Config struct {
	// Codec — чтение потока изделия (вход свёртки и записанные реакции).
	Codec *engineapp.Codec
	// Projections — проекции движка: item.row, item.index, состояние стадии.
	Projections engineapp.ProjectionStore
	// Bundles — нормативный слой изделия (nil — Env модуля item по умолчанию).
	Bundles engineapp.BundleSource
	// Fold — свёртка изделия (nil — engine.Fold).
	Fold engine.Folder
	// Writer — запись фактов и решений (journal.Append, AD-44).
	Writer Writer
	// Clock — доменное «сейчас» при приёме команды (AD-37); nil — системное.
	Clock appjournal.DomainClock
	// Env — номенклатура для гардов и Bundles по умолчанию.
	Env dom.Env
	// Enterprise — код предприятия для новых ID изделий (AD-16); пусто — ENT01.
	Enterprise string
	// ProcessVersion, NormativeRev — версия процесса (хеш BPMN) и ревизия
	// нормативного слоя, закрепляемые при регистрации (AD-17), пока источник
	// версии (эпик 17) не отдаёт их сам.
	ProcessVersion string
	NormativeRev   string
}

// Service — реализация live ведущих портов модуля item (AD-36): чтение — та
// же свёртка изделия, что у воркера, на момент (AD-22), плюс генеалогия
// межизделийной стадии; команды — доменный гард над свёрткой на basis_seq,
// затем запись в журнал с проверками AD-39. Без зависимостей (NewService) —
// заглушка: операции отвечают 501.
type Service struct {
	Unimplemented
	cfg Config
}

// NewService создаёт заглушку live (все операции — 501); живую реализацию
// собирает NewLive.
func NewService() *Service { return &Service{} }

// NewLive создаёт живую реализацию.
func NewLive(cfg Config) *Service {
	if cfg.Fold == nil {
		cfg.Fold = engine.Fold
	}
	if cfg.Bundles == nil {
		cfg.Bundles = Bundles{Env: cfg.Env}
	}
	if cfg.Enterprise == "" {
		cfg.Enterprise = "ENT01"
	}
	return &Service{cfg: cfg}
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

func (s *Service) live() bool { return s.cfg.Codec != nil && s.cfg.Projections != nil }

// now — доменное «сейчас» команды (AD-37).
func (s *Service) now(ctx context.Context) (time.Time, error) {
	if s.cfg.Clock == nil {
		return time.Now().UTC(), nil
	}
	return s.cfg.Clock.Now(ctx)
}

// at — изделие на момент (AD-22): вход целиком, префикс на момент, свёртка.
type at struct {
	In       engineapp.ItemInput
	Prefix   []kernel.Record
	Snap     engine.Snapshot
	Rs       []kernel.Reaction
	Recorded []engine.Recorded
}

func (s *Service) itemAt(ctx context.Context, itemID string, m platform.Moment) (at, error) {
	in, err := s.cfg.Codec.LoadItem(ctx, itemID, 0)
	if err != nil {
		return at{}, err
	}
	if len(in.Input) == 0 {
		return at{}, notFound(itemID)
	}
	dm := engine.Moment{Axis: engine.AxisOccurred}
	if m.Axis == platform.AxisRecorded {
		dm.Axis = engine.AxisRecorded
	}
	if m.AsOf != nil {
		dm.At = *m.AsOf
	}
	prefix := engine.Prefix(in.Input, dm)
	b, _, err := s.cfg.Bundles.Bundle(ctx, itemID, prefix)
	if err != nil {
		return at{}, err
	}
	snap, rs := s.safeFold(b, prefix)
	var rec []engine.Recorded
	for _, r := range in.Recorded {
		if r.Seq <= snap.BasisSeq || m.AsOf == nil {
			rec = append(rec, r)
		}
	}
	return at{In: in, Prefix: engine.SortInput(prefix), Snap: snap, Rs: rs, Recorded: rec}, nil
}

// safeFold — свёртка без паники на битой записи (паспорт показывает то, что
// удалось свернуть; воркер отмечает «обработка остановлена», AD-45).
func (s *Service) safeFold(b engine.Bundle, in []kernel.Record) (snap engine.Snapshot, rs []kernel.Reaction) {
	defer func() {
		if recover() != nil {
			snap, rs = engine.Snapshot{}, nil
		}
	}()
	return s.cfg.Fold(b, in)
}

// stage — состояние межизделийной стадии (генеалогия, носители): кэш её
// свёртки, который пишет роль crossitem (AD-42).
func (s *Service) stage(ctx context.Context) (crossitem.Stage, error) {
	var st crossitem.Stage
	raw, ok, err := s.cfg.Projections.Get(ctx, StageState, stageStateKey)
	if err != nil || !ok {
		return st, err
	}
	return st, json.Unmarshal(raw, &st)
}

func (s *Service) row(ctx context.Context, itemID string) (RowRecord, bool, error) {
	var v RowRecord
	raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionRow, itemID)
	if err != nil || !ok {
		return v, false, err
	}
	return v, true, json.Unmarshal(raw, &v)
}

func notFound(itemID string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", "Изделие", "id", itemID)
	e.Detail = "Изделие «" + itemID + "» не найдено"
	return e
}

// refusal — доменный отказ как ошибка API.
func refusal(err error) error {
	if err == nil {
		return nil
	}
	if pe, ok := platform.AsError(err); ok {
		return pe
	}
	return err
}

// guard — гард операции над свёрткой изделия на текущий момент (AD-39).
func (s *Service) guard(ctx context.Context, itemID, action string, meta platform.CommandMeta, payload any) (at, error) {
	a, err := s.itemAt(ctx, itemID, platform.Moment{})
	var pe *platform.Error
	if errors.As(err, &pe) && pe.Code == errcodes.ApiNotFound && action == "item.item.register" {
		return at{}, nil
	}
	if err != nil {
		return a, err
	}
	b, _, err := s.cfg.Bundles.Bundle(ctx, itemID, a.Prefix)
	if err != nil {
		return a, err
	}
	cmd := kernel.Command{Action: action, CommandID: meta.CommandID, Object: "item:" + itemID, BasisSeq: meta.BasisSeq, Payload: payload}
	return a, refusal(engine.Guard(a.Snap, b, cmd))
}
