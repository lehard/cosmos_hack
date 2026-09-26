package documents

import (
	"encoding/hex"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"

	"ant/internal/contracts/constants"
)

// Отпечаток документа (AD-12, кейс §6.3): одни и те же правила у сервера,
// агента токена и demo-signer — все собираются из одного коммита и вызывают
// эти функции; тест «отпечаток сервера = отпечаток агента» сверяет их с
// эталонами contracts/crypto/test-vectors (раздел doc_digest) и
// contracts/crypto/test-vectors/documents.
//
//	content        = JCS(данные документа без rendering_hash)
//	rendering_hash = H(render(шаблон@версия, content))
//	doc_digest     = H(JCS({content, rendering_hash, template_ref, doc_format_version}))
//
// H — хеш формата цепочки v1 (Стрибог-256, AD-44) с префиксом `streebog256:`.

// Canonical — канонический JSON значения (RFC 8785, JCS): ключи отсортированы,
// числа и строки в нормальной форме. Числа — только целые (AD-4, AD-10).
func Canonical(v any) ([]byte, error) {
	var raw []byte
	switch x := v.(type) {
	case json.RawMessage:
		raw = x
	case []byte:
		raw = x
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	val := jsontext.Value(append([]byte(nil), raw...))
	if err := val.Canonicalize(); err != nil {
		return nil, fmt.Errorf("documents: канонический JSON: %w", err)
	}
	return []byte(val), nil
}

// Hash — H(байты) в виде `streebog256:‹hex›` (AD-44).
func Hash(b []byte) string {
	h := gost34112012256.New()
	h.Write(b)
	return constants.DigestPrefix + hex.EncodeToString(h.Sum(nil))
}

// RenderingHash — H(каноническая отрисовка HTML) (AD-12).
func RenderingHash(html string) string { return Hash([]byte(html)) }

// digestInput — объект, от которого берётся отпечаток (AD-12).
type digestInput struct {
	Content          json.RawMessage `json:"content"`
	RenderingHash    string          `json:"rendering_hash"`
	TemplateRef      string          `json:"template_ref"`
	DocFormatVersion int             `json:"doc_format_version"`
}

// DigestInput — канонические байты, от которых берётся отпечаток (для
// эталонов и агента токена).
func DigestInput(content json.RawMessage, renderingHash, templateRef string, formatVersion int) ([]byte, error) {
	c, err := Canonical(content)
	if err != nil {
		return nil, err
	}
	return Canonical(digestInput{Content: c, RenderingHash: renderingHash, TemplateRef: templateRef, DocFormatVersion: formatVersion})
}

// DocDigest — отпечаток документа doc_digest (AD-12). Неизвестная версия
// формата — ошибка (агент токена отвергает её так же, AD-12).
func DocDigest(content json.RawMessage, renderingHash, templateRef string, formatVersion int) (string, error) {
	if formatVersion != DocFormatVersion {
		return "", fmt.Errorf("documents: неизвестная версия формата документа %d (signing.unknown_doc_format)", formatVersion)
	}
	in, err := DigestInput(content, renderingHash, templateRef, formatVersion)
	if err != nil {
		return "", err
	}
	return Hash(in), nil
}

// QR — содержимое QR печатной рамки `ant:doc:‹id›:‹отпечаток›` (соглашение
// «QR», AD-12, AD-43). QR в отрисовку не входит — иначе цикл «QR ↔ отпечаток».
func QR(documentID, digest string) string {
	r := strings.NewReplacer("{doc_id}", documentID, "{doc_digest}", digest)
	return r.Replace(constants.QrDocumentTemplate)
}

// ParseQR — документ и отпечаток из QR скана; ok=false — не QR документа.
func ParseQR(s string) (documentID, digest string, ok bool) {
	rest, found := strings.CutPrefix(strings.TrimSpace(s), "ant:doc:")
	if !found {
		return "", "", false
	}
	i := strings.LastIndex(rest, ":"+constants.DigestPrefix)
	if i <= 0 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}

// Built — собранная версия документа: канонический content, отрисовка, хеш
// отрисовки и отпечаток (AD-12).
type Built struct {
	Content       json.RawMessage
	HTML          string
	RenderingHash string
	Digest        string
}

// Build — канонический content, отрисовка по шаблону и отпечаток. Одна
// функция для свёртки, api, агента токена и верификатора.
func Build(t Template, content any) (Built, error) {
	c, err := Canonical(content)
	if err != nil {
		return Built{}, err
	}
	html, err := Render(t, c)
	if err != nil {
		return Built{}, err
	}
	rh := RenderingHash(html)
	d, err := DocDigest(c, rh, t.Ref(), DocFormatVersion)
	if err != nil {
		return Built{}, err
	}
	return Built{Content: c, HTML: html, RenderingHash: rh, Digest: d}, nil
}
