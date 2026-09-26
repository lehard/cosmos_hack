package ingest

import (
	"bytes"
	"encoding/hex"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"strconv"

	"go.stargrave.org/gogost/v7/gost34112012256"

	"ant/internal/contracts/constants"
)

// MaxSafeInteger — наибольшее целое подписываемого JSON (AD-10, «Соглашения/Числа»).
const MaxSafeInteger = constants.MaxSafeInteger

// ErrCanonical — байты нельзя привести к каноническому виду RFC 8785 без
// потери смысла: повторяющиеся ключи, нецелые числа или целые вне ±(2^53−1),
// неверный UTF-8, не JSON (AD-10; код ingest.canonical_form_violation).
var ErrCanonical = errors.New("канонический вид JSON недостижим")

// Canonicalize возвращает JCS (RFC 8785) значения raw. Числа — только целые в
// пределах ±(2^53−1): в контрактах нет type: number (AD-20), а дробное или
// слишком большое число JCS округлил бы через double — отпечаток разошёлся бы
// с источником. Повторяющиеся ключи — отказ (AD-10).
func Canonicalize(raw []byte) ([]byte, error) {
	if err := checkIntegers(raw); err != nil {
		return nil, err
	}
	v := jsontext.Value(bytes.Clone(raw))
	if err := v.Canonicalize(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCanonical, err)
	}
	return []byte(v), nil
}

// checkIntegers проходит все токены и отвергает числа, которые не являются
// целыми в пределах ±(2^53−1).
func checkIntegers(raw []byte) error {
	dec := jsontext.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.ReadToken()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("%w: %v", ErrCanonical, err)
		}
		if tok.Kind() != '0' {
			continue
		}
		s := tok.String()
		n, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || n > MaxSafeInteger || n < -MaxSafeInteger {
			return fmt.Errorf("%w: число %s — не целое в пределах ±(2^53−1)", ErrCanonical, s)
		}
	}
}

// Digest — отпечаток по хешу формата цепочки: `streebog256:‹64 hex›` (AD-44):
// Стрибог-256 (ГОСТ Р 34.11-2012). Им же адресуется содержимое карантина.
func Digest(b []byte) string {
	h := gost34112012256.New()
	h.Write(b)
	return constants.DigestPrefix + hex.EncodeToString(h.Sum(nil))
}
