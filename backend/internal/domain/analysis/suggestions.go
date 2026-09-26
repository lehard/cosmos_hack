package analysis

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Предложения (FR-63, эпик 42): генераторы — подключаемые адаптеры за портом
// application/analysis.Generator; здесь — чистые правила встроенных
// генераторов («где возможно — правила и обычная логика») и проекция
// analysis.suggestion. Предложение — факт incident.suggestion.recorded с
// основаниями и ответственным; дальше — только решения людей: передать
// ответственному (incident.suggestion.forwarded) и принять или отклонить
// (incident.suggestion.resolved). Ничего не применяется автоматически:
// изменение нормы — новая версия процесса через кворум, мера — FR-64.

// Виды предложений — перечисление контракта incident.suggestion.recorded.
const (
	SuggestBottleneck   = "bottleneck"
	SuggestRiskScope    = "risk_scope"
	SuggestReactionRule = "reaction_rule_candidate"
	SuggestAnalyzer     = "analyzer_adaptation"
	SuggestDataDeficit  = "data_deficit"
)

// Статусы предложения (вычисляются из записей журнала).
const (
	SuggestionNew       = "new"
	SuggestionForwarded = "forwarded"
	SuggestionAccepted  = "accepted"
	SuggestionRejected  = "rejected"
)

// Роли ответственных по умолчанию (id ролей access, PRD §3a).
const (
	RoleSiteForeman       = "site_foreman"
	RoleTechnologist      = "technologist"
	RoleHeadOfQC          = "head_of_qc"
	RoleProductionManager = "production_manager"
)

// Proposal — предложение генератора до записи (FR-63): что предлагается,
// почему (основания — event_id записей), кому и какой ожидается эффект.
type Proposal struct {
	Generator       string   `json:"generator"`
	Kind            string   `json:"kind"`
	Title           string   `json:"title"`
	Statement       string   `json:"statement"`
	Estimate        string   `json:"estimate,omitempty"`
	ResponsibleRole string   `json:"responsible_role,omitempty"`
	StepKey         string   `json:"step_key,omitempty"`
	IncidentID      string   `json:"incident_id,omitempty"`
	MissingKind     string   `json:"missing_kind,omitempty"`
	DedupKey        string   `json:"dedup_key"`
	Basis           []string `json:"basis"`
}

// SuggestionID — стабильный id предложения по генератору и ключу повторения
// (`SUG-‹12 hex›`): повторный прогон генератора над тем же состоянием даёт
// тот же id, и второе предложение не записывается.
func SuggestionID(generator, dedupKey string) string {
	u := strings.ReplaceAll(kernel.UUIDv5(constants.NsAnt, "suggestion|"+generator+"|"+dedupKey), "-", "")
	return "SUG-" + strings.ToUpper(u[len(u)-12:])
}

// SuggestionEventID — event_id факта предложения (UUIDv5): повтор записи —
// дубль по ключу приёма (AD-7), а не второй факт.
func SuggestionEventID(generator, dedupKey string) string {
	return kernel.UUIDv5(constants.NsAnt, "suggestion.recorded|"+generator+"|"+dedupKey)
}

// SuggestionStream — поток предложения `suggestion:‹id›` (каталог, вид suggestion).
func SuggestionStream(id string) string { return "suggestion:" + id }

// SuggestionStep — строка истории предложения.
type SuggestionStep struct {
	Type          string    `json:"type"`
	Actor         string    `json:"actor,omitempty"`
	At            time.Time `json:"at"`
	ResponsibleID string    `json:"responsible_id,omitempty"`
	Role          string    `json:"responsible_role,omitempty"`
	Text          string    `json:"text,omitempty"`
	EventID       string    `json:"event_id"`
}

// SuggestionRecord — значение проекции analysis.suggestion (ключ — suggestion_id).
type SuggestionRecord struct {
	Proposal
	SuggestionID  string           `json:"suggestion_id"`
	ResponsibleID string           `json:"responsible_id,omitempty"`
	Status        string           `json:"status"`
	RecordedAt    time.Time        `json:"recorded_at"`
	EventID       string           `json:"event_id"`
	History       []SuggestionStep `json:"history"`
	// BasisSeq — seq последней записи потока предложения: basis_seq команд (AD-39).
	BasisSeq int64 `json:"basis_seq"`
}

