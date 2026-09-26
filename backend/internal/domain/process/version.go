package process

import (
	"encoding/hex"
	"slices"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"

	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Версии процесса (FR-22, FR-23, AD-17): хеш версии — H(байты XML как
// загружены) без канонизации; движок отказывается исполнять версию без
// полного набора действительных подписей кворума (или генезиса для
// стартовой, AD-33) или с изменённым после подписания содержимым.

// Статусы версии (FR-22).
const (
	StatusDraft      = "draft"
	StatusOnApproval = "on_approval"
	StatusActive     = "active"
	StatusRetired    = "retired"
)

// QuorumAuthority — полномочие подписи в кворуме (normative/policy).
const QuorumAuthority = "quorum_signature"

// DefaultQuorum — роли кворума по умолчанию (PRD §8 п. 1, FR-23): технолог-
// автор, контролёр качества с полномочием «подпись в кворуме», руководитель
// производства. Состав настраивается (Approval.Required).
var DefaultQuorum = []string{"technologist", "quality_inspector", "production_manager"}

// VersionHash — хеш версии: H(байты XML как загружены) = streebog256:‹hex›
// (AD-17, AD-44).
func VersionHash(xml []byte) string {
	h := gost34112012256.New()
	h.Write(xml)
	return constants.DigestPrefix + hex.EncodeToString(h.Sum(nil))
}

// Signature — подпись кворума над листом утверждения версии (AD-43): роль и
// сотрудник подписанта, полномочие; Valid — подпись проверена (эпики 05, 27:
// ключ, отзыв, полномочие на seq подписи).
type Signature struct {
	Role      string `json:"role"`
	Person    string `json:"person,omitempty"`
	Authority string `json:"authority"`
	Valid     bool   `json:"valid"`
}

// Approval — утверждение версии: какой хеш утверждён и кем. Genesis —
// стартовая версия, подписанная генезисом (AD-33; демо-трек: стартовая
// версия считается подписанной генезисом, настоящие подписи — эпики 05, 27).
type Approval struct {
	VersionID string `json:"version_id"`
	// Hash — хеш XML, над которым собраны подписи (лист утверждения, AD-12).
	Hash    string `json:"hash"`
	Genesis bool   `json:"genesis,omitempty"`
	// RouteClosedEventID — document.route.closed листа утверждения (AD-43).
	RouteClosedEventID string      `json:"route_closed_event_id,omitempty"`
	Signatures         []Signature `json:"signatures,omitempty"`
	// Required — роли кворума; пусто — DefaultQuorum.
	Required []string `json:"required,omitempty"`
}

// Missing — роли кворума без действительной подписи с полномочием кворума.
func (a Approval) Missing() []string {
	if a.Genesis {
		return nil
	}
	req := a.Required
	if len(req) == 0 {
		req = DefaultQuorum
	}
	var out []string
	for _, role := range req {
		ok := false
		for _, s := range a.Signatures {
			if s.Role == role && s.Valid && s.Authority == QuorumAuthority {
				ok = true
			}
		}
		if !ok {
			out = append(out, role)
		}
	}
	slices.Sort(out)
	return out
}

// Have, Need — сколько подписей кворума есть и сколько нужно.
func (a Approval) Have() (int, int) {
	need := len(a.Required)
	if need == 0 {
		need = len(DefaultQuorum)
	}
	if a.Genesis {
		return need, need
	}
	return need - len(a.Missing()), need
}

// CheckExecutable — можно ли исполнять версию (FR-23, AD-17): байты XML
// хранилища xml должны давать хеш, закреплённый за изделием (pinnedHash) и
// утверждённый подписями (a.Hash); кворум полон. Иначе — отказ
// process.version_tampered или process.quorum_incomplete.
func CheckExecutable(versionID, pinnedHash string, xml []byte, a Approval) *kernel.Refusal {
	got := VersionHash(xml)
	if got != pinnedHash || a.Hash != pinnedHash {
		r := kernel.Refuse(errcodes.ProcessVersionTampered, "version_id", versionID)
		r.Detail = "хеш XML " + got + " не совпадает с подписанным " + a.Hash + " (закреплён за изделием " + pinnedHash + ") — версия не исполняется"
		return r
	}
	if miss := a.Missing(); len(miss) > 0 {
		return kernel.Refuse(errcodes.ProcessQuorumIncomplete, "who", strings.Join(miss, ", "), "version_id", versionID)
	}
	return nil
}
