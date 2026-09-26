package materials

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	app "ant/internal/application/materials"
	dm "ant/internal/domain/materials"
)

// Материал хранится по адресу содержимого: повтор — тот же адрес, подмена
// байтов в томе обнаруживается при чтении (AD-23, контрактный тест порта).
func TestVolumeByAddress(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	v, err := NewVolume(root)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("скан сопроводительной карты ENT01:FL-0007")
	addr, err := v.Put(ctx, bytes.NewReader(data), app.Meta{ContentType: "image/png", Kind: "scan"})
	if err != nil || addr != dm.Address(data) {
		t.Fatalf("Put: %s %v", addr, err)
	}
	again, err := v.Put(ctx, bytes.NewReader(data), app.Meta{Kind: "scan"})
	if err != nil || again != addr {
		t.Fatalf("повтор: %s %v", again, err)
	}
	if ok, err := v.Has(ctx, addr); !ok || err != nil {
		t.Fatalf("Has: %v %v", ok, err)
	}
	rc, meta, err := v.Get(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(got, data) || meta.ContentType != "image/png" || meta.Size != int64(len(data)) {
		t.Fatalf("Get: %q %+v", got, meta)
	}
	missing := dm.Address([]byte("нет такого"))
	if ok, _ := v.Has(ctx, missing); ok {
		t.Fatal("Has для отсутствующего")
	}
	if _, _, err := v.Get(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get отсутствующего: %v", err)
	}
	h, _ := dm.ParseAddress(addr)
	if err := os.WriteFile(filepath.Join(root, h[:2], h), []byte("подмена"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := v.Get(ctx, addr); !errors.Is(err, ErrTampered) {
		t.Fatalf("подмена не обнаружена: %v", err)
	}
	if _, _, err := v.Get(ctx, "sha256:00"); err == nil {
		t.Fatal("неверный адрес принят")
	}
}
