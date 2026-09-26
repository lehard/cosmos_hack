package ingest

import (
	"encoding/json"
	"fmt"
)

// ContentFingerprint — отпечаток содержимого события для идемпотентности (AD-7,
// FR-31): H(JCS(событие без блока integrity)). Сравнивается «содержимое», а не
// конверт и не подписи: тот же payload, переподписанный тем же ключом или
// ключом-преемником по акту ротации (другой integrity.signers), — дубль, а не
// конфликт. canonical — событие в каноническом виде (Canonicalize).
func ContentFingerprint(canonical []byte) (string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &obj); err != nil {
		return "", fmt.Errorf("%w: событие — не объект JSON: %v", ErrCanonical, err)
	}
	delete(obj, "integrity")
	b, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCanonical, err)
	}
	c, err := Canonicalize(b)
	if err != nil {
		return "", err
	}
	return Digest(c), nil
}
