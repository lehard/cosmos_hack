package ops

import (
	"fmt"
	"strings"
	"time"
)

// Состояния компонента (сервиса или роли процесса) на столе администратора.
const (
	StateOK       = "ok"
	StateDegraded = "degraded"
	StateDown     = "down"
	StateUnknown  = "unknown"
)

// Имена аренд (AD-6): партиции воркера, члены группы воркеров, роли-лидеры.
const (
	LeasePartitionPrefix = "partition:"
	LeaseWorkerPrefix    = "worker:"
)

// LeaderRoles — роли с одной копией-лидером по аренде (AD-6, AD-45): имя
// аренды совпадает с именем роли.
var LeaderRoles = []string{"crossitem", "projector", "scheduler", "outbox", "security"}

// Lease — аренда из хранилища аренд (действующая или истёкшая).
type Lease struct {
	Name      string
	Holder    string
	Epoch     int64
	ExpiresAt time.Time
}

// Component — состояние роли процесса по арендам.
type Component struct {
	Name      string
	State     string
	Instances int
	Leader    string
	Detail    string
}

// Components — состояние ролей по арендам на момент now (AD-6, FR-127):
// воркеры — по членству в группе и покрытию partitions партиций; роли-лидеры
// — по аренде лидера. Аренды не было никогда — роль не запускалась (unknown),
// аренда истекла — роль остановлена (down).
func Components(leases []Lease, now time.Time, partitions int) []Component {
	live := func(l Lease) bool { return l.ExpiresAt.After(now) }
	var members, held, seenWorker int
	byName := map[string]Lease{}
	for _, l := range leases {
		byName[l.Name] = l
		switch {
		case strings.HasPrefix(l.Name, LeaseWorkerPrefix):
			seenWorker++
			if live(l) {
				members++
			}
		case strings.HasPrefix(l.Name, LeasePartitionPrefix):
			seenWorker++
			if live(l) {
				held++
			}
		}
	}
	w := Component{Name: "ant/worker", Instances: members}
	switch {
	case seenWorker == 0:
		w.State, w.Detail = StateUnknown, "воркеры не запускались"
	case members == 0:
		w.State, w.Detail = StateDown, "нет действующих копий воркера"
	case partitions > 0 && held < partitions:
		w.State, w.Detail = StateDegraded, fmt.Sprintf("арендовано партиций %d из %d", held, partitions)
	default:
		w.State, w.Detail = StateOK, fmt.Sprintf("копий %d, партиций %d", members, held)
	}
	out := []Component{w}
	for _, r := range LeaderRoles {
		c := Component{Name: "ant/" + r}
		l, ok := byName[r]
		switch {
		case !ok:
			c.State, c.Detail = StateUnknown, "роль не запускалась"
		case live(l):
			c.State, c.Instances, c.Leader = StateOK, 1, l.Holder
			c.Detail = fmt.Sprintf("лидер, эпоха %d", l.Epoch)
		default:
			c.State, c.Leader = StateDown, l.Holder
			c.Detail = "аренда лидера истекла " + l.ExpiresAt.UTC().Format(time.RFC3339)
		}
		out = append(out, c)
	}
	return out
}
