package signing

import "context"

// Signer — ведомый порт подписи (AD-10, AD-11, AD-35, ключ signer): подписи
// людей — только в агенте токена (сервер ключей людей не хранит), демо-персон —
// demo-signer, системные (движок, шлюзы) — ключи в томе ant. PKCS#11 и
// сертифицированное СКЗИ — замена.
type Signer interface {
	// Sign подписывает канонический payload ключом keyRef по профилю и
	// возвращает конверт DSSE (AD-10).
	Sign(ctx context.Context, payloadType string, payload []byte, keyRef string) ([]byte, error)
}

// Verifier — ведомый порт проверки подписи (AD-10, ключ verifier): профиль —
// по реестру на момент подписи; понижение профиля — отказ.
type Verifier interface {
	// Verify проверяет конверт DSSE и возвращает подписантов, прошедших проверку.
	Verify(ctx context.Context, envelope []byte) (Verified, error)
}

// Verified — результат проверки конверта.
type Verified struct {
	PayloadType string
	Payload     []byte
	// Signers — key_id@версия и профиль каждой действительной подписи.
	Signers []SignerRef
}

// SignerRef — подписант.
type SignerRef struct {
	KeyRef  string
	Profile string
}

// Cipher — ведомый порт шифрования при хранении (AD-23, ключ cipher):
// конвертная схема (DEK на объект, обёртки KEK), AEAD «Кузнечик»-MGM или
// AES-256-GCM; в дополнительные данные — цепочка, seq, commit.
type Cipher interface {
	Seal(ctx context.Context, plaintext, additional []byte) (sealed []byte, keyRef string, err error)
	Open(ctx context.Context, sealed, additional []byte, keyRef string) ([]byte, error)
}
