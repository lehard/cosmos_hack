package notifications

import (
	"context"
	"encoding/json"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	dom "ant/internal/domain/notifications"
)

// Config — зависимости живой реализации (AD-36, режим live): проекции
// notifications.*, запись решений и доменные часы.
type Config struct {
	// Projections — чтение проекций модуля.
	Projections Projections
	// Decisions — запись решений человека (journal.Append, AD-44).
	Decisions DecisionWriter
	// Clock — доменное «сейчас» (AD-37): от него считается просрочка; nil — системное.
	Clock appjournal.DomainClock
	// Places — справочник мест для области задач (InScope); nil — задачи
	// сужаются только адресностью.
	Places Places
}

// Service — реализация live ведущих портов модуля notifications (AD-36):
// чтение — над проекциями сроков, задач и уведомлений; цена задержки — по
// проекции сроков на доменное «сейчас». Без зависимостей (NewService) —
// заглушка: операции отвечают 501.
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

// now — «сейчас» чтения: момент воспроизведения или доменное «сейчас»
// прогона (AD-22, AD-37).
func (s *Service) now(ctx context.Context, m platform.Moment) (time.Time, error) {
	if m.AsOf != nil {
		return m.AsOf.UTC(), nil
	}
	if s.cfg.Clock == nil {
		return time.Now().UTC(), nil
	}
	if m.RunID != "" {
		ctx = appjournal.WithRun(ctx, m.RunID)
	}
	t, err := s.cfg.Clock.Now(ctx)
	return t.UTC(), err
}

// rows — все строки проекции name, разобранные в T.
func rows[T any](ctx context.Context, p Projections, name string) ([]T, error) {
	all, err := p.All(ctx, name)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(all))
	for _, raw := range all {
		var v T
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) obligations(ctx context.Context) ([]dom.ObligationRecord, error) {
	return rows[dom.ObligationRecord](ctx, s.cfg.Projections, ProjectionObligation)
}

func (s *Service) tasks(ctx context.Context) ([]dom.TaskRecord, error) {
	return rows[dom.TaskRecord](ctx, s.cfg.Projections, ProjectionTask)
}

func (s *Service) notices(ctx context.Context) ([]dom.NoticeRecord, error) {
	return rows[dom.NoticeRecord](ctx, s.cfg.Projections, ProjectionNotice)
}
