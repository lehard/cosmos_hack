package mes

import (
	"context"
	"encoding/json"
	"strconv"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	dom "ant/internal/domain/mes"
)

// Config — зависимости живой реализации ведущих портов (AD-36, режим live).
type Config struct {
	// Projections — проекции mes.* движка (роль projector).
	Projections engineapp.ProjectionStore
}

// Service — реализация live ведущих портов модуля mes (AD-36): чтение над
// проекциями mes.block и mes.job. Без зависимостей (NewService) — заглушка 501.
type Service struct {
	Unimplemented
	cfg Config
}

// NewService создаёт заглушку live (все операции — 501); живую реализацию
// собирает NewLive.
func NewService() *Service { return &Service{} }

// NewLive создаёт живую реализацию.
func NewLive(cfg Config) *Service { return &Service{cfg: cfg} }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

func (s *Service) keys(ctx context.Context, index string) ([]string, error) {
	var ks []string
	raw, ok, err := s.cfg.Projections.Get(ctx, index, "all")
	if err != nil || !ok {
		return nil, err
	}
	return ks, json.Unmarshal(raw, &ks)
}

// Jobs — задания MES (mes.order.list), последние — первыми; курсор — смещение.
func (s *Service) Jobs(ctx context.Context, m platform.Moment, p platform.Page) (MesJobList, error) {
	if s.cfg.Projections == nil {
		return s.Unimplemented.Jobs(ctx, m, p)
	}
	out := MesJobList{Items: []MesJob{}}
	ks, err := s.keys(ctx, ProjectionJobIndex)
	if err != nil {
		return out, err
	}
	skip, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	for i := len(ks) - 1 - skip; i >= 0 && len(out.Items) < limit; i-- {
		raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionJob, ks[i])
		if err != nil {
			return out, err
		}
		if !ok {
			continue
		}
		var j dom.JobView
		if err := json.Unmarshal(raw, &j); err != nil {
			return out, err
		}
		if m.AsOf != nil && j.ReceivedAt.After(*m.AsOf) {
			continue
		}
		x := MesJob{JobID: string(j.Data.JobID), ExternalNumber: j.Data.ExternalNumber, OperationCode: j.Data.OperationCode, ReceivedAt: j.ReceivedAt}
		if j.Data.OrderID != nil {
			x.OrderID = string(*j.Data.OrderID)
		}
		if j.Data.StationID != nil {
			x.StationID = string(*j.Data.StationID)
		}
		if j.Data.PlannedStart != nil {
			t := j.Data.PlannedStart.Time()
			x.PlannedStart = &t
		}
		out.Items = append(out.Items, x)
	}
	if n := skip + len(out.Items); n < len(ks) && len(out.Items) == limit {
		out.NextCursor = strconv.Itoa(n)
	}
	return out, nil
}

// Blocks — блоки и снятия в MES (mes.block.list), последние — первыми.
func (s *Service) Blocks(ctx context.Context, m platform.Moment) (MesBlockList, error) {
	if s.cfg.Projections == nil {
		return s.Unimplemented.Blocks(ctx, m)
	}
	out := MesBlockList{Items: []MesBlock{}}
	ks, err := s.keys(ctx, ProjectionBlockIndex)
	if err != nil {
		return out, err
	}
	for i := len(ks) - 1; i >= 0; i-- {
		raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionBlock, ks[i])
		if err != nil {
			return out, err
		}
		if !ok {
			continue
		}
		var b dom.Block
		if err := json.Unmarshal(raw, &b); err != nil {
			return out, err
		}
		if m.AsOf != nil && b.RequestedAt.After(*m.AsOf) {
			continue
		}
		out.Items = append(out.Items, MesBlock{BusinessKey: b.Key, ItemID: b.ItemID, LotID: b.LotID, Hold: b.Hold, RequestedAt: b.RequestedAt,
			Outcome: b.Outcome, ErrorCode: b.ErrorCode, RespondedAt: b.RespondedAt})
	}
	return out, nil
}
