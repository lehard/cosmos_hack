package analysis

import (
	"maps"
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Глобальные проекции модуля analysis (AD-45, писатель — analysis): чистые
// свёртки `(значение по ключу, запись) → значение`. Их ведёт роль projector,
// пересобирает `ant rebuild`, тот же код может исполнить верификатор (AD-9).
//
//   - analysis.incident — инцидент по incident_id: версии области с
//     основаниями, изделия с двумя осями статуса, решения людей, причина,
//     гипотезы людей, измерения, меры; ключ ListKey — список инцидентов;
//   - analysis.nc — несоответствие по nc_id: изделие, вид дефекта, версии
//     вывода разбора, гипотезы людей, причина; ключ ListKey — список.

// ListKey — ключ списка в глобальных проекциях analysis.
const ListKey = "_list"

// ReasonRecord — основание: код и текст.
type ReasonRecord struct {
	Code string `json:"code,omitempty"`
	Text string `json:"text"`
}

// VersionRecord — версия области риска (FR-61): что изменилось, кем, почему.
type VersionRecord struct {
	Version    int           `json:"version"`
	Change     string        `json:"change"`
	Size       int           `json:"size"`
	RecordedAt time.Time     `json:"recorded_at"`
	Author     string        `json:"author,omitempty"`
	Reason     *ReasonRecord `json:"reason,omitempty"`
	Evidence   []string      `json:"evidence,omitempty"`
	Breakdown  Breakdown     `json:"breakdown"`
	Added      []string      `json:"added,omitempty"`
	Removed    []string      `json:"removed,omitempty"`
	DecisionID string        `json:"decision_id,omitempty"`
	EventID    string        `json:"event_id"`
}

// MemberRecord — изделие в области: две оси статуса (FR-62).
type MemberRecord struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Via     string `json:"via,omitempty"`
	Version int    `json:"version"`
}

// DecisionRecord — решение человека над областью (сужение, расширение, оценка).
type DecisionRecord struct {
	Type       string       `json:"type"`
	Actor      string       `json:"actor,omitempty"`
	At         time.Time    `json:"at"`
	Reason     ReasonRecord `json:"reason"`
	Evidence   []string     `json:"evidence,omitempty"`
	Items      []string     `json:"items,omitempty"`
	Assessment string       `json:"assessment,omitempty"`
}

// CauseRecord — вывод о причине (решение человека, FR-59).
type CauseRecord struct {
	IncidentID   string       `json:"incident_id"`
	Branch       string       `json:"branch,omitempty"`
	Conclusion   string       `json:"conclusion"`
	Category     string       `json:"category,omitempty"`
	NCIDs        []string     `json:"nc_ids,omitempty"`
	Verification string       `json:"verification,omitempty"`
	Reason       ReasonRecord `json:"reason"`
	Actor        string       `json:"actor,omitempty"`
	At           time.Time    `json:"at"`
	EventID      string       `json:"event_id"`
}

// HumanHypothesis — гипотеза, записанная или отклонённая человеком (FR-59).
type HumanHypothesis struct {
	HypothesisID string        `json:"hypothesis_id"`
	IncidentID   string        `json:"incident_id"`
	NCIDs        []string      `json:"nc_ids,omitempty"`
	Branch       string        `json:"branch,omitempty"`
	Category     string        `json:"category"`
	Statement    string        `json:"statement,omitempty"`
	Verdict      string        `json:"verdict"`
	Reason       *ReasonRecord `json:"reason,omitempty"`
	Supporting   []string      `json:"supporting,omitempty"`
	Actor        string        `json:"actor,omitempty"`
	At           time.Time     `json:"at"`
	EventID      string        `json:"event_id"`
}

// MeasurementRecord — запрос измерения для проверки гипотезы.
type MeasurementRecord struct {
	HypothesisID string    `json:"hypothesis_id"`
	What         string    `json:"what"`
	Assignee     string    `json:"assignee,omitempty"`
	At           time.Time `json:"at"`
	EventID      string    `json:"event_id"`
	// Result — записанный результат измерения (incident.measurement.recorded).
	Result *MeasurementResult `json:"result,omitempty"`
}

