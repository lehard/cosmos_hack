package casbin

import (
	"errors"

	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"

	accessdom "ant/internal/domain/access"
)

// Adapter — адаптер политики Casbin (persist.Adapter) над снимком проекции
// политики (AD-15: источник правды — журнал). Только чтение: политика
// меняется записями policy.* журнала, а не через Casbin.
type Adapter struct {
	Policy accessdom.Policy
}

var _ persist.Adapter = Adapter{}

// errReadOnly — запись политики через Casbin запрещена.
var errReadOnly = errors.New("casbin: политика меняется только записями журнала policy.* (AD-15)")

// LoadPolicy загружает строки политики (accessdom.Policy.Rules) в модель.
func (a Adapter) LoadPolicy(m model.Model) error {
	for _, r := range a.Policy.Rules() {
		if err := persist.LoadPolicyArray(append([]string{r.PType}, r.V...), m); err != nil {
			return err
		}
	}
	return nil
}

// SavePolicy — запрещено.
func (Adapter) SavePolicy(model.Model) error { return errReadOnly }

// AddPolicy — запрещено.
func (Adapter) AddPolicy(string, string, []string) error { return errReadOnly }

// RemovePolicy — запрещено.
func (Adapter) RemovePolicy(string, string, []string) error { return errReadOnly }

// RemoveFilteredPolicy — запрещено.
func (Adapter) RemoveFilteredPolicy(string, string, int, ...string) error { return errReadOnly }
