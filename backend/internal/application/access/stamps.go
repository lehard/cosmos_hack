package access

import (
	"context"
	"time"
)

// StampRegistry — реестр действующих цифровых клейм контролёров (FR-145,
// AD-15; эпик 37) над проекцией политики: клеймо выдаётся по приказу, одно
// на вид контроля, с областью и сроком, отзывается записью policy.stamp.revoked.
// Им пользуется documents: подпись этапа с видом контроля без действующего
// клейма отклоняется (access.no_stamp).
type StampRegistry struct {
	Policy PolicySource
}

// ValidStamp — действующее в момент at клеймо сотрудника по виду контроля kind.
func (r StampRegistry) ValidStamp(ctx context.Context, personID, kind string, at time.Time) (string, bool, error) {
	pol, err := r.Policy.Policy(ctx)
	if err != nil {
		return "", false, err
	}
	st, ok := pol.StampFor(personID, kind, "", at)
	return st.StampID, ok, nil
}