// MeasurementResult — итог измерения для гипотезы.
type MeasurementResult struct {
	Outcome string    `json:"outcome"`
	Text    string    `json:"text"`
	At      time.Time `json:"at"`
	EventID string    `json:"event_id"`
	Actor   string    `json:"actor,omitempty"`
}

// Результат измерения для гипотезы.
const (
	MeasurementSupports     = "supports"
	MeasurementRefutes      = "refutes"
	MeasurementInconclusive = "inconclusive"
)

// withResult — результат измерения к своему запросу (request_event_id; нет —
// последний запрос по гипотезе без результата).
func withResult(ms []MeasurementRecord, r kernel.Record) []MeasurementRecord {
	d, ok := decodeAs[struct {
		HypothesisID   string `json:"hypothesis_id"`
		RequestEventID string `json:"request_event_id"`
		Outcome        string `json:"outcome"`
		Result         string `json:"result"`
	}](r)
	if !ok {
		return ms
	}
	out := slices.Clone(ms)
	for i := len(out) - 1; i >= 0; i-- {
		m := &out[i]
		if (d.RequestEventID != "" && m.EventID == d.RequestEventID) || (d.RequestEventID == "" && m.HypothesisID == d.HypothesisID && m.Result == nil) {
			m.Result = &MeasurementResult{Outcome: d.Outcome, Text: d.Result, At: r.OccurredAt, EventID: r.EventID, Actor: r.Actor}
			break
		}
	}
	return out
}

// ActionRecord — мера по инциденту (FR-64): assigned → implemented → effective | failed.
type ActionRecord struct {
	ActionID   string `json:"action_id"`
	ActionType string `json:"action_type,omitempty"`
	Direction  string `json:"direction,omitempty"`
	Owner      string `json:"owner,omitempty"`
	Status     string `json:"status"`
}

// IncidentRecord — значение проекции analysis.incident.
type IncidentRecord struct {
	IncidentID      string                    `json:"incident_id"`
	Label           string                    `json:"label,omitempty"`
	Factor          string                    `json:"factor,omitempty"`
	FactorValue     string                    `json:"factor_value,omitempty"`
	StepKey         string                    `json:"step_key,omitempty"`
	OpenedAt        time.Time                 `json:"opened_at"`
	TriggerEventIDs []string                  `json:"trigger_event_ids,omitempty"`
	WindowStart     *time.Time                `json:"window_start,omitempty"`
	WindowEnd       *time.Time                `json:"window_end,omitempty"`
	KnownGoodEvent  string                    `json:"known_good_event,omitempty"`
	KnownGoodItem   string                    `json:"known_good_item,omitempty"`
	Versions        []VersionRecord           `json:"versions,omitempty"`
	Members         map[string]MemberRecord   `json:"members,omitempty"`
	Decisions       map[string]DecisionRecord `json:"decisions,omitempty"`
	InitialSize     int                       `json:"initial_size"`
	Closed          bool                      `json:"closed,omitempty"`
	ClosedAt        *time.Time                `json:"closed_at,omitempty"`
	Cause           *CauseRecord              `json:"cause,omitempty"`
	OperatorError   bool                      `json:"operator_error,omitempty"`
	FullAnalysis    *bool                     `json:"full_analysis,omitempty"`
	Hypotheses      []HumanHypothesis         `json:"hypotheses,omitempty"`
	Measurements    []MeasurementRecord       `json:"measurements,omitempty"`
	Actions         []ActionRecord            `json:"actions,omitempty"`
	// Causes — выводы о причине по веткам why_made / why_missed (кейс §2.3).
	Causes map[string]CauseRecord `json:"causes,omitempty"`
	// InvestigationClosed — расследование закрыто (incident.incident.closed, scope=investigation).
	InvestigationClosed bool `json:"investigation_closed,omitempty"`
	// LastEventAt — время последней записи по инциденту (шапка расследования).
	LastEventAt *time.Time `json:"last_event_at,omitempty"`
	// BasisSeq — seq последней записи потока инцидента: basis_seq команд (AD-39).
	BasisSeq int64 `json:"basis_seq"`
}