// Open — предложение ещё ждёт решения (новое или передано).
func (v SuggestionRecord) Open() bool {
	return v.Status == SuggestionNew || v.Status == SuggestionForwarded
}

// SuggestionKeys — ключи проекции analysis.suggestion, которые меняет запись.
func SuggestionKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.IncidentSuggestionRecorded:
		if id := suggestionOf(r); id != "" {
			return []string{id, ListKey}
		}
	case catalog.IncidentSuggestionForwarded, catalog.IncidentSuggestionResolved:
		if id := suggestionOf(r); id != "" {
			return []string{id}
		}
	}
	return nil
}

// SuggestionListID — id, который запись добавляет в список предложений.
func SuggestionListID(r kernel.Record) string {
	if r.Type == catalog.IncidentSuggestionRecorded {
		return suggestionOf(r)
	}
	return ""
}

func suggestionOf(r kernel.Record) string {
	d, _ := decodeAs[struct {
		SuggestionID string `json:"suggestion_id"`
	}](r)
	return d.SuggestionID
}

// StepSuggestion — шаг проекции предложения по записи.
func StepSuggestion(v SuggestionRecord, r kernel.Record) SuggestionRecord {
	at := r.OccurredAt
	switch r.Type {
	case catalog.IncidentSuggestionRecorded:
		d, err := kernel.Decode[ev.IncidentSuggestionRecordedV1](r)
		if err != nil {
			return v
		}
		v.Proposal = Proposal{Generator: d.Generator, Kind: string(d.Kind), Statement: d.Statement, Title: deref(d.Title),
			Estimate: deref(d.Estimate), ResponsibleRole: deref(d.ResponsibleRole), MissingKind: deref(d.MissingKind),
			DedupKey: deref(d.DedupKey), Basis: uuidStrings(d.Basis)}
		if d.StepKey != nil {
			v.StepKey = string(*d.StepKey)
		}
		if d.IncidentID != nil {
			v.IncidentID = string(*d.IncidentID)
		}
		if d.ResponsibleID != nil {
			v.ResponsibleID = string(*d.ResponsibleID)
		}
		v.SuggestionID, v.Status, v.RecordedAt, v.EventID = string(d.SuggestionID), SuggestionNew, at, r.EventID
		v.History = append(slices.Clone(v.History), SuggestionStep{Type: "recorded", Actor: d.Generator, At: at, EventID: r.EventID})
	case catalog.IncidentSuggestionForwarded:
		d, err := kernel.Decode[ev.IncidentSuggestionForwardedV1](r)
		if err != nil {
			return v
		}
		v.ResponsibleID = string(d.ResponsibleID)
		if d.ResponsibleRole != nil && *d.ResponsibleRole != "" {
			v.ResponsibleRole = *d.ResponsibleRole
		}
		if v.Status == SuggestionNew || v.Status == "" {
			v.Status = SuggestionForwarded
		}
		v.History = append(slices.Clone(v.History), SuggestionStep{Type: "forwarded", Actor: ActorName(r.Actor), At: at,
			ResponsibleID: string(d.ResponsibleID), Role: deref(d.ResponsibleRole), Text: deref(d.Note), EventID: r.EventID})
	case catalog.IncidentSuggestionResolved:
		d, err := kernel.Decode[ev.IncidentSuggestionResolvedV1](r)
		if err != nil {
			return v
		}
		v.Status = SuggestionRejected
		if d.Resolution == ev.IncidentSuggestionResolvedV1ResolutionAccepted {
			v.Status = SuggestionAccepted
		}
		v.History = append(slices.Clone(v.History), SuggestionStep{Type: v.Status, Actor: ActorName(r.Actor), At: at,
			Text: string(d.Reason.Text), EventID: r.EventID})
	default:
		return v
	}
	if v.History == nil {
		v.History = []SuggestionStep{}
	}
	v.BasisSeq = r.Seq
	return v
}

// GuardSuggestionOpen — решение по предложению: только пока оно открыто
// (новое или передано); принятое или отклонённое не пересматривается —
// новое предложение генератор сформирует по новому основанию.
func GuardSuggestionOpen(v SuggestionRecord, why string) error {
	if v.Open() {
		return nil
	}
	return kernel.Refuse(errcodes.IncidentSuggestionState, "suggestion_id", v.SuggestionID, "status", statusText(v.Status), "why", why)
}

