package profiles

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	app "ant/internal/application/signing"
	dom "ant/internal/domain/signing"
)

// PublicKeys — открытые ключи по key_ref (реестр ключей, AD-11). Ошибка
// ErrKeyUnavailable — ключ зарегистрирован, но его открытая часть сейчас
// недоступна: итог «не проверяемо», никогда не «валидно» (AD-32).
type PublicKeys interface {
	PublicKey(ctx context.Context, keyRef string) (profile string, pub []byte, err error)
}

// ErrKeyUnavailable — ключ недоступен (signing.key_unavailable).
var ErrKeyUnavailable = errors.New("signing.key_unavailable")

// Verifier — адаптер порта application/signing.Verifier (AD-10): проверяет
// каждую подпись конверта открытым ключом из реестра. Решение по пакету
// (профиль на момент подписи, отзыв, компрометация) — domain/signing.Judge
// над результатом Check.
type Verifier struct {
	Keys PublicKeys
}

var _ app.Verifier = Verifier{}

// Check — криптографическая проверка каждой подписи конверта: ok / bad /
// unavailable (ключа нет в реестре — тоже unavailable: неизвестен ли ключ
// вовсе, решает Judge по реестру).
func (v Verifier) Check(ctx context.Context, raw []byte) (dom.Envelope, []byte, []dom.CryptoCheck, error) {
	env, payload, err := dom.ParseEnvelope(raw)
	if err != nil {
		return env, nil, nil, err
	}
	pae := dom.PAE(env.PayloadType, payload)
	out := make([]dom.CryptoCheck, 0, len(env.Signatures))
	for _, s := range env.Signatures {
		c := dom.CryptoCheck{KeyRef: s.KeyID, Result: dom.CryptoUnavailable}
		profile, pub, err := v.Keys.PublicKey(ctx, s.KeyID)
		if err == nil && len(pub) > 0 {
			sig, _ := base64.StdEncoding.DecodeString(s.Sig)
			c.Result = dom.CryptoBad
			if Verify(profile, pub, env.PayloadType, pae, sig) {
				c.Result = dom.CryptoOK
			}
		}
		out = append(out, c)
	}
	return env, payload, out, nil
}

// Verify — порт приёма (FR-26): payload и подписанты с верной подписью.
// Ни одной верной подписи — ошибка (недоступный ключ — ErrKeyUnavailable).
func (v Verifier) Verify(ctx context.Context, raw []byte) (app.Verified, error) {
	env, payload, checks, err := v.Check(ctx, raw)
	if err != nil {
		return app.Verified{}, err
	}
	out := app.Verified{PayloadType: env.PayloadType, Payload: payload}
	unavailable := false
	for _, c := range checks {
		switch c.Result {
		case dom.CryptoOK:
			profile, _, _ := v.Keys.PublicKey(ctx, c.KeyRef)
			out.Signers = append(out.Signers, app.SignerRef{KeyRef: c.KeyRef, Profile: profile})
		case dom.CryptoUnavailable:
			unavailable = true
		}
	}
	switch {
	case len(out.Signers) > 0:
		return out, nil
	case unavailable:
		return out, fmt.Errorf("%w: открытого ключа нет — не проверяемо", ErrKeyUnavailable)
	case len(checks) == 0:
		return out, errors.New("пакет без подписей")
	}
	return out, errors.New("подпись не сходится с содержимым пакета")
}
