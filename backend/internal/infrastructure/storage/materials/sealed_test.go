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
	"ant/internal/infrastructure/security/atrest"
)

// AD-23: в томе материалов — только шифротекст; чтение с KEK отдаёт
// открытые байты и проверяет адрес; без KEK — «зашифровано»; порча
// шифротекста обнаруживается.
func TestVolumeSealed(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	kek, err := atrest.New(bytes.Repeat([]byte{3}, atrest.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewVolume(root, WithSealer(kek))
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("карантин: исходные байты сообщения edge-ws2")
	addr, err := v.Put(ctx, bytes.NewReader(data), app.Meta{Kind: "quarantine"})
	if err != nil || addr != dm.Address(data) {
		t.Fatalf("Put: %v", err)
	}
	h, _ := dm.ParseAddress(addr)
	file := filepath.Join(root, h[:2], h)
	onDisk, _ := os.ReadFile(file)
	if bytes.Contains(onDisk, []byte("edge-ws2")) || !bytes.HasPrefix(onDisk, []byte(sealedMagic)) {
		t.Fatal("в томе открытые байты")
	}
	rc, _, err := v.Get(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	if !bytes.Equal(got, data) {
		t.Fatalf("Get: %q", got)
	}
	blind, _ := NewVolume(root)
	if _, _, err := blind.Get(ctx, addr); !errors.Is(err, ErrSealed) {
		t.Fatalf("без KEK: %v", err)
	}
	onDisk[len(onDisk)-1] ^= 1
	_ = os.WriteFile(file, onDisk, 0o640)
	if _, _, err := v.Get(ctx, addr); !errors.Is(err, ErrTampered) {
		t.Fatalf("порча: %v", err)
	}
}
