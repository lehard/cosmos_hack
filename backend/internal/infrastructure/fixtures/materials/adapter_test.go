package materials

import (
	"context"
	"strings"
	"testing"

	app "ant/internal/application/materials"
)

// TestUploadInMemory — скан на заготовках: адрес содержимого, чтение по адресу.
func TestUploadInMemory(t *testing.T) {
	a, ctx := New(), context.Background()
	info, err := a.Upload(ctx, app.MaterialUpload{Kind: "scan", MediaType: "image/jpeg", Bytes: []byte("jpeg")})
	if err != nil || !strings.HasPrefix(info.MaterialAddress, "streebog256:") || len(info.MaterialAddress) != len("streebog256:")+64 {
		t.Fatalf("загрузка: %+v %v", info, err)
	}
	again, _ := a.Upload(ctx, app.MaterialUpload{Kind: "scan", MediaType: "image/jpeg", Bytes: []byte("jpeg")})
	if again.MaterialAddress != info.MaterialAddress {
		t.Fatal("повтор тех же байтов — другой адрес")
	}
	c, err := a.Content(ctx, info.MaterialAddress)
	if err != nil || string(c.Bytes) != "jpeg" || c.MediaType != "image/jpeg" {
		t.Fatalf("содержимое: %+v %v", c, err)
	}
}
