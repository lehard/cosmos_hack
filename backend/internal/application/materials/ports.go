package materials

import (
	"context"
	"io"
)

// MaterialStore — ведомый порт хранилища материалов (AD-23, AD-35, ключ
// material_store): сканы, кадры, вложения, содержимое карантина по адресу
// содержимого H(байты) (`streebog256:…`), зашифрованные конвертной схемой.
// Том в MVP, S3-совместимое хранилище в контуре — замена; тест — по адресу.
type MaterialStore interface {
	// Put сохраняет байты и возвращает адрес содержимого; повтор — тот же адрес.
	Put(ctx context.Context, r io.Reader, meta Meta) (Address string, err error)
	// Get открывает материал по адресу; адрес проверяется после чтения.
	Get(ctx context.Context, address string) (io.ReadCloser, Meta, error)
	// Has — есть ли материал (верификатор: «не проверяемо», если нет).
	Has(ctx context.Context, address string) (bool, error)
}

// Meta — метаданные материала (в журнал попадают только адрес и метаданные).
type Meta struct {
	ContentType string
	Size        int64
	// Kind — вид: scan | frame | attachment | quarantine.
	Kind string
}
