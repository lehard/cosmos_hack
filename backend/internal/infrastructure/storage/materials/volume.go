package materials

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.stargrave.org/gogost/v7/gost34112012256"

	app "ant/internal/application/materials"
	dm "ant/internal/domain/materials"
)

// Volume — адаптер volume порта MaterialStore (AD-23, AD-35, ключ
// material_store): материалы лежат в томе по адресу содержимого
// `‹root›/‹hex[0:2]›/‹hex›`, метаданные — рядом в `‹hex›.meta.json`.
// Повторная запись тех же байтов даёт тот же адрес и ничего не меняет;
// при чтении адрес проверяется заново (подмена файла в томе обнаруживается).
//
// TODO(29): шифрование конвертной схемой (DEK на объект, KEK — том), AD-23.
// В демо-треке байты лежат открыто.
type Volume struct {
	root string
}

// NewVolume создаёт хранилище в каталоге root (создаётся при необходимости).
func NewVolume(root string) (*Volume, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("хранилище материалов: %w", err)
	}
	return &Volume{root: root}, nil
}

var _ app.MaterialStore = (*Volume)(nil)

// ErrNotFound — материала с таким адресом нет (верификатор: «не проверяемо»).
var ErrNotFound = errors.New("materials: материал не найден")

// ErrTampered — байты в томе не совпадают с адресом.
var ErrTampered = errors.New("materials: содержимое не совпадает с адресом")

func (v *Volume) paths(hexAddr string) (dir, data, meta string) {
	dir = filepath.Join(v.root, hexAddr[:2])
	return dir, filepath.Join(dir, hexAddr), filepath.Join(dir, hexAddr+".meta.json")
}

// Put сохраняет байты r и возвращает адрес H(байты). Запись — во временный
// файл с потоковым хешированием, затем атомарное переименование.
func (v *Volume) Put(ctx context.Context, r io.Reader, meta app.Meta) (string, error) {
	tmp, err := os.CreateTemp(v.root, ".put-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	h := gost34112012256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), ctxReader{ctx, r})
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	sum := h.Sum(nil)
	addr := dm.FormatAddress(sum)
	dir, data, metaPath := v.paths(hex.EncodeToString(sum))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	if _, err := os.Stat(data); err == nil {
		return addr, nil // тот же адрес — те же байты
	}
	meta.Size = n
	mb, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(metaPath, mb, 0o640); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), data); err != nil {
		return "", err
	}
	return addr, nil
}

// Get открывает материал; байты проверяются на совпадение с адресом до выдачи.
func (v *Volume) Get(ctx context.Context, address string) (io.ReadCloser, app.Meta, error) {
	h, err := dm.ParseAddress(address)
	if err != nil {
		return nil, app.Meta{}, err
	}
	_, data, metaPath := v.paths(h)
	raw, err := os.ReadFile(data)
	if errors.Is(err, os.ErrNotExist) {
		return nil, app.Meta{}, fmt.Errorf("%w: %s", ErrNotFound, address)
	}
	if err != nil {
		return nil, app.Meta{}, err
	}
	if dm.Address(raw) != address {
		return nil, app.Meta{}, fmt.Errorf("%w: %s", ErrTampered, address)
	}
	var meta app.Meta
	if mb, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(mb, &meta)
	}
	meta.Size = int64(len(raw))
	return io.NopCloser(bytes.NewReader(raw)), meta, ctx.Err()
}

// Has — есть ли материал.
func (v *Volume) Has(_ context.Context, address string) (bool, error) {
	h, err := dm.ParseAddress(address)
	if err != nil {
		return false, err
	}
	_, data, _ := v.paths(h)
	_, err = os.Stat(data)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