func statusText(s string) string {
	switch s {
	case SuggestionAccepted:
		return "принято"
	case SuggestionRejected:
		return "отклонено"
	case SuggestionForwarded:
		return "передано"
	}
	return "новое"
}

// ── встроенные генераторы (правила) ──

// LineBottleneck — вход генератора «ограничение линии»: узел-ограничение из
// счётчиков узлов (считает analytics, FR-5) — очередь, ожидание, выход за период.
type LineBottleneck struct {
	StepKey string `json:"step_key"`
	// WaitSec — среднее ожидание детали в очереди узла, с.
	WaitSec int64 `json:"wait_sec"`
	// Queue — деталей в очереди сейчас; Passed — прошло узел за период.
	Queue  int `json:"queue"`
	Passed int `json:"passed"`
	// Period — период счётчиков словами («смена»).
	Period string `json:"period,omitempty"`
}

// BottleneckProposal — генератор «ограничение линии» (UJ-1): узел — текущее
// ограничение, оценка потерь в деталях и минутах ожидания; ответственный —
// мастер участка (персонал), руководитель может передать технологу, если
// нужна новая версия процесса.
func BottleneckProposal(b LineBottleneck) (Proposal, bool) {
	if b.StepKey == "" || (b.Queue == 0 && b.WaitSec == 0) {
		return Proposal{}, false
	}
	waitMin := (b.WaitSec + 30) / 60
	period := b.Period
	if period == "" {
		period = "смену"
	}
	est := "Детали ждут в среднем " + strconv.FormatInt(waitMin, 10) + " мин; в очереди сейчас " + strconv.Itoa(b.Queue) + " дет."
	if b.Passed > 0 && waitMin > 0 {
		est += "; за " + period + " узел прошли " + strconv.Itoa(b.Passed) + " дет. — это около " +
			strconv.FormatInt(int64(b.Passed)*waitMin, 10) + " мин ожидания"
	}
	return Proposal{
		Generator: "rules.bottleneck", Kind: SuggestBottleneck, StepKey: b.StepKey, ResponsibleRole: RoleSiteForeman,
		Title: "Ограничение линии: узел " + b.StepKey,
		Statement: "Узел " + b.StepKey + " — текущее ограничение линии: наибольшее ожидание при наибольшей загрузке. " +
			"Проверить обеспеченность поста людьми по графику смен (мастер участка); если причина в маршруте — новая версия процесса (технолог).",
		Estimate: est, DedupKey: "bottleneck|" + b.StepKey + "|" + strconv.FormatInt(waitMin/10, 10), Basis: []string{},
	}, true
}

// RiskScopeProposals — генератор «область риска и её сужение» (FR-61):
// открытый инцидент с областью от трёх изделий без единого сужения —
// предложение сузить по окну сбоя с основаниями; изделия, ушедшие дальше или
// отгруженные, — известить получателей.
func RiskScopeProposals(incidents []IncidentRecord) []Proposal {
	var out []Proposal
	for _, v := range incidents {
		if v.Closed || len(v.Versions) == 0 {
			continue
		}
		last := v.Versions[len(v.Versions)-1]
		narrowed := false
		for _, x := range v.Versions {
			narrowed = narrowed || x.Change == ChangeNarrowed
		}
		size := v.Size()
		basis := append(slices.Clone(v.TriggerEventIDs), last.EventID)
		factor := v.FactorValue
		if factor == "" {
			factor = v.Factor
		}
		if !narrowed && size >= 3 {
			st := "Область риска инцидента " + v.IncidentID + " (общий фактор " + factor + "): " + strconv.Itoa(size) +
				" изделий, сужений по основаниям не было."
			if v.WindowStart != nil && v.WindowEnd != nil {
				st += " Окно сбоя — " + v.WindowStart.UTC().Format("02.01 15:04") + "…" + v.WindowEnd.UTC().Format("02.01 15:04") + " UTC."
			}
			st += " Предлагается проверить изделия у границ окна и исключить подтверждённо годные с доказательствами — область сузится без риска пропуска."
			out = append(out, Proposal{Generator: "rules.risk_scope", Kind: SuggestRiskScope, IncidentID: v.IncidentID, StepKey: v.StepKey,
				ResponsibleRole: RoleTechnologist, Title: "Сузить область риска " + v.IncidentID + " (" + strconv.Itoa(size) + " изд.)",
				Statement: st, Estimate: "Под подозрением " + strconv.Itoa(size) + " изд.; каждое исключение — с основанием (FR-61)",
				DedupKey: "risk_scope|narrow|" + v.IncidentID + "|v" + strconv.Itoa(last.Version), Basis: sortedUnique(basis)})
		}
		if gone := last.Breakdown.MovedOn + last.Breakdown.Shipped; gone > 0 {
			out = append(out, Proposal{Generator: "rules.risk_scope", Kind: SuggestRiskScope, IncidentID: v.IncidentID, StepKey: v.StepKey,
				ResponsibleRole: RoleProductionManager, Title: "Изделия области " + v.IncidentID + " ушли дальше: " + strconv.Itoa(gone),
				Statement: "Из области риска инцидента " + v.IncidentID + " ушли дальше " + strconv.Itoa(last.Breakdown.MovedOn) +
					" и отгружены " + strconv.Itoa(last.Breakdown.Shipped) + " изд. Предлагается известить получателей и держать изделия под наблюдением до решения по области.",
				DedupKey: "risk_scope|downstream|" + v.IncidentID + "|v" + strconv.Itoa(last.Version), Basis: sortedUnique(basis)})
		}
	}
	return out
}