// Size — размер текущей области.
func (v IncidentRecord) Size() int {
	if n := len(v.Versions); n > 0 {
		return v.Versions[n-1].Size
	}
	return 0
}

// Version — номер текущей версии области.
func (v IncidentRecord) Version() int {
	if n := len(v.Versions); n > 0 {
		return v.Versions[n-1].Version
	}
	return 0
}

// InScope — изделие в текущей области и не исключено.
func (v IncidentRecord) InScope(itemID string) bool {
	m, ok := v.Members[itemID]
	return ok && m.Status != StatusExcluded
}

// ListRecord — значение ключа ListKey: идентификаторы в порядке появления.
type ListRecord struct {
	IDs []string `json:"ids"`
}

// NCRecord — значение проекции analysis.nc.
type NCRecord struct {
	NCID         string              `json:"nc_id"`
	ItemID       string              `json:"item_id,omitempty"`
	DefectType   string              `json:"defect_type,omitempty"`
	Severity     string              `json:"severity,omitempty"`
	EventID      string              `json:"event_id,omitempty"`
	At           time.Time           `json:"at"`
	Seq          int64               `json:"seq,omitempty"`
	Versions     int                 `json:"versions"`
	Hypotheses   []HumanHypothesis   `json:"hypotheses,omitempty"`
	Measurements []MeasurementRecord `json:"measurements,omitempty"`
	Cause        *CauseRecord        `json:"cause,omitempty"`
	IncidentIDs  []string            `json:"incident_ids,omitempty"`
	// Computed — версии вывода разбора: уверенность по категориям (история гипотез).
	Computed []ComputedVersion `json:"computed,omitempty"`
}

// ComputedVersion — версия вывода incident.hypothesis.computed: уверенность
// гипотез по категориям — «что изменило уверенность» (стол технолога).
type ComputedVersion struct {
	At         time.Time      `json:"at"`
	EventID    string         `json:"event_id"`
	Confidence map[string]int `json:"confidence,omitempty"`
}

// ActorName — псевдоним автора решения из key_id@версия первого подписанта
// (AD-10): до реестра ключей (эпики 05, 27) ключ демо-решения — псевдоним в
// нижнем регистре.
func ActorName(actor string) string {
	k, _, _ := strings.Cut(actor, "@")
	return strings.ToUpper(k)
}

// IncidentKeys — ключи проекции analysis.incident, которые меняет запись.
func IncidentKeys(r kernel.Record) []string {
	if family(r.Type) != "incident" || r.Type == catalog.IncidentHypothesisComputed || r.Type == catalog.IncidentSuggestionRecorded {
		return nil
	}
	id := incidentOf(r)
	if id == "" {
		return nil
	}
	if r.Type == catalog.IncidentIncidentOpened {
		return []string{id, ListKey}
	}
	return []string{id}
}

