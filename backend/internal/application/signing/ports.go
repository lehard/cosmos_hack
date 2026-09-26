package signing

import (
	"context"

	"ant/internal/application/journal"
	dom "ant/internal/domain/signing"
)

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

// Crypto — криптографическая проверка каждой подписи конверта (адаптер
// infrastructure/security/profiles.Verifier над реестром ключей этого модуля).
type Crypto interface {
	Check(ctx context.Context, envelope []byte) (dom.Envelope, []byte, []dom.CryptoCheck, error)
	// VerifyRaw — подпись sig над PAE(payloadType, payload) открытым ключом pub
	// профиля profile: доказательство владения новым ключом (AD-11).
	VerifyRaw(profile string, pub []byte, payloadType string, payload, sig []byte) bool
}

// Authorities — ведомый порт полномочий (AD-15, AD-43): есть ли у человека
// полномочие на позиции seq (вторая подпись акта ключа, заверение бумаги).
// Реализация — модуль access (эпик 26); до него — StaticAuthorities по
// стартовой политике normative/policy.
type Authorities interface {
	Has(ctx context.Context, personID, authorityID string, seq int64) (bool, error)
	// Domain — сфера сотрудника для выбора второй подписи: qc / production / admin (AD-11).
	Domain(ctx context.Context, personID string) string
}

// Scans — ведомый порт хранилища материалов (AD-23): байты скана по адресу H(байты).
type Scans interface {
	Scan(ctx context.Context, address string) ([]byte, error)
}

// QRReader — ведомый порт чтения QR со скана (infrastructure/integration/signing/paperscan, gozxing).
type QRReader interface {
	ReadQR(image []byte) (string, error)
}

// QRWriter — ведомый порт рисования QR печатной рамки (Д-30: QR рисует сервер).
type QRWriter interface {
	SVG(text string) (string, error)
}

// Alerts — ведомый порт шины безопасности (AD-24): тревоги по ключу
// security.key.alert (эмитент — модуль security, эпик 29) — записи для той же
// пачки Append. nil — тревога только возвращается вызывающему.
type Alerts interface {
	KeyAlert(ctx context.Context, a dom.KeyAlert) ([]journal.Pending, error)
}