// ReactionRuleCandidates — генератор «кандидаты в правила реакции из
// повторяющихся ручных решений»: одно и то же решение людей (тип и код или
// текст основания) в трёх и более случаях или одна и та же подтверждённая
// причина по одному виду фактора в двух инцидентах — кандидат в правило
// режима 2 «предлагать» (FR-50); правило — часть нормативного слоя, вводится
// новой версией через кворум.
func ReactionRuleCandidates(incidents []IncidentRecord) []Proposal {
	type agg struct {
		label     string
		ids       []string
		incidents []string
	}
	decisions := map[string]*agg{}
	causes := map[string]*agg{}
	for _, v := range incidents {
		for _, id := range slices.Sorted(maps.Keys(v.Decisions)) {
			d := v.Decisions[id]
			reason := d.Reason.Code
			if reason == "" {
				reason = strings.ToLower(strings.TrimSpace(d.Reason.Text))
			}
			if reason == "" {
				continue
			}
			k := d.Type + "|" + reason
			a := decisions[k]
			if a == nil {
				a = &agg{label: decisionTypeText(d.Type) + " с основанием «" + firstNonEmpty(d.Reason.Text, d.Reason.Code) + "»"}
				decisions[k] = a
			}
			a.ids = append(a.ids, id)
			a.incidents = appendUnique(a.incidents, v.IncidentID)
		}
		if v.Cause != nil && v.Cause.Category != "" && v.Factor != "" {
			k := v.Factor + "|" + v.Cause.Category
			a := causes[k]
			if a == nil {
				a = &agg{label: "причина «" + categoryText(v.Cause.Category) + "» по фактору «" + v.Factor + "»"}
				causes[k] = a
			}
			a.ids = append(a.ids, v.Cause.EventID)
			a.incidents = appendUnique(a.incidents, v.IncidentID)
		}
	}
	var out []Proposal
	for _, k := range slices.Sorted(maps.Keys(decisions)) {
		a := decisions[k]
		if len(a.ids) < 3 {
			continue
		}
		out = append(out, Proposal{Generator: "rules.reaction_rules", Kind: SuggestReactionRule, ResponsibleRole: RoleTechnologist,
			Title: "Кандидат в правило реакции: " + a.label,
			Statement: "Люди " + strconv.Itoa(len(a.ids)) + " раз приняли одно и то же решение — " + a.label + " (инциденты: " +
				strings.Join(a.incidents, ", ") + "). Предлагается оформить правило реакции в режиме 2 «предлагать»: система будет предлагать это решение, человек — подтверждать.",
			Estimate: strconv.Itoa(len(a.ids)) + " ручных решений", DedupKey: "reaction_rule|decision|" + k + "|" + strconv.Itoa(len(a.ids)/3),
			Basis: sortedUnique(a.ids)})
	}
	for _, k := range slices.Sorted(maps.Keys(causes)) {
		a := causes[k]
		if len(a.incidents) < 2 {
			continue
		}
		out = append(out, Proposal{Generator: "rules.reaction_rules", Kind: SuggestReactionRule, ResponsibleRole: RoleTechnologist,
			Title: "Кандидат в правило реакции: повторяющаяся " + a.label,
			Statement: "В " + strconv.Itoa(len(a.incidents)) + " инцидентах (" + strings.Join(a.incidents, ", ") + ") подтверждена " + a.label +
				". Предлагается правило: при сигнале по этому фактору сразу назначать дополнительную проверку (режим 2 «предлагать»).",
			Estimate: strconv.Itoa(len(a.incidents)) + " инцидента с одной причиной", DedupKey: "reaction_rule|cause|" + k + "|" + strconv.Itoa(len(a.incidents)),
			Basis: sortedUnique(a.ids)})
	}
	return out
}

