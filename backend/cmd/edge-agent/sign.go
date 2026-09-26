package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.stargrave.org/gogost/v7/gost3410"
	"go.stargrave.org/gogost/v7/gost34112012256"
)

// PayloadTypeEvent — payloadType пакета события (AD-10).
const PayloadTypeEvent = "application/vnd.ant.event+json; v=1"

// Signer — подпись канонического события ключом устройства (AD-10, AD-11).
type Signer interface {
	// Sign возвращает сообщение для ядра: конверт DSSE или (без ключа) само событие.
	Sign(canon []byte) ([]byte, error)
	// KeyRef — key_id@версия ключа устройства (integrity.signers).
	KeyRef() string
	// Signs — подписывает ли (false — профиль demo без ключа).
	Signs() bool
}

// Unsigned — без ключа (профиль demo до эпика 05): ядро примет событие с
// пометкой «подпись не проверялась»; в профиле prod — отказ с кодом.
type Unsigned struct{ Ref string }

// Sign — событие как есть.
func (u Unsigned) Sign(canon []byte) ([]byte, error) { return canon, nil }

// KeyRef — ожидаемый ключ устройства.
func (u Unsigned) KeyRef() string { return u.Ref }

// Signs — нет.
func (Unsigned) Signs() bool { return false }

// GostSigner — ключ устройства ГОСТ Р 34.10-2012, 256 бит,
// id-tc26-gost-3410-12-256-paramSetA; кодирование — contracts/crypto/README.md:
// дайджест Стрибог-256(PAE), число e — little-endian дайджеста (обёртка
// ReverseDigest), подпись s‖r big-endian, открытый ключ X‖Y little-endian.
type GostSigner struct {
	Ref string
	prv *gost3410.PrivateKey
}

func curve() *gost3410.Curve { return gost3410.CurveIdtc26gost341012256paramSetA() }

// PAE — Pre-Authentication Encoding DSSE v1.
func PAE(payloadType string, payload []byte) []byte {
	return []byte("DSSEv1 " + strconv.Itoa(len(payloadType)) + " " + payloadType + " " + strconv.Itoa(len(payload)) + " " + string(payload))
}

func streebog(b []byte) []byte {
	h := gost34112012256.New()
	h.Write(b)
	return h.Sum(nil)
}

// LoadGostKey читает ключ устройства: 64 hex-символа — закрытый ключ little-endian.
func LoadGostKey(path, ref string) (*GostSigner, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return nil, fmt.Errorf("ключ %s: %w", path, err)
	}
	prv, err := gost3410.NewPrivateKeyLE(curve(), raw)
	if err != nil {
		return nil, err
	}
	return &GostSigner{Ref: ref, prv: prv}, nil
}

// GenGostKey создаёт ключ устройства и записывает его в path (0600).
func GenGostKey(path string) (*gost3410.PublicKey, error) {
	prv, err := gost3410.GenPrivateKey(curve(), rand.Reader)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		return nil, errors.New("файл ключа уже есть: " + path)
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(prv.RawLE())+"\n"), 0o600); err != nil {
		return nil, err
	}
	return prv.PublicKey()
}

// PublicKeyB64 — открытый ключ X‖Y little-endian в base64 и его отпечаток (для акта регистрации).
func PublicKeyB64(pub *gost3410.PublicKey) (string, string) {
	raw := pub.RawLE()
	return base64.StdEncoding.EncodeToString(raw), "streebog256:" + hex.EncodeToString(streebog(raw))
}

// Sign — конверт DSSE с подписью ключа устройства.
func (g *GostSigner) Sign(canon []byte) ([]byte, error) {
	sig, err := (&gost3410.PrivateKeyReverseDigest{Prv: g.prv}).Sign(rand.Reader, streebog(PAE(PayloadTypeEvent, canon)), nil)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"payloadType": PayloadTypeEvent,
		"payload":     base64.StdEncoding.EncodeToString(canon),
		"signatures":  []map[string]string{{"keyid": g.Ref, "sig": base64.StdEncoding.EncodeToString(sig)}},
	})
}

// KeyRef — ключ устройства.
func (g *GostSigner) KeyRef() string { return g.Ref }

// Signs — да.
func (*GostSigner) Signs() bool { return true }

// VerifyGost проверяет подпись s‖r над PAE открытым ключом X‖Y little-endian
// (для самопроверки и тестов; на ядре проверку делает порт Verifier, эпик 05).
func VerifyGost(pubLE []byte, pae, sig []byte) (bool, error) {
	pub, err := gost3410.NewPublicKeyLE(curve(), pubLE)
	if err != nil {
		return false, err
	}
	return gost3410.PublicKeyReverseDigest{Pub: pub}.VerifyDigest(streebog(pae), sig)
}
