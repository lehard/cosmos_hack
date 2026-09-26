package ingest

import (
	"context"
	"errors"
	"io"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

func view(r QuarantineRecord) QuarantineView {
	title := string(r.Code)
	if i, ok := errcodes.Lookup(r.Code); ok {
		title = i.Title
	}
	return QuarantineView{ID: r.ID, JournalSeq: r.JournalSeq, SourceID: r.SourceID, EventID: r.EventID, EventType: r.EventType,
		SourceSeq: r.SourceSeq, Code: r.Code, Title: title, Field: r.Field, Detail: r.Detail, Fingerprint: r.Fingerprint,
		MaterialAddress: r.MaterialAddress, Status: r.Status, ReceivedAt: r.ReceivedAt, ResolvedBy: r.ResolvedBy}
}

// QuarantineRecords — записи карантина для администратора (FR-30).
func (s *Service) QuarantineRecords(ctx context.Context, f QuarantineQuery) ([]QuarantineView, error) {
	if err := s.ready(); err != nil {
		return nil, platform.NotImplemented("ingest.quarantine.list")
	}
	rs, err := s.deps.Quarantine.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]QuarantineView, 0, len(rs))
	for _, r := range rs {
		out = append(out, view(r))
	}
	return out, nil
}

// QuarantineItem — одна запись карантина.
func (s *Service) QuarantineItem(ctx context.Context, id string) (QuarantineView, error) {
	if err := s.ready(); err != nil {
		return QuarantineView{}, platform.NotImplemented("ingest.quarantine.read")
	}
	r, err := s.deps.Quarantine.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return QuarantineView{}, platform.Fail(errcodes.ApiNotFound, "object", "Запись карантина", "id", id)
	}
	if err != nil {
		return QuarantineView{}, err
	}
	return view(r), nil
}

// content — исходные байты записи карантина.
func (s *Service) content(ctx context.Context, r QuarantineRecord) ([]byte, error) {
	if r.Raw != nil {
		return r.Raw, nil
	}
	if s.deps.Materials == nil {
		return nil, errors.New("содержимого карантина нет: хранилище материалов не подключено")
	}
	rc, _, err := s.deps.Materials.Get(ctx, r.MaterialAddress)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}
