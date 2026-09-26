package documents

import (
	"slices"
	"strconv"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Маршрут подписей (AD-13, AD-43): «маршрут закрыт» решает только модуль
// documents после проверки каждой подписи — над текущим отпечатком,
// полномочие и клеймо подписанта, уровень, разделение обязанностей, бумага с
// заверением. Одна функция Evaluate для свёртки изделия, api (объекты вне
// изделия, объяснение «почему не засчитана») и верификатора.

// Причины, по которым подпись не засчитана (verdict.Why).
const (
	WhyPriorVersion   = "prior_version"
	WhyUnknownStage   = "unknown_stage"
	WhyAuthority      = "authority"
	WhyNoStamp        = "no_stamp"
	WhyLevel          = "level"
	WhyDistinct       = "separation_distinct_signers"
	WhyParticipant    = "separation_item_participant"
	WhyPaperForbidden = "paper_forbidden"
	WhyAttesterSigner = "attester_is_signer"
	WhyAttesterAuth   = "attester_authority"
	WhySurplus        = "surplus"
	WhyDeclined       = "declined"
)

// Verdict — засчитана ли подпись и почему нет.
type Verdict struct {
	EventID string `json:"event_id"`
	Counted bool   `json:"counted"`
	Why     string `json:"why,omitempty"`
}

// StageProgress — этап маршрута с засчитанными подписями.
type StageProgress struct {
	Stage   access.ApprovalStage `json:"stage"`
	Counted []Signature          `json:"counted"`
	Done    bool                 `json:"done"`
}

// Evaluation — итог проверки маршрута версии.
type Evaluation struct {
	Stages   []StageProgress `json:"stages"`
	Verdicts []Verdict       `json:"verdicts"`
	Closed   bool            `json:"closed"`
	// Next — первый незакрытый этап (0 — все закрыты).
	Next int `json:"next"`
}

// Evaluate — проверка подписей версии v по замороженному набору (AD-43).
// participants — участники изготовления изделия (FR-56); people — срез
// политики (пусто — полномочия не проверяются, пометка demo).
func Evaluate(v *Version, people People, participants []string) Evaluation {
	ev := Evaluation{}
	for _, st := range v.Stages {
		ev.Stages = append(ev.Stages, StageProgress{Stage: st})
	}
	signedStage := map[string]int{} // человек → этап, где засчитана его подпись
	count := func(i int, s Signature) {
		ev.Stages[i].Counted = append(ev.Stages[i].Counted, s)
		if s.Person != "" {
			if _, ok := signedStage[s.Person]; !ok {
				signedStage[s.Person] = ev.Stages[i].Stage.Stage
			}
		}
	}
	for _, s := range v.SourceSigners {
		if i := stageIndex(v.Stages, s.Stage); i >= 0 && v.Stages[i].BySource {
			count(i, s)
		}
	}
	for _, s := range v.Signatures {
		why := ""
		i := stageIndex(v.Stages, s.Stage)
		switch {
		case s.Version != v.No || s.Digest != v.Digest:
			why = WhyPriorVersion
		case i < 0:
			why = WhyUnknownStage
		default:
			why = checkSigner(v.Stages[i], s, people, participants, signedStage)
			if why == "" && len(ev.Stages[i].Counted) >= v.Stages[i].Required {
				why = WhySurplus
			}
		}
		if why != "" {
			ev.Verdicts = append(ev.Verdicts, Verdict{EventID: s.EventID, Why: why})
			continue
		}
		count(i, s)
		ev.Verdicts = append(ev.Verdicts, Verdict{EventID: s.EventID, Counted: true})
	}
	ev.Closed = len(v.Stages) > 0 && len(v.Declines) == 0 && !v.Annulled
	for i := range ev.Stages {
		ev.Stages[i].Done = len(ev.Stages[i].Counted) >= ev.Stages[i].Stage.Required
		if !ev.Stages[i].Done {
			ev.Closed = false
			if ev.Next == 0 {
				ev.Next = ev.Stages[i].Stage.Stage
			}
		}
	}
	return ev
}

func stageIndex(stages []access.ApprovalStage, n int) int {
	return slices.IndexFunc(stages, func(s access.ApprovalStage) bool { return s.Stage == n })
}

// checkSigner — подпись s на этапе st: полномочие и роль, клеймо, уровень,
// разделение обязанностей, бумага с заверением. Пусто — засчитывается.
func checkSigner(st access.ApprovalStage, s Signature, people People, participants []string, signedStage map[string]int) string {
	if s.Level < st.SignatureLevel {
		return WhyLevel
	}
	if len(people) > 0 {
		p, ok := people.Find(s.Person)
		if !ok || (st.Role != "" && !p.HasRole(st.Role)) || (!p.HasAuthority(st.AuthorityID) && !p.HasRole(st.AuthorityID)) {
			return WhyAuthority
		}
		if st.StampKind != "" && p.StampOf(st.StampKind) == "" {
			return WhyNoStamp
		}
	}
	if st.Has(access.SeparationNotItemParticipant) && slices.Contains(participants, s.Person) {
		return WhyParticipant
	}
	if prev, ok := signedStage[s.Person]; ok && prev != st.Stage && st.Has(access.SeparationDistinctSigners) {
		return WhyDistinct
	}
	if s.Method == MethodPaper {
		switch {
		case !st.PaperAllowed:
			return WhyPaperForbidden
		case s.AttestedBy == "" || s.AttestedBy == s.Person:
			return WhyAttesterSigner
		}
		if len(people) > 0 {
			a, ok := people.Find(s.AttestedBy)
			if !ok || !a.HasAuthority(st.AttesterAuthorityID) {
				return WhyAttesterAuth
			}
		}
	}
	return ""
}

// SignRequest — команда подписи этапа (documents.signature.record,
// documents.paper.attest) в представлении домена.
type SignRequest struct {
	DocumentID string
	Version    int
	Stage      int
	// Digest — отпечаток, который видел подписант (из окна агента или QR скана).
	Digest string
	Person string
	Method string
	Level  int
	// AttestedBy — заверитель бумажной подписи (method = paper).
	AttestedBy string
}

// CheckSign — доменный гард подписи (AD-39, AD-43): версия текущая и не
// аннулирована, отпечаток совпадает, этап открыт и по порядку, подписант
// вправе подписать этап, разделение обязанностей, бумага. Отказ — код из
// contracts/errors.yaml.
func CheckSign(d *Doc, c SignRequest, people People, participants []string) error {
	v, err := openVersion(d, c.Version, c.Stage)
	if err != nil {
		return err
	}
	if c.Digest != v.Digest {
		if c.Method == MethodPaper {
			return kernel.Refuse(errcodes.SigningQrMismatch, "qr_digest", c.Digest, "doc_digest", v.Digest)
		}
		return kernel.Refuse(errcodes.SigningDocumentChanged, "doc_id", d.ID)
	}
	ev := Evaluate(v, people, participants)
	i := stageIndex(v.Stages, c.Stage)
	if i < 0 {
		return notOpen(d, v, c.Stage, "такого этапа нет в маршруте")
	}
	st := v.Stages[i]
	switch {
	case ev.Stages[i].Done:
		return notOpen(d, v, c.Stage, "этап уже подписан")
	case st.BySource:
		return notOpen(d, v, c.Stage, "этап закрывается решением-источником")
	case ev.Next != 0 && ev.Next < c.Stage:
		return notOpen(d, v, c.Stage, "сначала этап "+strconv.Itoa(ev.Next))
	}
	for _, x := range ev.Stages[i].Counted {
		if x.Person == c.Person {
			return notOpen(d, v, c.Stage, "вы уже подписали этот этап")
		}
	}
	signed := map[string]int{}
	for _, sp := range ev.Stages {
		for _, x := range sp.Counted {
			if _, ok := signed[x.Person]; !ok && x.Person != "" {
				signed[x.Person] = sp.Stage.Stage
			}
		}
	}
	s := Signature{Version: v.No, Stage: c.Stage, Person: c.Person, Method: c.Method, Digest: c.Digest, Level: c.Level, AttestedBy: c.AttestedBy}
	switch why := checkSigner(st, s, people, participants, signed); why {
	case "":
		return nil
	case WhyLevel:
		return kernel.Refuse(errcodes.SigningLevelNotAllowed, "action_id", "documents.signature.record", "required", strconv.Itoa(st.SignatureLevel), "actual", strconv.Itoa(c.Level))
	case WhyNoStamp:
		return kernel.Refuse(errcodes.AccessNoStamp, "kind", st.StampKind)
	case WhyParticipant, WhyDistinct:
		r := kernel.Refuse(errcodes.AccessSeparationOfDuties)
		if why == WhyDistinct {
			r.Detail = "Вы уже подписали другой этап этой версии — этапы подписывают разные люди (AD-43)"
		}
		return r
	case WhyPaperForbidden:
		return kernel.Refuse(errcodes.SigningPaperForbidden, "stage", strconv.Itoa(c.Stage))
	case WhyAttesterSigner:
		return kernel.Refuse(errcodes.SigningAttesterIsSigner)
	case WhyAttesterAuth:
		r := kernel.Refuse(errcodes.AccessSignatureRequired, "who", st.AttesterAuthorityID)
		r.Detail = "Заверить бумажную подпись может только сотрудник с полномочием «" + st.AttesterAuthorityID + "» (AD-43)"
		return r
	default:
		who := st.Title
		if who == "" {
			who = st.AuthorityID
		}
		return kernel.Refuse(errcodes.AccessSignatureRequired, "who", who)
	}
}

// CheckDecline — доменный гард отказа в согласовании: версия текущая,
// отпечаток совпадает, этап открыт, подписант вправе подписывать этап.
func CheckDecline(d *Doc, c SignRequest, people People, participants []string) error {
	v, err := openVersion(d, c.Version, c.Stage)
	if err != nil {
		return err
	}
	if c.Digest != "" && c.Digest != v.Digest {
		return kernel.Refuse(errcodes.SigningDocumentChanged, "doc_id", d.ID)
	}
	i := stageIndex(v.Stages, c.Stage)
	if i < 0 || v.Stages[i].BySource {
		return notOpen(d, v, c.Stage, "этап не ждёт подписи")
	}
	if len(people) > 0 {
		st := v.Stages[i]
		p, ok := people.Find(c.Person)
		if !ok || (st.Role != "" && !p.HasRole(st.Role)) || (!p.HasAuthority(st.AuthorityID) && !p.HasRole(st.AuthorityID)) {
			return kernel.Refuse(errcodes.AccessSignatureRequired, "who", st.Title)
		}
	}
	return nil
}

// openVersion — текущая версия, ждущая подписей.
func openVersion(d *Doc, no, stage int) (*Version, error) {
	if d == nil {
		return nil, kernel.Refuse(errcodes.ApiNotFound, "object", "Документ", "id", "")
	}
	cur := d.Current()
	if cur == nil {
		return nil, notOpen(d, &Version{No: no}, stage, "версия документа ещё не сформирована — запросите версию")
	}
	v := d.Version(no)
	switch {
	case v == nil:
		return nil, kernel.Refuse(errcodes.ApiNotFound, "object", "Версия документа", "id", d.ID+"@"+strconv.Itoa(no))
	case v.No != cur.No:
		return nil, kernel.Refuse(errcodes.SigningDocumentChanged, "doc_id", d.ID)
	case v.Annulled:
		return nil, notOpen(d, v, stage, "версия аннулирована")
	case v.Closed:
		return nil, notOpen(d, v, stage, "маршрут уже закрыт")
	case len(v.Declines) > 0:
		return nil, notOpen(d, v, stage, "версия возвращена с замечанием — нужна новая версия")
	}
	return v, nil
}

func notOpen(d *Doc, v *Version, stage int, why string) *kernel.Refusal {
	return kernel.Refuse(errcodes.DocumentStageNotOpen, "doc_id", d.ID, "version", strconv.Itoa(v.No), "stage", strconv.Itoa(stage), "why", why)
}
