package ingest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"

	"ant/internal/contracts/errcodes"
)

// frame — сообщение на входе: конверт DSSE (payloadType, payload, подписи) или,
// в профиле demo, само событие без конверта.
type frame struct {
	DSSE        bool
	PayloadType string
	Payload     []byte
	KeyRefs     []string
	Raw         []byte
}

// parseFrame распознаёт DSSE по полю payloadType (AD-10).
func parseFrame(raw []byte) (frame, error) {
	var probe struct {
		PayloadType *string `json:"payloadType"`
		Payload     string  `json:"payload"`
		Signatures  []struct {
			KeyID string `json:"keyid"`
		} `json:"signatures"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return frame{Raw: raw, Payload: raw}, err
	}
	if probe.PayloadType == nil {
		return frame{Raw: raw, Payload: raw}, nil
	}
	p, err := base64.StdEncoding.DecodeString(probe.Payload)
	if err != nil {
		return frame{DSSE: true, Raw: raw}, err
	}
	f := frame{DSSE: true, PayloadType: *probe.PayloadType, Payload: p, Raw: raw}
	for _, s := range probe.Signatures {
		f.KeyRefs = append(f.KeyRefs, s.KeyID)
	}
	return f, nil
}

// authResult — итог проверки подлинности (FR-26).
type authResult struct {
	Verified   bool
	Provenance string
	Note       string
	Fail       *authFail
}

// authFail — отказ: код приёма и значение failure для security.signature.invalid.
type authFail struct {
	Code    errcodes.Code
	Failure string
	KeyRef  string
	Detail  string
}

// provenanceFor — класс происхождения неподтверждённого подписью факта (AD-2):
// прогон сценария — scenario; ручной ввод — personal; шлюз внешней системы и
// импорт — server_attested; прочее (станок, датчик, камера через edge-агент) — device.
func provenanceFor(h header) string {
	switch {
	case h.RunID != "":
		return "scenario"
	case h.SourceKind == "manual_entry":
		return "personal"
	case h.SourceKind == "external_system" || h.SourceKind == "import":
		return "server_attested"
	}
	return "device"
}

// authenticate — FR-26: приём только от зарегистрированных источников;
// сообщение подписано ключом источника; неподписанное, неизвестный или
// отозванный ключ — отказ с кодом. В профиле demo (SignatureDemoUnverified)
// неподписанное принимается с пометкой «подпись не проверялась».
func (s *Service) authenticate(ctx context.Context, f frame, h header) authResult {
	signed := f.DSSE && len(f.KeyRefs) > 0
	canCheck := s.deps.Verifier != nil && s.deps.Keys != nil
	if s.cfg.Signature == SignatureDemoUnverified && (!signed || !canCheck) {
		return authResult{Provenance: provenanceFor(h), Note: SignatureNotVerified}
	}
	if !signed {
		return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "bad_signature", Detail: "сообщение не подписано"}}
	}
	if !canCheck {
		return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "key_unavailable", Detail: "проверка подписи не подключена"}}
	}
	if err := s.deps.Keys.Source(ctx, h.SourceID); err != nil {
		if errors.Is(err, ErrUnknownSource) {
			return authResult{Fail: &authFail{Code: errcodes.IngestUnknownSource, Failure: "unknown_key", KeyRef: first(f.KeyRefs),
				Detail: "источник «" + h.SourceID + "» не зарегистрирован"}}
		}
		return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "key_unavailable", Detail: err.Error()}}
	}
	prov := ""
	for _, ref := range f.KeyRefs {
		k, err := s.deps.Keys.Key(ctx, ref)
		switch {
		case errors.Is(err, ErrUnknownKey):
			return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "unknown_key", KeyRef: ref, Detail: "ключ не зарегистрирован"}}
		case err != nil:
			return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "key_unavailable", KeyRef: ref, Detail: err.Error()}}
		case k.Revoked:
			// FR-26, FR-70: отозванный ключ — отказ и шина безопасности.
			return authResult{Fail: &authFail{Code: errcodes.IngestRevokedKey, Failure: "revoked_key", KeyRef: ref, Detail: "ключ отозван"}}
		case k.SourceID != h.SourceID:
			return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "foreign_key", KeyRef: ref,
				Detail: "ключ выдан источнику «" + k.SourceID + "», а не «" + h.SourceID + "»"}}
		}
		prov = k.Provenance
	}
	v, err := s.deps.Verifier.Verify(ctx, f.Raw)
	if err != nil {
		return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "bad_signature", KeyRef: first(f.KeyRefs), Detail: err.Error()}}
	}
	if !bytes.Equal(v.Payload, f.Payload) {
		return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "payload_tampered", KeyRef: first(f.KeyRefs), Detail: "подписано другое содержимое"}}
	}
	var ok []string
	for _, sg := range v.Signers {
		ok = append(ok, sg.KeyRef)
	}
	// AD-10: обязательные подписанты пакета — integrity.signers; лишние не засчитываются.
	for _, need := range h.Integrity.Signers {
		if !slices.Contains(ok, need) {
			return authResult{Fail: &authFail{Code: errcodes.IngestSignatureInvalid, Failure: "bad_signature", KeyRef: need, Detail: "нет действительной подписи обязательного подписанта"}}
		}
	}
	if prov == "" {
		prov = provenanceFor(h)
	}
	if h.RunID != "" {
		prov = "scenario"
	}
	return authResult{Verified: true, Provenance: prov}
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
