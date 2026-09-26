package engine

import (
	"context"

	"ant/internal/application/signing"
)

// SignerSealer — адаптер порта Sealer над портом подписи (AD-10, AD-11):
// записи движка подписывает системный ключ «движок» (класс server-attested);
// доверие к реакциям даёт пересчёт верификатором, а не ключ (AD-3, AD-9).
type SignerSealer struct {
	Signer signing.Signer
	// KeyRef — ключ движка key_id@версия.
	KeyRef string
}

// Seal подписывает канонический конверт события.
func (s SignerSealer) Seal(ctx context.Context, payload []byte) ([]byte, error) {
	return s.Signer.Sign(ctx, PayloadTypeEvent, payload, s.KeyRef)
}