// StepIncident — шаг проекции analysis.incident по ключу key.
func StepIncident(key string, v IncidentRecord, r kernel.Record) IncidentRecord {
	if key == ListKey {
		return v // список ведёт StepList
	}
	if v.IncidentID == "" {
		v.IncidentID = key
	}
	if r.Stream == "incident:"+key && r.Seq > v.BasisSeq {
		v.BasisSeq = r.Seq
	}
	if at := r.OccurredAt; !at.IsZero() && (v.LastEventAt == nil || at.After(*v.LastEventAt)) {
		v.LastEventAt = &at
	}
	actor := ActorName(r.Actor)
	switch r.Type {
	case catalog.IncidentIncidentOpened:
		d, _ := decodeAs[struct {
			CommonFactor string   `json:"common_factor"`
			FactorRef    string   `json:"factor_ref"`
			StepKey      string   `json:"step_key"`
			Trigger      []string `json:"trigger_event_ids"`
		}](r)
		v.Factor, v.FactorValue, v.StepKey, v.OpenedAt = FactorOfCommon(d.CommonFactor), d.FactorRef, d.StepKey, r.OccurredAt
		v.TriggerEventIDs = d.Trigger
		v.Label = "Инцидент " + d.FactorRef
		if v.Factor == FactorMaterialBatch {
			v.Label = "Входной брак партии " + d.FactorRef
		}
	case catalog.IncidentScopeComputed:
		d, ok := decodeAs[scopeComputedRead](r)
		if !ok {
			return v
		}
		ws, we := d.WindowStart, d.WindowEnd
		v.WindowStart, v.WindowEnd = &ws, &we
		if d.LastKnownGoodItemID != "" || d.LastKnownGoodEventID != "" {
			v.KnownGoodEvent, v.KnownGoodItem = d.LastKnownGoodEventID, d.LastKnownGoodItemID
		}
		change := d.Change
		if change == "" {
			change = ChangeComputed
		}
		at := r.RecordedAt
		if at.IsZero() {
			at = r.OccurredAt
		}
		x := VersionRecord{Version: d.ScopeVersion, Change: change, Size: d.Size, RecordedAt: at, Breakdown: d.Breakdown,
			Added: d.Added, Removed: d.Removed, DecisionID: d.DecisionEventID, EventID: r.EventID, Evidence: []string{}}
		if dec, ok := v.Decisions[d.DecisionEventID]; ok && d.DecisionEventID != "" {
			x.Author, x.Evidence = dec.Actor, append([]string{d.DecisionEventID}, dec.Evidence...)
			reason := dec.Reason
			x.Reason = &reason
		} else {
			x.Evidence = slices.DeleteFunc(slices.Clone(d.Basis), func(s string) bool { return s == "" })
			x.Reason = &ReasonRecord{Code: "rule", Text: ruleReason(v, d)}
		}
		v.Versions = append(slices.Clone(v.Versions), x)
		if d.ScopeVersion == 1 || v.InitialSize == 0 {
			v.InitialSize = d.Size
		}
	case catalog.IncidentMembershipChanged:
		d, ok := decodeAs[membershipData](r)
		if !ok || r.ItemID == "" {
			return v
		}
		m := MemberRecord{Status: d.Status, Action: d.Action, Via: deref(d.ViaAssemblyOf), Version: d.ScopeVersion}
		if prev, ok := v.Members[r.ItemID]; !ok || m.Version >= prev.Version {
			v.Members = cloneMap(v.Members)
			v.Members[r.ItemID] = m
		}
	case catalog.IncidentScopeNarrowed:
		d, _ := kernel.Decode[ev.IncidentScopeNarrowedV1](r)
		v = putDecision(v, r, DecisionRecord{Type: string(r.Type), Actor: actor, At: r.OccurredAt, Reason: reasonOf(d.Reason),
			Evidence: uuidStrings(d.EvidenceEventIds), Items: itemStrings(d.ItemIds)})
	case catalog.IncidentScopeExpanded:
		d, _ := kernel.Decode[ev.IncidentScopeExpandedV1](r)
		v = putDecision(v, r, DecisionRecord{Type: string(r.Type), Actor: actor, At: r.OccurredAt, Reason: reasonOf(d.Reason),
			Items: itemStrings(d.ItemIds)})
	case catalog.IncidentItemAssessed:
		d, _ := kernel.Decode[ev.IncidentItemAssessedV1](r)
		v = putDecision(v, r, DecisionRecord{Type: string(r.Type), Actor: actor, At: r.OccurredAt,
			Reason:   ReasonRecord{Code: "assessed:" + string(d.Assessment), Text: "Проверка изделия " + LocalID(string(d.ItemID)) + ": " + assessmentText(string(d.Assessment))},
			Evidence: uuidStrings(d.EvidenceEventIds), Items: []string{string(d.ItemID)}, Assessment: string(d.Assessment)})
	case catalog.IncidentIncidentClosed:
		at := r.OccurredAt
		if !v.Closed {
			v.Closed, v.ClosedAt = true, &at
		}
		if d, _ := decodeAs[struct {
			Scope string `json:"scope"`
		}](r); d.Scope == CloseInvestigation {
			v.InvestigationClosed = true
		}
	case catalog.IncidentCauseConcluded:
		c := causeOf(r)
		v.Cause = &c
		v.Causes = cloneMap(v.Causes)
		v.Causes[c.Branch] = c
	case catalog.IncidentOperatorErrorConfirmed:
		v.OperatorError = true
	case catalog.IncidentAnalysisScoped:
		d, _ := kernel.Decode[ev.IncidentAnalysisScopedV1](r)
		full := d.FullAnalysis
		v.FullAnalysis = &full
	case catalog.IncidentHypothesisRecorded:
		v.Hypotheses = append(slices.Clone(v.Hypotheses), humanHypothesisOf(r))
	case catalog.IncidentMeasurementRequested:
		v.Measurements = append(slices.Clone(v.Measurements), measurementOf(r))
	case catalog.IncidentMeasurementRecorded:
		v.Measurements = withResult(v.Measurements, r)
	case catalog.IncidentActionAssigned:
		d, _ := kernel.Decode[ev.IncidentActionAssignedV1](r)
		v.Actions = append(slices.Clone(v.Actions), ActionRecord{ActionID: string(d.ActionID), ActionType: string(d.ActionType),
			Direction: string(d.Direction), Owner: string(d.OwnerID), Status: "assigned"})
	case catalog.IncidentActionImplemented:
		d, _ := kernel.Decode[ev.IncidentActionImplementedV1](r)
		v.Actions = setAction(v.Actions, string(d.ActionID), "implemented")
	case catalog.IncidentActionEvaluated:
		d, _ := kernel.Decode[ev.IncidentActionEvaluatedV1](r)
		v.Actions = setAction(v.Actions, string(d.ActionID), string(d.Result))
	}
	return v
}

