package materials

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
)

// memStore — хранилище материалов в памяти для теста.
type memStore struct {
	data map[string][]byte
	meta map[string]Meta
}

func (m *memStore) Put(_ context.Context, r io.Reader, meta Meta) (string, error) {
	b, _ := io.ReadAll(r)
	h := sha256.Sum256(b)
	addr := "streebog256:" + hex.EncodeToString(h[:])
	m.data[addr], m.meta[addr] = b, meta
	return addr, nil
}

func (m *memStore) Get(_ context.Context, address string) (io.ReadCloser, Meta, error) {
	return io.NopCloser(bytes.NewReader(m.data[address])), m.meta[address], nil
}

func (m *memStore) Has(_ context.Context, address string) (bool, error) {
	_, ok := m.data[address]
	return ok, nil
}

// TestUploadScan — скан бумажной подписи сохраняется и читается по адресу;
// без хранилища — 501, пустой материал — 422.
func TestUploadScan(t *testing.T) {
	ctx := context.Background()
	if _, err := NewService().Upload(ctx, MaterialUpload{Kind: "scan", Bytes: []byte("x")}); err == nil {
		t.Fatal("без хранилища загрузка должна отвечать 501")
	}
	s := NewService(&memStore{data: map[string][]byte{}, meta: map[string]Meta{}})
	if _, err := s.Upload(ctx, MaterialUpload{Kind: "scan"}); err == nil {
		t.Fatal("пустой материал принят")
	}
	info, err := s.Upload(ctx, MaterialUpload{Kind: "scan", MediaType: "image/png", Bytes: []byte("png")})
	if err != nil || info.MaterialAddress == "" || info.Kind != "scan" || info.SizeBytes != 3 {
		t.Fatalf("загрузка: %+v %v", info, err)
	}
	got, err := s.Material(ctx, info.MaterialAddress)
	if err != nil || got.MediaType != "image/png" || got.Kind != "scan" {
		t.Fatalf("метаданные: %+v %v", got, err)
	}
	c, err := s.Content(ctx, info.MaterialAddress)
	if err != nil || string(c.Bytes) != "png" {
		t.Fatalf("содержимое: %+v %v", c, err)
	}
	if _, err := s.Material(ctx, "streebog256:00"); err == nil {
		t.Fatal("нет материала — должно быть 404")
	}
}
