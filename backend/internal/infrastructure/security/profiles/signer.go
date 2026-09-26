package profiles

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	app "ant/internal/application/signing"
	dom "ant/internal/domain/signing"
)

// Signer — адаптер порта application/signing.Signer (AD-10, AD-35): конверт
// DSSE над каноническим payload ключами набора. keyRef — один key_id@версия
// или несколько через запятую: профиль hybrid — два ключа одного субъекта
// (ГОСТ и ML-DSA-65), обе подписи в одном конверте.
type Signer struct {
	Keys *Keyring
}

var _ app.Signer = Signer{}

// Sign подписывает payload класса payloadType ключами keyRef.
func (s Signer) Sign(_ context.Context, payloadType string, payload []byte, keyRef string) ([]byte, error) {
	env, err := s.Envelope(payloadType, payload, strings.Split(keyRef, ",")...)
	if err != nil {
		return nil, err
	}
	return env.Marshal(), nil
}

// Envelope — конверт DSSE с подписями ключей refs по порядку.
func (s Signer) Envelope(payloadType string, payload []byte, refs ...string) (dom.Envelope, error) {
	if _, _, err := dom.ParsePayloadType(payloadType); err != nil {
		return dom.Envelope{}, err
	}
	env := dom.Seal(payloadType, payload)
	pae := dom.PAE(payloadType, payload)
	for _, ref := range refs {
		k, ok := s.Keys.Key(strings.TrimSpace(ref))
		if !ok {
			return dom.Envelope{}, fmt.Errorf("profiles: ключа %s в томе нет", ref)
		}
		sig, err := k.SignPAE(payloadType, pae)
		if err != nil {
			return dom.Envelope{}, err
		}
		env.Signatures = append(env.Signatures, dom.Signature{KeyID: k.Ref, Sig: base64.StdEncoding.EncodeToString(sig)})
	}
	return env, nil
}