// scopeComputedRead — чтение incident.scope.computed.
type scopeComputedRead struct {
	ScopeVersion         int       `json:"scope_version"`
	WindowStart          time.Time `json:"window_start"`
	WindowEnd            time.Time `json:"window_end"`
	LastKnownGoodEventID string    `json:"last_known_good_event_id"`
	LastKnownGoodItemID  string    `json:"last_known_good_item_id"`
	Added                []string  `json:"added_item_ids"`
	Removed              []string  `json:"removed_item_ids"`
	Size                 int       `json:"size"`
	Breakdown            Breakdown `json:"breakdown"`
	Basis                []string  `json:"basis"`
	Change               string    `json:"change"`
	DecisionEventID      string    `json:"decision_event_id"`
}

// ruleReason — основание версии, вычисленной правилом системы.
func ruleReason(v IncidentRecord, d scopeComputedRead) string {
	if d.ScopeVersion == 1 || len(v.Versions) == 0 {
		s := "Правило системы: общий фактор " + v.FactorValue
		if d.LastKnownGoodItemID != "" {
			s += "; отсчёт — после последней подтверждённо годной детали " + LocalID(d.LastKnownGoodItemID)
		} else if v.Factor != FactorMaterialBatch {
			s += "; подтверждённо годной детали через тот же фактор нет — окно от первого известного выполнения"
		}
		return s
	}
	var items []string
	for _, id := range d.Added {
		items = append(items, LocalID(id))
	}
	return "Правило системы расширило область (защитное действие): " + strings.Join(items, ", ")
}

