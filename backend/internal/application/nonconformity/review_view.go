package nonconformity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
	dp "ant/internal/domain/process"
)

// Экран решения на точке предъявления и пересмотра (Д-81, стол контролёра):
// сервер сам говорит, что было в основании решения (включая «данных не
// было»), почему новые факты значимы, что рекомендует система (отдельно от
// политики) и какие решения доступны вошедшему с последствиями, вычисленными
// из состояния изделия. Интерфейс показывает только это и ничего не выдумывает.

// resolutionLabel — решение на точке словами.
func resolutionLabel(r string) string {
	switch r {
	case "accept":
		return "годно"
	case "accept_with_concession":
		return "годно по разрешению на отклонение"
	case "reject":
		return "не принято"
	case "insufficient_data":
		return "недостаточно данных"
	}
	return r
}

// gateName — закрывающая точка для людей: имя узла процесса, иначе код.
func gateName(v *itemView, cp, stepKey string) string {
	if def := v.Env.Process.Def; def != nil {
		if l := closingPointLabel(def, cp); l != nil {
			return *l
		}
	}
	if cp != "" {
		return cp
	}
	if n := stepName(v, stepKey); n != "" {
		return n
	}
	return stepKey
}

// equipmentOf — оборудование записи режима (equipment_id в data).
func equipmentOf(r kernel.Record) string {
	var d struct {
		EquipmentID string `json:"equipment_id"`
	}
	_ = json.Unmarshal(r.Data, &d)
	return d.EquipmentID
}

// equipmentName — название оборудования по справочнику на момент at; нет — код.
func (s *Service) equipmentName(ctx context.Context, id string, at time.Time) string {
	if s.d.Equipment != nil {
		if n, ok := s.d.Equipment.EquipmentName(ctx, id, at); ok && n != "" {
			return n
		}
	}
	return id
}

// knownSeq — до какой записи журнала решение знало вход: basis_seq решения,
// без него — seq самого решения (как правило движка AD-5).
func knownSeq(dec kernel.Record) int64 {
	if dec.BasisSeq > 0 {
		return dec.BasisSeq
	}
	return dec.Seq
}

// absentSources — оборудование новых фактов, от которого на момент решения
// в изделии не было ни одной записи: «данных не было». Порядок — по первому
// пришедшему факту.
func absentSources(v *itemView, dec kernel.Record, facts []kernel.Record) map[string]kernel.Record {
	out := map[string]kernel.Record{}
	known := knownSeq(dec)
	for _, f := range facts {
		eq := equipmentOf(f)
		if eq == "" || !strings.HasPrefix(string(f.Type), "equipment.") {
			continue
		}
		if _, seen := out[eq]; seen {
			continue
		}
		had := slices.ContainsFunc(v.Input, func(r kernel.Record) bool {
			return r.Seq <= known && r.Kind == catalog.KindFact && strings.HasPrefix(string(r.Type), "equipment.") && equipmentOf(r) == eq
		})
		if !had {
			out[eq] = f
		}
	}
	return out
}

// runOf — выполнение операции, к которому относится запись режима: то же
// оборудование и время внутри выполнения (без конца — после начала). Нет
// выполнения в состоянии процесса — по записям начала выполнения во входе.
func runOf(v *itemView, f kernel.Record) *dp.Run {
	eq := equipmentOf(f)
	if eq == "" {
		return nil
	}
	var best *dp.Run
	for _, id := range sortedKeys(v.Snap.Process.Runs) {
		r := v.Snap.Process.Runs[id]
		if r.Equipment != eq || f.OccurredAt.Before(r.StartedAt) {
			continue
		}
		if r.FinishedAt != nil && f.OccurredAt.After(*r.FinishedAt) {
			continue
		}
		x := r
		best = &x
	}
	if best != nil {
		return best
	}
	for _, r := range v.Input {
		if r.Type != catalog.OperationRunStarted || r.OccurredAt.After(f.OccurredAt) || equipmentOf(r) != eq {
			continue
		}
		var d runData
		_ = json.Unmarshal(r.Data, &d)
		x := dp.Run{RunID: d.OperationRunID, StepKey: d.StepKey, Equipment: eq, StartedAt: r.OccurredAt}
		if def := v.Env.Process.Def; def != nil {
			if n := def.ByStep(d.StepKey); n != nil {
				x.Special = n.Special()
			}
		}
		best = &x
	}
	return best
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// Span — длительность словами: «2 ч 10 мин», «35 мин».
func Span(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	h, m := int(d.Hours()), int(d.Minutes())%60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%d ч %d мин", h, m)
	case h > 0:
		return fmt.Sprintf("%d ч", h)
	}
	return fmt.Sprintf("%d мин", max(m, 1))
}

