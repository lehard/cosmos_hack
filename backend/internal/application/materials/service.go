package materials

import (
	"bytes"
	"context"
	"io"
	"strings"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Service — реализация live ведущих портов модуля materials (AD-36, AD-23):
// загрузка и чтение материалов по адресу содержимого H(байты) через ведомый
// порт MaterialStore (том в MVP). Без хранилища (выгрузка OpenAPI, тесты) —
// операции отвечают 501. Скан бумажной подписи (kind = scan) загружает тот,
// кто заверяет бумагу (FR-139, AD-43): адрес уходит в documents.paper.attest.
type Service struct {
	Unimplemented
	store MaterialStore
}

// NewService создаёт реализацию live; store — хранилище материалов (nil — 501).
func NewService(store ...MaterialStore) *Service {
	s := &Service{}
	if len(store) > 0 {
		s.store = store[0]
	}
	return s
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// Upload — сохранить материал (materials.material.upload): байты в хранилище,
// ответ — адрес содержимого и метаданные; повтор тех же байтов — тот же адрес.
func (s *Service) Upload(ctx context.Context, in MaterialUpload) (MaterialInfo, error) {
	if s.store == nil {
		return s.Unimplemented.Upload(ctx, in)
	}
	if len(in.Bytes) == 0 {
		return MaterialInfo{}, platform.Fail(errcodes.ApiValidationFailed, "field", "body", "reason", "пустой материал")
	}
	mt := in.MediaType
	if mt == "" {
		mt = "application/octet-stream"
	}
	addr, err := s.store.Put(ctx, bytes.NewReader(in.Bytes), Meta{ContentType: mt, Size: int64(len(in.Bytes)), Kind: in.Kind})
	if err != nil {
		return MaterialInfo{}, err
	}
	return MaterialInfo{MaterialAddress: addr, MediaType: mt, SizeBytes: int64(len(in.Bytes)), Kind: in.Kind, CapturedAt: in.CapturedAt,
		ItemID: in.ItemID, IsIllustration: in.IsIllustration, ProvenanceNote: in.ProvenanceNote}, nil
}

// Material — метаданные материала по адресу (materials.material.read).
func (s *Service) Material(ctx context.Context, address string) (MaterialInfo, error) {
	if s.store == nil {
		return s.Unimplemented.Material(ctx, address)
	}
	r, meta, err := s.open(ctx, address)
	if err != nil {
		return MaterialInfo{}, err
	}
	_ = r.Close()
	return MaterialInfo{MaterialAddress: address, MediaType: meta.ContentType, SizeBytes: meta.Size, Kind: kindOf(meta.Kind)}, nil
}

// Content — байты материала по адресу (materials.material.content).
func (s *Service) Content(ctx context.Context, address string) (MaterialContent, error) {
	if s.store == nil {
		return s.Unimplemented.Content(ctx, address)
	}
	r, meta, err := s.open(ctx, address)
	if err != nil {
		return MaterialContent{}, err
	}
	defer func() { _ = r.Close() }()
	b, err := io.ReadAll(r)
	if err != nil {
		return MaterialContent{}, err
	}
	return MaterialContent{MediaType: meta.ContentType, Bytes: b}, nil
}

// open — материал по адресу; нет — 404 api.not_found.
func (s *Service) open(ctx context.Context, address string) (io.ReadCloser, Meta, error) {
	if ok, err := s.store.Has(ctx, address); err != nil || !ok {
		if err == nil {
			err = platform.Fail(errcodes.ApiNotFound, "object", "материал", "id", address)
		}
		return nil, Meta{}, err
	}
	return s.store.Get(ctx, address)
}

// kindOf — вид материала для ответа: вид хранилища вне перечня API — other.
func kindOf(k string) string {
	switch strings.TrimSpace(k) {
	case "photo", "video", "illustration", "protocol", "log_excerpt", "scan", "quarantine_payload":
		return k
	case "quarantine":
		return "quarantine_payload"
	}
	return "other"
}