func putDecision(v IncidentRecord, r kernel.Record, d DecisionRecord) IncidentRecord {
	v.Decisions = cloneMap(v.Decisions)
	v.Decisions[r.EventID] = d
	return v
}

func assessmentText(a string) string {
	if a == "excluded" {
		return "исключено"
	}
	return "подтверждено"
}

func setAction(as []ActionRecord, id, status string) []ActionRecord {
	as = slices.Clone(as)
	for i := range as {
		if as[i].ActionID == id {
			as[i].Status = status
		}
	}
	return as
}

func causeOf(r kernel.Record) CauseRecord {
	d, _ := kernel.Decode[ev.IncidentCauseConcludedV1](r)
	c := CauseRecord{IncidentID: string(d.IncidentID), Conclusion: string(d.Conclusion), Reason: reasonOf(d.Reason),
		Actor: ActorName(r.Actor), At: r.OccurredAt, EventID: r.EventID, Branch: BranchWhyMade}
	if b, _ := decodeAs[struct {
		Branch string `json:"branch"`
	}](r); b.Branch != "" {
		c.Branch = b.Branch
	}
	if d.Category != nil {
		c.Category = string(*d.Category)
	}
	c.Verification = d.Verification
	for _, n := range d.NcIds {
		c.NCIDs = append(c.NCIDs, string(n))
	}
	return c
}

func humanHypothesisOf(r kernel.Record) HumanHypothesis {
	d, _ := decodeAs[struct {
		IncidentID   string   `json:"incident_id"`
		NCIDs        []string `json:"nc_ids"`
		Branch       string   `json:"branch"`
		Category     string   `json:"category"`
		Statement    string   `json:"statement"`
		Supporting   []string `json:"supporting_event_ids"`
		HypothesisID string   `json:"hypothesis_id"`
		Verdict      string   `json:"verdict"`
		Reason       *struct {
			Code string `json:"code"`
			Text string `json:"text"`
		} `json:"reason"`
	}](r)
	h := HumanHypothesis{HypothesisID: d.HypothesisID, IncidentID: d.IncidentID, NCIDs: d.NCIDs, Branch: d.Branch, Category: d.Category,
		Statement: d.Statement, Verdict: d.Verdict, Supporting: d.Supporting, Actor: ActorName(r.Actor), At: r.OccurredAt, EventID: r.EventID}
	if h.Verdict == "" {
		h.Verdict = "proposed"
	}
	if h.HypothesisID == "" {
		h.HypothesisID = r.EventID
	}
	if d.Reason != nil {
		h.Reason = &ReasonRecord{Code: d.Reason.Code, Text: d.Reason.Text}
	}
	return h
}

func measurementOf(r kernel.Record) MeasurementRecord {
	d, _ := kernel.Decode[ev.IncidentMeasurementRequestedV1](r)
	m := MeasurementRecord{HypothesisID: string(d.HypothesisID), What: d.What, At: r.OccurredAt, EventID: r.EventID}
	if d.AssigneeID != nil {
		m.Assignee = string(*d.AssigneeID)
	}
	return m
}

// NCKeys — ключи проекции analysis.nc, которые меняет запись.
func NCKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.DecisionNonconformityConfirmed:
		d, ok := decodeAs[ncConfirmedData](r)
		if ok && d.NcID != "" {
			return []string{d.NcID, ListKey}
		}
	case catalog.IncidentHypothesisComputed:
		d, ok := decodeAs[struct {
			NcID string `json:"nc_id"`
		}](r)
		if ok && d.NcID != "" {
			return []string{d.NcID}
		}
	case catalog.IncidentHypothesisRecorded, catalog.IncidentCauseConcluded, catalog.IncidentMeasurementRequested, catalog.IncidentMeasurementRecorded:
		d, _ := decodeAs[struct {
			NCIDs []string `json:"nc_ids"`
		}](r)
		return sortedUnique(d.NCIDs)
	}
	return nil
}

