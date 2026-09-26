package signing

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля signing (AD-36).
type Queries interface {
	// Keys — реестр ключей (signing.key.list, FR-79).
	Keys(ctx context.Context, subjectKind, subjectID, status string, m platform.Moment, p platform.Page) (KeyList, error)
	// Key — ключ и его акты (signing.key.read).
	Key(ctx context.Context, keyRef string, m platform.Moment) (KeyDetails, error)
	// Profiles — криптопрофили (signing.profile.list, AD-32).
	Profiles(ctx context.Context, m platform.Moment) (CryptoProfileList, error)
}

// Commands — ведущий порт команд модуля signing (AD-39).
type Commands interface {
	RegisterKey(ctx context.Context, in RegisterKey) (platform.Receipt, error)
	RevokeKey(ctx context.Context, keyRef string, in RevokeKey) (platform.Receipt, error)
}

// Unimplemented — заглушка портов signing: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Keys(context.Context, string, string, string, platform.Moment, platform.Page) (KeyList, error) {
	return KeyList{}, ni("signing.key.list")
}
func (Unimplemented) Key(context.Context, string, platform.Moment) (KeyDetails, error) {
	return KeyDetails{}, ni("signing.key.read")
}
func (Unimplemented) Profiles(context.Context, platform.Moment) (CryptoProfileList, error) {
	return CryptoProfileList{}, ni("signing.profile.list")
}
func (Unimplemented) RegisterKey(context.Context, RegisterKey) (platform.Receipt, error) {
	return platform.Receipt{}, ni("signing.key.register")
}
func (Unimplemented) RevokeKey(context.Context, string, RevokeKey) (platform.Receipt, error) {
	return platform.Receipt{}, ni("signing.key.revoke")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