// number — целое с масштабом словами (AD-4: без float).
func number(v int64, scale int) string {
	s := strconv.FormatInt(v, 10)
	if scale <= 0 {
		return s
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for len(s) <= scale {
		s = "0" + s
	}
	s = s[:len(s)-scale] + "," + s[len(s)-scale:]
	if neg {
		s = "-" + s
	}
	return s
}

// OutOfSetpoint — наблюдённое значение вне уставки словами; в уставке или нет чисел — "".
func OutOfSetpoint(rd *NCParameterReading) string {
	if rd == nil || rd.SetpointMin == nil || rd.SetpointMax == nil {
		return ""
	}
	var bad []string
	if rd.ObservedMin != nil && *rd.ObservedMin < *rd.SetpointMin {
		bad = append(bad, number(*rd.ObservedMin, rd.Scale))
	}
	if rd.ObservedMax != nil && *rd.ObservedMax > *rd.SetpointMax && (rd.ObservedMin == nil || *rd.ObservedMax != *rd.ObservedMin || len(bad) == 0) {
		bad = append(bad, number(*rd.ObservedMax, rd.Scale))
	}
	if len(bad) == 0 {
		return ""
	}
	return fmt.Sprintf("%s: %s %s вне уставки %s…%s %s", parameterName(rd.Parameter), strings.Join(bad, " и "), rd.Unit,
		number(*rd.SetpointMin, rd.Scale), number(*rd.SetpointMax, rd.Scale), rd.Unit)
}

// parameterName — параметр режима словами (известные — по-русски).
func parameterName(p string) string {
	switch p {
	case "current_a":
		return "Ток сварки"
	case "voltage_v":
		return "Напряжение дуги"
	case "wire_feed_mm_s", "wire_feed":
		return "Подача проволоки"
	case "gas_flow_l_min", "gas_flow":
		return "Расход защитного газа"
	}
	return "Параметр " + p
}

// fillReview — пересмотр (Д-81): прежнее решение с основанием, что было
// известно при подписи (включая «данных не было»), и почему новые факты
// значимы. Возвращает, значимы ли новые факты для приёмки (для рекомендации).
func (s *Service) fillReview(ctx context.Context, v *itemView, rv review, d dom.PresentationResolvedData, out *NCPresentationReview) []string {
	absent := absentSources(v, rv.decision, rv.facts)
	gate := gateName(v, d.ClosingPoint, d.StepKey)

	sum := gate + ": " + resolutionLabel(d.Resolution)
	if len(absent) > 0 && (d.Resolution == "accept" || d.Resolution == "accept_with_concession") {
		sum += " при неполных данных"
	}
	if d.Reason != nil && strings.TrimSpace(d.Reason.Text) != "" {
		sum += " — " + strings.TrimSpace(d.Reason.Text)
	}
	out.Decision.Summary = sum
	out.Decision.Params = map[string]string{"resolution": d.Resolution, "closing_point": d.ClosingPoint}

	for _, eq := range sortedKeys(absent) {
		f := absent[eq]
		x := ref(f, "Нет данных: журнала режима «"+s.equipmentName(ctx, eq, f.OccurredAt)+"» на момент решения не было")
		x.Absent, x.Reading = true, nil
		out.KnownAtDecision = append(out.KnownAtDecision, x)
	}

	var why, significant []string
	add := func(dst *[]string, line string) {
		if line != "" && !slices.Contains(*dst, line) {
			*dst = append(*dst, line)
		}
	}
	for _, f := range rv.facts {
		title := summaryOf(f)
		when := "возникло за " + Span(rv.decision.OccurredAt.Sub(f.OccurredAt)) + " до решения"
		if !f.RecordedAt.IsZero() && f.RecordedAt.After(rv.decision.OccurredAt) {
			when += ", стало известно через " + Span(f.RecordedAt.Sub(rv.decision.OccurredAt)) + " после него"
		}
		add(&why, title+": "+when+" — решение принималось без этой записи")
		if run := runOf(v, f); run != nil {
			op := stepName(v, run.StepKey)
			if op == "" {
				op = run.StepKey
			}
			add(&why, "Относится к операции «"+op+"» (выполнение "+run.RunID+") до приёмки на точке «"+gate+"»")
			if run.Special {
				line := "«" + op + "» — специальный процесс: нарушение режима само по себе — несоответствие, даже если контроль дефекта не нашёл (FR-151)"
				add(&why, line)
				if f.Type == catalog.EquipmentDeviationDetected {
					add(&significant, line)
				}
			}
		}
		if line := OutOfSetpoint(readingOf(f)); line != "" {
			add(&why, line)
			add(&significant, line)
		} else if f.Type == catalog.EquipmentDeviationDetected {
			add(&significant, title+": отклонение режима на операции до приёмки")
		}
	}
	for _, eq := range sortedKeys(absent) {
		add(&why, "При подписи данных «"+s.equipmentName(ctx, eq, absent[eq].OccurredAt)+"» не было: приёмка стояла только на результатах методов контроля")
	}
	out.WhySignificant = why
	return significant
}

// reviewRecommendation — рекомендация по пересмотру (отдельно от политики):
// значимые новые факты или блок изделия правилом — отозвать приёмку; иначе —
// оставить в силе.
func reviewRecommendation(st dom.State, significant []string) *NCRecommendation {
	why := slices.Clone(significant)
	for _, c := range st.Containment {
		if c.By == dom.ByRule && !c.Released && c.Level != "none" {
			why = append(why, "Изделие под сдерживанием правилом: "+first(c.Reason, c.Rule))
		}
	}
	if len(why) > 0 {
		return &NCRecommendation{Outcome: dom.ReviewRevoked, Why: why}
	}
	return &NCRecommendation{Outcome: dom.ReviewUpheld, Why: []string{"Новые данные в пределах уставок и не связаны с нарушением требований — основание приёмки держится"}}
}

func first(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// refusalText — отказ гарда словами для «почему недоступно».
func refusalText(err error) string {
	var r *kernel.Refusal
	if errors.As(err, &r) {
		if r.Detail != "" {
			return r.Detail
		}
		if info, ok := errcodes.Lookup(r.Code); ok {
			t := info.Detail
			for k, val := range r.Params {
				t = strings.ReplaceAll(t, "{"+k+"}", val)
			}
			return t
		}
		return string(r.Code)
	}
	if e, ok := platform.AsError(err); ok {
		return first(e.Detail, string(e.Code))
	}
	return err.Error()
}

// where — где изделие сейчас (главный токен процесса) словами.
func where(v *itemView) string {
	if tv, ok := v.Snap.Process.Primary(v.Env.Process); ok && tv.StepKey != "" {
		if n := stepName(v, tv.StepKey); n != "" {
			return "«" + n + "»"
		}
		return tv.StepKey
	}
	return "текущем шаге"
}

// incidentsOf — области риска (инциденты), в которых изделие сдержано правилом.
func incidentsOf(st dom.State) []string {
	var out []string
	for _, c := range st.Containment {
		if id, ok := strings.CutPrefix(c.Key, "incident:"); ok && !c.Released {
			out = append(out, id)
		}
	}
	return out
}

// reviewActions — «оставить в силе» и «отозвать приёмку» с доступностью для
// вошедшего (GuardReview и полномочие точки) и последствиями (Д-81).
func (s *Service) reviewActions(ctx context.Context, v *itemView, rv review, d dom.PresentationResolvedData) []NCPresentationAction {
	actor := platform.PrincipalFrom(ctx).PersonID
	st := v.State()
	gate := gateName(v, d.ClosingPoint, d.StepKey)
	authErr := s.gateAuthority(v, actor, d.StepKey)
	var policy *string
	if g, ok := v.Snap.Process.Gate(d.StepKey); ok && g.Authority != "" {
		ref := "полномочие точки «" + g.Authority + "»; вторая подпись по политике не требуется (Д-81)"
		policy = &ref
	} else {
		ref := "политика: nonconformity.presentation.review; вторая подпись не требуется (Д-81)"
		policy = &ref
	}
	facts := make([]string, 0, len(rv.facts))
	for _, f := range rv.facts {
		facts = append(facts, f.EventID)
	}
	mk := func(outcome, label string, cons []string) NCPresentationAction {
		o := outcome
		a := NCPresentationAction{Operation: dom.ActPresentationReview, Outcome: &o, Label: label, Consequences: cons, PolicyRef: policy}
		err := authErr
		if err == nil {
			err = dom.GuardReview(st, actor, dom.PresentationReviewedData{ReviewedEventID: rv.decision.EventID, StepKey: d.StepKey,
				ClosingPoint: d.ClosingPoint, PresentationNo: d.PresentationNo, Outcome: outcome, NewFactIDs: facts, Reason: dom.Reason{Text: "проверка доступности"}})
		}
		a.Allowed = err == nil
		a.WhyAvailable = ReviewWhyAllowed(outcome, gate)
		if err != nil {
			a.WhyAvailable = refusalText(err)
		}
		return a
	}
	revoked, upheld := ReviewConsequences(gate, d.Resolution, where(v), incidentsOf(st))
	return []NCPresentationAction{
		mk(dom.ReviewRevoked, LabelRevoke, revoked),
		mk(dom.ReviewUpheld, LabelUphold, upheld),
	}
}

// Надписи исходов пересмотра (Д-81).
const (
	LabelRevoke = "Отозвать приёмку — изделие на блок до решения"
	LabelUphold = "Оставить решение в силе"
)

// ReviewWhyAllowed — почему исход пересмотра доступен (когда гард пропускает).
func ReviewWhyAllowed(outcome, gate string) string {
	if outcome == dom.ReviewRevoked {
		return "Отзыв приёмки — защитное направление: доступен уполномоченному на точке «" + gate + "», пока приёмка не отозвана"
	}
	return "Приёмка прошла бы и сейчас: изделие не заблокировано, открытых несоответствий без решения нет, вы не участвовали в изготовлении"
}

// ReviewConsequences — последствия исходов пересмотра словами (Д-81): что
// будет с изделием, маршрутом, блокировкой, областью риска, 1С и историей.
// gate — точка для людей, resolution — прежнее решение, where — где изделие
// сейчас, incidents — области риска, в которых изделие сдержано.
func ReviewConsequences(gate, resolution, where string, incidents []string) (revoked, upheld []string) {
	upheld = []string{
		"Прежнее решение «" + resolutionLabel(resolution) + "» на точке «" + gate + "» остаётся в силе; изделие остаётся на " + where,
		"Маршрут и блокировки не меняются; задача «пересмотрите» закрывается",
		"1С: без изменений",
		"История: запись пересмотра с основанием поверх прежнего решения (критическое действие, подпись уровня 2); прежняя запись не меняется",
	}
	revoked = []string{
		"Изделие блокируется (блок изделия человеком): приёмка на следующих точках и снятие блока — только уполномоченным после доп. проверки или решения по несоответствию",
		"Качество: «не проверено» — годность не подтверждена; несоответствие не создаётся, решение по изделию не меняется",
		"Маршрут: изделие назад не переводится; мастеру там, где изделие сейчас (" + where + "), — задача остановить и отложить до решения",
	}
	if len(incidents) > 0 {
		revoked = append(revoked, "Область риска: изделие уже в области инцидента "+strings.Join(incidents, ", ")+" — решение по области принимается отдельно, отзыв его не снимает")
	} else {
		revoked = append(revoked, "Область риска: не меняется — её расширяет правило по новым фактам, а не отзыв")
	}
	revoked = append(revoked,
		"1С: результат контроля точки «"+gate+"» исправляется на «недостаточно данных»; если прежний уже принят 1С — сторно и новое по решению ответственного за обмен (AD-7)",
		"История: прежнее решение остаётся в журнале, отзыв — новой записью с основанием (критическое действие, подпись уровня 2)")
	return revoked, upheld
}

// resolveActions — решения на ждущем предъявлении с доступностью (те же
// гарды, что у команды) и последствиями.
func (s *Service) resolveActions(ctx context.Context, v *itemView, p NCPresentationPoint) []NCPresentationAction {
	actor := platform.PrincipalFrom(ctx).PersonID
	now := s.at(ctx, platform.Moment{}, v.RunID)
	gate := gateName(v, p.ClosingPoint, p.StepKey)
	concession, concessionID := "", ""
	if book, err := s.concessionBook(ctx, platform.Moment{}); err == nil {
		for _, c := range book.Sorted() {
			if c.InScope(v.ItemID) && dom.ConcessionGuard(&c, c.GrantedEventID, v.ItemID, "", now) == nil {
				concession, concessionID = first(c.Grant.Number, c.Grant.ConcessionID), c.Grant.ConcessionID
			}
		}
	}
	var policy *string
	if g, ok := v.Snap.Process.Gate(p.StepKey); ok && g.Authority != "" {
		ref := "полномочие точки «" + g.Authority + "»"
		policy = &ref
	}
	authErr := s.gateAuthority(v, actor, p.StepKey)
	next := "следующий шаг процесса"
	if p.NextStepLabel != nil {
		next = "«" + *p.NextStepLabel + "»"
	}
	stream := "item:" + v.ItemID
	var out []NCPresentationAction
	for _, r := range []string{"accept", "accept_with_concession", "reject", "insufficient_data"} {
		if r == "accept_with_concession" && concession == "" {
			continue
		}
		res := r
		a := NCPresentationAction{Operation: dom.ActPresentation, Resolution: &res, PolicyRef: policy}
		data := dom.PresentationResolvedData{StepKey: p.StepKey, ClosingPoint: p.ClosingPoint, Resolution: r, PresentationNo: p.PresentationNo,
			MethodEventIDs: p.MethodEventIDs}
		if r == "accept_with_concession" {
			data.ConcessionID = concessionID
		}
		err := authErr
		if err == nil {
			err = dom.Guard(v.State(), v.Env, v.Upstream(), kernel.Command{Action: dom.ActPresentation, Actor: actor, Object: stream, BasisSeq: v.BasisSeq,
				GuardStreams: []string{stream}, OccurredAt: now, SignatureLevel: 2, Payload: data})
		}
		a.Allowed = err == nil
		a.Label, a.WhyAvailable, a.Consequences = ResolveTexts(r, gate, next, concession, p.PresentationNo)
		if err != nil {
			a.WhyAvailable = refusalText(err)
		}
		out = append(out, a)
	}
	return out
}

// ResolveTexts — надпись, «почему доступно» (когда гард пропускает) и
// последствия решения на точке словами; next — куда передаётся изделие при
// «принять», concession — номер разрешения на отклонение.
func ResolveTexts(resolution, gate, next, concession string, presentationNo int) (label, why string, cons []string) {
	hist := "История: решение с подписью уровня 2 (критическое действие) на точке «" + gate + "», предъявление №" + strconv.Itoa(presentationNo)
	switch resolution {
	case "accept":
		return "Принять — передать на " + next, "Результаты методов есть, блока и открытых несоответствий нет, вы не участвовали в изготовлении",
			[]string{"Изделие передаётся на " + next, "Качество: «годно»", "1С: результат контроля точки «" + gate + "» — годно", hist}
	case "accept_with_concession":
		return "Принять по разрешению на отклонение " + concession, "Изделие в области действующего разрешения на отклонение " + concession,
			[]string{"Изделие передаётся на " + next, "Качество: «годно по разрешению на отклонение»", "Расходуется 1 из лимита разрешения " + concession,
				"1С: результат контроля точки «" + gate + "» — годно по разрешению", hist}
	case "reject":
		return "Не принять — вернуть", "Вернуть можно, если изделие стоит на точке и у вас есть полномочие точки",
			[]string{"Изделие уходит по ветке «не принято» процесса — на доработку или разбор", "1С: результат контроля точки «" + gate + "» — не годно", hist}
	case "insufficient_data":
		return "Недостаточно данных — ждать", "«Нет данных» ≠ «годно»: решение фиксирует нехватку сведений",
			[]string{"Изделие остаётся на точке до новых результатов контроля", "1С: результат контроля точки «" + gate + "» — недостаточно данных", hist}
	}
	return resolution, "", []string{hist}
}

// presentationRecommendation — рекомендация на ждущем предъявлении по
// результатам методов и состоянию изделия (решает человек).
func presentationRecommendation(v *itemView, methods []NCRecordRef, actions []NCPresentationAction) *NCRecommendation {
	var defect, unable []string
	for _, m := range methods {
		r, ok := v.Record(m.EventID)
		if !ok {
			continue
		}
		var d inspectionData
		_ = json.Unmarshal(r.Data, &d)
		switch d.Outcome {
		case "defect_indicated", "defect_found":
			defect = append(defect, m.Summary)
		case "unable_to_assess":
			unable = append(unable, m.Summary)
		}
	}
	st := v.State()
	switch {
	case len(defect) > 0:
		return &NCRecommendation{Outcome: "reject", Why: defect}
	case st.Blocked():
		return &NCRecommendation{Outcome: "insufficient_data", Why: []string{"Изделие заблокировано: решение о годности — после снятия блока"}}
	case len(unable) > 0 || len(methods) == 0:
		why := append([]string{}, unable...)
		if len(methods) == 0 {
			why = append(why, "Нет результатов методов контроля")
		}
		return &NCRecommendation{Outcome: "insufficient_data", Why: why}
	}
	for _, a := range actions {
		if a.Resolution != nil && *a.Resolution == "accept" && !a.Allowed {
			return &NCRecommendation{Outcome: "insufficient_data", Why: []string{a.WhyAvailable}}
		}
	}
	return &NCRecommendation{Outcome: "accept", Why: []string{"Методы контроля признаков дефекта не нашли; блока и открытых несоответствий нет"}}
}
