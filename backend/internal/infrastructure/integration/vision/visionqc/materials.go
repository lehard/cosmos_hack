package visionqc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	appvision "ant/internal/application/vision"
)

// maxIllustration — предел размера файла образца.
const maxIllustration = 16 << 20

// DirIllustrations — файлы открытого набора, распакованные на краю (FR-102):
// ‹Dir›/‹sample› или ‹Dir›/‹набор›/‹sample›; если файла образца нет — первый
// по имени файл каталога класса (‹Dir›[/‹набор›]/‹класс›/). Файлы набора в
// репозиторий не кладутся (PRD §11.12).
type DirIllustrations struct {
	Dir string
}

var _ appvision.IllustrationFiles = DirIllustrations{}

// Sample — байты образца класса.
func (d DirIllustrations) Sample(datasetID, class, sample string) ([]byte, error) {
	if d.Dir == "" {
		return nil, os.ErrNotExist
	}
	clean := func(p string) string { return filepath.Clean("/" + p)[1:] } // без выхода за каталог
	for _, base := range []string{d.Dir, filepath.Join(d.Dir, clean(datasetID))} {
		if b, err := readSmall(filepath.Join(base, clean(sample))); err == nil {
			return b, nil
		}
		ents, err := os.ReadDir(filepath.Join(base, clean(class)))
		if err != nil {
			continue
		}
		names := []string{}
		for _, e := range ents {
			if e.Type().IsRegular() {
				names = append(names, e.Name())
			}
		}
		slices.Sort(names)
		if len(names) > 0 {
			return readSmall(filepath.Join(base, clean(class), names[0]))
		}
	}
	return nil, os.ErrNotExist
}

func readSmall(p string) ([]byte, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > maxIllustration {
		return nil, fmt.Errorf("%s: не файл или больше %d байт", p, maxIllustration)
	}
	return os.ReadFile(p)
}

// HTTPMaterials — хранилище материалов ядра через операцию
// materials.material.upload (POST /api/v1/materials, AD-23). Адрес
// принимается, только если ядро его вернуло: иначе иллюстрация не прикладывается.
type HTTPMaterials struct {
	CoreURL string
	Client  *http.Client
}

var _ appvision.MaterialSink = HTTPMaterials{}

var digestRe = regexp.MustCompile(`^streebog256:[0-9a-f]{64}$`)

// PutIllustration — загрузить иллюстрацию: вид illustration, «иллюстрация, а
// не доказательство», происхождение — пометка источника, лицензии и автора.
func (h HTTPMaterials) PutIllustration(ctx context.Context, content []byte, _ string, note string) (string, error) {
	if h.CoreURL == "" {
		return "", errors.New("адрес ядра не задан")
	}
	c := h.Client
	if c == nil {
		c = &http.Client{Timeout: 30 * time.Second}
	}
	q := url.Values{"kind": {"illustration"}, "is_illustration": {"true"}, "provenance_note": {clipBytes(note, 512)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.CoreURL, "/")+"/api/v1/materials?"+q.Encode(), bytes.NewReader(content))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("хранилище материалов: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var info struct {
		MaterialAddress string `json:"material_address"`
	}
	if err := json.Unmarshal(b, &info); err != nil || !digestRe.MatchString(info.MaterialAddress) {
		return "", fmt.Errorf("хранилище материалов: адрес не получен")
	}
	return info.MaterialAddress, nil
}

// clipBytes — строка не длиннее n рун.
func clipBytes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