// AnalyzerAdaptationProposals — генератор «сигналы для адаптации VisionQC»:
// в окне возможного возникновения камера сообщила «признаков нет» или
// «оценка невозможна», а дефект потом нашли другим методом — пропуск камеры.
// Случаи группируются по узлу и виду дефекта; предложение — передать их в
// размеченные примеры и пересмотреть рецепт (допуск новой версии — паспорт
// анализатора, эпик 40; сам генератор ничего не меняет).
func AnalyzerAdaptationProposals(analyses []Analysis) []Proposal {
	type agg struct {
		step, defect string
		ncs, ids     []string
	}
	groups := map[string]*agg{}
	for _, a := range analyses {
		if a.Incoming {
			continue
		}
		var from time.Time
		if a.Window != nil {
			from = a.Window.Start
		}
		var missed []string
		var found *Mark
		for i := range a.Records {
			m := a.Records[i]
			if m.Lane != LaneItem || m.OccurredAt.Before(from) {
				continue
			}
			camera := m.Params["method"] == "camera"
			switch {
			case camera && (m.Variant == "no_defect_indicated" || m.Variant == "unable_to_assess") && found == nil:
				missed = append(missed, m.EventID)
			case !camera && m.Variant == "defect_indicated" && len(missed) > 0 && found == nil:
				found = &a.Records[i]
			}
		}
		if found == nil || len(missed) == 0 {
			continue
		}
		step := firstNonEmpty(a.Profile.StepKey, "—")
		k := step + "|" + a.Profile.DefectType
		g := groups[k]
		if g == nil {
			g = &agg{step: step, defect: firstNonEmpty(a.Profile.DefectType, "не указан")}
			groups[k] = g
		}
		g.ncs = appendUnique(g.ncs, a.NCID)
		g.ids = append(g.ids, append(missed, found.EventID)...)
	}
	var out []Proposal
	for _, k := range slices.Sorted(maps.Keys(groups)) {
		g := groups[k]
		out = append(out, Proposal{Generator: "rules.vision_adaptation", Kind: SuggestAnalyzer, StepKey: g.step, ResponsibleRole: RoleHeadOfQC,
			Title: "Адаптация VisionQC: пропуски «" + g.defect + "» на " + g.step,
			Statement: "Камера не показала признаков дефекта «" + g.defect + "» на узле " + g.step + ", а дефект затем нашли другим методом: " +
				strconv.Itoa(len(g.ncs)) + " случ. (" + strings.Join(g.ncs, ", ") + "). Предлагается передать случаи в размеченные примеры и пересмотреть рецепт; " +
				"новая версия анализатора — только через паспорт допуска.",
			Estimate: strconv.Itoa(len(g.ncs)) + " пропуск(ов) камеры", DedupKey: "vision|" + k + "|" + strconv.Itoa(len(g.ncs)), Basis: sortedUnique(g.ids)})
	}
	return out
}

func decisionTypeText(t string) string {
	switch t {
	case "narrowed", "scope.narrowed", string(catalog.IncidentScopeNarrowed):
		return "сужение области"
	case "expanded", string(catalog.IncidentScopeExpanded):
		return "расширение области"
	case "assessed", string(catalog.IncidentItemAssessed):
		return "оценка изделия"
	}
	return t
}

func categoryText(c string) string {
	switch c {
	case CatIncoming:
		return "входной брак"
	case CatEquipment:
		return "оборудование"
	case CatPerformer:
		return "исполнитель"
	case CatHandling:
		return "перемещение и хранение"
	case CatAssembly:
		return "сборка"
	case CatDocumentation:
		return "документация"
	case CatNotEstablished:
		return "не установлена"
	}
	return c
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
