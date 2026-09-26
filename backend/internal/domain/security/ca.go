package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Main — основная запись, для которой строится запись CA (AD-28): её
// идентификатор, тип, поток и обязательство commit из той же транзакции
// journal.Append — поэтому ссылка CA на подписанную запись точна.
type Main struct {
	EventID   string
	EventType catalog.Type
	Stream    string
	Commit    string
	// Data — data основной записи (уточнить «было → стало»).
	Data json.RawMessage
	// Signers — подписанты конверта (integrity.signers): кто, если команда не сказала.
	Signers []string
	// Corrects — исправляемая запись (corrects.event_id), если это исправление.
	Corrects string
	// Causation — причина записи (основание по умолчанию).
	Causation string
}

// Command — сведения команды критического действия (AD-28): кто, полномочие
// и клеймо с ревизией политики, основание, материалы, отмена прежней CA.
// Пустые поля — по основной записи.
type Command struct {
	ActorID     string
	AttestedBy  string
	AuthorityID string
	StampID     string
	PolicySeq   int64
	Basis       []string
	// Before, After — было → стало, если его знает модуль-владелец.
	Before, After string
	// Cancels — отменяемая запись CA-‹n› (отмена — только новой записью с причиной).
	Cancels      string
	CancelReason string
}

// Record — data записи security.critical_action.recorded
// (contracts/events/security/security.critical_action.recorded.v1.json).
type Record struct {
	CANo          int64    `json:"ca_no"`
	CAGroup       string   `json:"ca_group"`
	ActionType    string   `json:"action_type"`
	MainEventID   string   `json:"main_event_id"`
	MainCommit    string   `json:"main_commit"`
	ObjectRef     string   `json:"object_ref"`
	Before        string   `json:"before"`
	After         string   `json:"after"`
	ActorID       string   `json:"actor_id,omitempty"`
	AttestedBy    string   `json:"attested_by,omitempty"`
	AuthorityID   string   `json:"authority_id,omitempty"`
	StampID       string   `json:"stamp_id,omitempty"`
	PolicySeq     int64    `json:"policy_seq,omitempty"`
	BasisEventIDs []string `json:"basis_event_ids"`
	Cancels       string   `json:"cancels,omitempty"`
	CancelReason  *Reason  `json:"cancel_reason,omitempty"`
}

// Reason — причина (common/defs.v1.json#/definitions/reason).
type Reason struct {
	Text string `json:"text"`
}

// Ref — номер CA-‹n›.
func Ref(no int64) string { return "CA-" + strconv.FormatInt(no, 10) }

var (
	reCARef  = regexp.MustCompile(`^CA-[1-9][0-9]*$`)
	rePerson = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	reUUID   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// ErrNotCritical — тип записи не критический: запись CA не нужна.
var ErrNotCritical = errors.New("security: тип записи не критический")

// BuildCA — запись журнала критических действий для основной записи (AD-28):
// действие, объект, было → стало, кто, полномочие и клеймо с ревизией
// политики, основание, ссылка на основную запись (event_id, commit). no —
// номер CA-‹n›: позиция в цепочке ca, её даёт journal.Append в той же
// транзакции. Чистая функция: её вызывают Append (через application/security)
// и независимый верификатор при сверке CA с решением один к одному (AD-9).
func BuildCA(m Main, no int64, cmd Command) (Record, error) {
	a, info, ok := ActionFor(m.EventType)
	if !ok {
		return Record{}, fmt.Errorf("%w: %s", ErrNotCritical, m.EventType)
	}
	if no < 1 {
		return Record{}, fmt.Errorf("security: номер CA %d", no)
	}
	if !reUUID.MatchString(m.EventID) {
		return Record{}, fmt.Errorf("security: event_id основной записи %q", m.EventID)
	}
	before, after := refine(a, m.Data)
	if m.Corrects != "" {
		before, after = "запись "+m.Corrects, CorrectionName+": "+after
	}
	if cmd.Before != "" {
		before = cmd.Before
	}
	if cmd.After != "" {
		after = cmd.After
	}
	r := Record{
		CANo: no, CAGroup: info.CAGroup, ActionType: string(m.EventType), MainEventID: m.EventID, MainCommit: m.Commit,
		ObjectRef: m.Stream, Before: clip(before), After: clip(after), AuthorityID: cmd.AuthorityID, StampID: cmd.StampID,
		PolicySeq: cmd.PolicySeq, BasisEventIDs: []string{},
	}
	r.ActorID = person(cmd.ActorID)
	if r.ActorID == "" && len(m.Signers) > 0 {
		r.ActorID = person(m.Signers[0])
	}
	r.AttestedBy = person(cmd.AttestedBy)
	for _, b := range append(slices.Clone(cmd.Basis), m.Causation, m.Corrects) {
		if reUUID.MatchString(b) && b != m.EventID && !slices.Contains(r.BasisEventIDs, b) {
			r.BasisEventIDs = append(r.BasisEventIDs, b)
		}
	}
	if cmd.Cancels != "" {
		if !reCARef.MatchString(cmd.Cancels) {
			return Record{}, fmt.Errorf("security: отменяемая запись %q — ожидается CA-‹n›", cmd.Cancels)
		}
		if strings.TrimSpace(cmd.CancelReason) == "" {
			return Record{}, errors.New("security: отмена CA — только с причиной (AD-28)")
		}
		if cmd.Cancels == Ref(no) {
			return Record{}, errors.New("security: запись CA не отменяет сама себя")
		}
		r.Cancels, r.CancelReason = cmd.Cancels, &Reason{Text: clip(cmd.CancelReason)}
	}
	return r, nil
}

// EventID — event_id записи CA: UUIDv5(NS_ANT, «ca» ‖ event_id основной
// записи) — одна основная запись даёт одну запись CA и при повторе Append.
func EventID(mainEventID string) string {
	return kernel.UUIDv5(constants.NsAnt, "security.critical_action\x1f"+mainEventID)
}

// person — псевдоним исполнителя без версии ключа (`O17@1` → `O17`).
func person(s string) string {
	s, _, _ = strings.Cut(s, "@")
	if !rePerson.MatchString(s) {
		return ""
	}
	return s
}

func clip(s string) string {
	if s == "" {
		return "—"
	}
	r := []rune(s)
	if len(r) > 4000 {
		return string(r[:4000])
	}
	return s
}
