package materials

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля materials (AD-36).
type Queries interface {
	// Material — метаданные материала по адресу (materials.material.read).
	Material(ctx context.Context, address string) (MaterialInfo, error)
	// Content — байты материала по адресу (materials.material.content).
	Content(ctx context.Context, address string) (MaterialContent, error)
}

// Commands — ведущий порт команд модуля materials.
type Commands interface {
	// Upload — сохранить материал по адресу содержимого (materials.material.upload, AD-23).
	Upload(ctx context.Context, in MaterialUpload) (MaterialInfo, error)
}

// Unimplemented — заглушка портов materials: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func (Unimplemented) Material(context.Context, string) (MaterialInfo, error) {
	return MaterialInfo{}, platform.NotImplemented("materials.material.read")
}

func (Unimplemented) Content(context.Context, string) (MaterialContent, error) {
	return MaterialContent{}, platform.NotImplemented("materials.material.content")
}

func (Unimplemented) Upload(context.Context, MaterialUpload) (MaterialInfo, error) {
	return MaterialInfo{}, platform.NotImplemented("materials.material.upload")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