// StepNC — шаг проекции analysis.nc по ключу key.
func StepNC(key string, v NCRecord, r kernel.Record) NCRecord {
	if key == ListKey {
		return v
	}
	if v.NCID == "" {
		v.NCID = key
	}
	switch r.Type {
	case catalog.DecisionNonconformityConfirmed:
		d, _ := decodeAs[ncConfirmedData](r)
		v.ItemID, v.DefectType, v.Severity, v.EventID, v.At, v.Seq = r.ItemID, deref(d.DefectTypeCode), d.Severity, r.EventID, r.OccurredAt, r.Seq
	case catalog.IncidentHypothesisComputed:
		// Каждая запись — новая версия слота вывода разбора (AD-3).
		v.Versions++
		v.Computed = append(slices.Clone(v.Computed), computedOf(r))
		if v.ItemID == "" {
			v.ItemID = r.ItemID
		}
	case catalog.IncidentHypothesisRecorded:
		h := humanHypothesisOf(r)
		v.Hypotheses = append(slices.Clone(v.Hypotheses), h)
		v.IncidentIDs = appendUnique(slices.Clone(v.IncidentIDs), h.IncidentID)
	case catalog.IncidentCauseConcluded:
		c := causeOf(r)
		v.Cause = &c
		v.IncidentIDs = appendUnique(slices.Clone(v.IncidentIDs), c.IncidentID)
	case catalog.IncidentMeasurementRequested:
		v.Measurements = append(slices.Clone(v.Measurements), measurementOf(r))
		v.IncidentIDs = appendUnique(slices.Clone(v.IncidentIDs), incidentOf(r))
	case catalog.IncidentMeasurementRecorded:
		v.Measurements = withResult(v.Measurements, r)
	}
	return v
}

// StepList — шаг ключа ListKey: id добавляется один раз в порядке появления.
func StepList(v ListRecord, id string) ListRecord {
	if id == "" || slices.Contains(v.IDs, id) {
		return v
	}
	v.IDs = append(slices.Clone(v.IDs), id)
	return v
}

// ListIDOf — id, который запись добавляет в список проекции (incident_id или nc_id).
func ListIDOf(r kernel.Record) string {
	switch r.Type {
	case catalog.IncidentIncidentOpened:
		return incidentOf(r)
	case catalog.DecisionNonconformityConfirmed:
		d, _ := decodeAs[ncConfirmedData](r)
		return d.NcID
	}
	return ""
}

// LocalID — локальная часть внутреннего id изделия (`ENT01:F-017` → `F-017`).
func LocalID(itemID string) string {
	if _, l, ok := strings.Cut(itemID, ":"); ok {
		return l
	}
	return itemID
}

func reasonOf(r ev.Reason) ReasonRecord {
	x := ReasonRecord{Text: string(r.Text)}
	if r.Code != nil {
		x.Code = string(*r.Code)
	}
	return x
}

func uuidStrings(xs []ev.UUID) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, string(x))
	}
	return out
}

func itemStrings(xs []ev.ItemID) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, string(x))
	}
	return out
}

func cloneMap[V any](m map[string]V) map[string]V {
	out := make(map[string]V, len(m)+1)
	for _, k := range slices.Sorted(maps.Keys(m)) {
		out[k] = m[k]
	}
	return out
}

// computedOf — уверенность гипотез версии вывода по категориям.
func computedOf(r kernel.Record) ComputedVersion {
	d, _ := decodeAs[struct {
		Hypotheses []struct {
			Category     string `json:"category"`
			ConfidenceBP *int   `json:"confidence_bp"`
		} `json:"hypotheses"`
	}](r)
	out := ComputedVersion{At: r.OccurredAt, EventID: r.EventID, Confidence: map[string]int{}}
	for _, h := range d.Hypotheses {
		if h.ConfidenceBP != nil {
			out.Confidence[h.Category] = *h.ConfidenceBP
		}
	}
	return out
}
