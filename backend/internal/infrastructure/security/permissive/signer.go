package permissive

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"ant/internal/application/signing"
)

// Signer — заглушка порта подписи (ключ signer) до эпика 05: конверт DSSE
// (AD-10) с payload и пустой подписью ключа keyRef. Записи движка с такой
// подписью верификатор отметит «не проверено» (профиль demo); доверие к
// реакциям даёт пересчёт свёртки, а не ключ (AD-3, AD-9).
// TODO(05): заменить адаптером ГОСТ (infrastructure/security/gost).
type Signer struct{}

var _ signing.Signer = Signer{}

// Sign упаковывает payload в DSSE без подписи.
func (Signer) Sign(_ context.Context, payloadType string, payload []byte, keyRef string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"payload":     base64.StdEncoding.EncodeToString(payload),
		"payloadType": payloadType,
		"signatures":  []map[string]string{{"keyid": keyRef, "sig": ""}},
	})
}
