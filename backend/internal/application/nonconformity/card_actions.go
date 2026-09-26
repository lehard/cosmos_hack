package nonconformity

import (
	"context"
	"encoding/json"
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

// Решения карточки несоответствия с последствиями (интерфейс 7, стол
// контролёра; по образцу presentation.read, Д-81): доступность — теми же
// гардами, что у команд (доменный гард, полномочие, разрешение на
// отклонение); последствия — из состояния изделия, процесса и политики:
// сначала деловые (что с изделием и маршрутом, кому уйдёт действие, чьё ещё
// решение нужно), потом технические (статусы, 1С, история). Интерфейс
// показывает только это.

// AuthorityNCDisposition — полномочие «Решение по несоответствию» (normative/policy).
const AuthorityNCDisposition = "nc_disposition"

// Dispositions — варианты решения по изделию в порядке показа (FR-53).
var Dispositions = []string{"rework", "repair", "use_as_is", "scrap", "return_to_supplier"}

// DecisionFacts — что нужно текстам решений карточки: изделие, где оно,
// операция несоответствия, разрешение на отклонение, подписи маршрута,
// области риска. Заполняет live (из свёртки) и заготовки (из мира).
type DecisionFacts struct {
	NCNumber  string
	ItemLabel string
	// Where — где изделие сейчас («Сварка»), в кавычках.
	Where string
	// Operation — операция несоответствия («Сварка»), RunLabel — её
	// выполнение для людей («СВ-017-1»).
	Operation string
	RunLabel  string
	// Next — куда изделие идёт дальше по маршруту («Сборка»), в кавычках.
	Next string
	// Concession — номер действующего разрешения на отклонение в области изделия.
	Concession string
	// Approvals — подписи маршрута решения по изделию: route_closed | pending | demo_stub.
	Approvals  string
	Commission bool
	Isolated   bool
	// Containment — уровень сдерживания изделия (none, additional_check, item_hold…).
	Containment string
	Incidents   []string
	// ReworkLimit — лимит переделок операции (0 — не задан).
	ReworkLimit int
	// DecisionDays — срок решения по изделию, рабочих дней.
	DecisionDays int
}

func (f DecisionFacts) where() string { return first(f.Where, "текущий шаг") }

func (f DecisionFacts) op() string {
	op := first(f.Operation, "операцию несоответствия")
	if f.Operation != "" {
		op = "«" + f.Operation + "»"
	}
	return op
}

// reworkWhat — «переделка на СВ-017-1» / «переделка операции «Сварка»».
func (f DecisionFacts) reworkWhat(word string) string {
	if f.RunLabel != "" {
		return word + " на " + f.RunLabel
	}
	return word + " операции " + f.op()
}

// ncNo — «несоответствие НС-01» или просто «несоответствие».
func (f DecisionFacts) ncNo() string {
	if f.NCNumber != "" {
		return "несоответствие " + f.NCNumber
	}
	return "несоответствие"
}

// DecisionTexts — надпись, «почему доступно» (когда гард пропускает) и
// последствия решения карточки словами: деловые и технические раздельно.
// op — операция, variant — вариант решения по изделию (disposition).
func DecisionTexts(op, variant string, f DecisionFacts) (label, why string, business, technical []string) {
	hist := "История: решение с подписью уровня 2 — отдельной записью; прежние записи не меняются"
	incidents := func() []string {
		if len(f.Incidents) == 0 {
			return nil
		}
		return []string{"Блок по области риска инцидента " + strings.Join(f.Incidents, ", ") + " остаётся — его снимает решение по области, а не это решение"}
	}
	switch op {
	case dom.ActConfirm:
		business = []string{
			"Сигнал становится подтверждённым несоответствием: изделие " + f.ItemLabel + " дальше по маршруту не идёт до решения по изделию",
			"Решение по изделию — технологу (при специальном процессе — комиссии): задача «Решение по несоответствию», срок " + days(f.DecisionDays),
		}
		if !f.Isolated {
			business = append(business, "Изоляция — отдельным решением «Изолировать до решения»: тогда мастеру уйдёт задача переместить изделие в изолятор")
		}
		return "Подтвердить несоответствие", "Черновик ждёт решения контролёра: сигнал не отклонён", business,
			[]string{"Статус несоответствия: черновик → подтверждено", "1С: без изменений до решения по изделию",
				"История: решение контролёра с подписью уровня 2 — отдельно от исходного сигнала; сигнал не меняется"}
	case dom.ActRejectSignal:
		business = []string{"Несоответствия нет: изделие продолжает маршрут (сейчас — " + f.where() + ")", "Задача технологу не ставится; решение по изделию не нужно"}
		if f.Containment != "" && f.Containment != "none" {
			business = append(business, "Сдерживание изделия этим не снимается — «Снять блок» отдельным решением уполномоченного")
		}
		return "Отклонить сигнал — не дефект", "Черновик ещё не подтверждён: отклонить можно с обязательной причиной", business,
			[]string{"Статус несоответствия: черновик → закрыто (сигнал отклонён)", "1С: без изменений",
				"История: отклонение с причиной — отдельной записью; исходный сигнал не меняется"}
	case dom.ActRecheck:
		return "Назначить доп. проверку", "Доп. проверку можно назначить, пока по изделию нет решения",
			[]string{"Изделие ждёт доп. проверку там, где оно сейчас (" + f.where() + "): автоматически дальше не идёт",
				"Контролёру — срок доп. проверки; «годно» — только по новому результату контроля и решению на точке"},
			[]string{"Сдерживание: доп. проверка", "1С: без изменений", hist}
	case dom.ActIsolate:
		return "Изолировать до решения", "Изделие не в изоляции",
			[]string{"Изделие изолируется: дальше по маршруту не идёт до решения по изделию",
				"Мастеру там, где изделие (" + f.where() + "), — задача переместить его в изолятор",
				"Решение по изолированному изделию — технологу, срок 3 рабочих дня по производственному календарю"},
			[]string{"Положение: изолировано; «физически не перемещено» — до приёмки в изоляторе", "1С: без изменений", hist}
	case dom.ActContainmentSet:
		return "Установить сдерживание", "Защитное действие: доступно уполномоченному всегда",
			[]string{"Изделие под сдерживанием: приёмка и снятие — только уполномоченным", "Мастеру там, где изделие (" + f.where() + "), — остановить и отложить при блоке"},
			[]string{"Сдерживание: новый уровень", "1С: без изменений", hist}
	case dom.ActContainmentRelease:
		business = []string{"Блок снимается: изделие может продолжить маршрут (сейчас — " + f.where() + ")",
			"Снятие блока ≠ годность: годность по-прежнему решает контролёр на точке предъявления"}
		business = append(business, incidents()...)
		return "Снять блок", "Есть действующее сдерживание; снимает только уполномоченный человек", business,
			[]string{"Сдерживание: " + first(f.Containment, "действующее") + " → нет (по снятым основаниям)", "1С: без изменений",
				"История: разрешающее действие человека с подписью уровня 2"}
	case dom.ActVerify:
		business = []string{"Исполнение решения подтверждено повторным контролем: изделие продолжает маршрут — предъявление на следующей точке",
			"Несоответствие можно закрыть"}
		if variant == "repair" {
			business[0] += "; качество — «годно по разрешению на отклонение», не «годно»"
		}
		return "Подтвердить выполнение решения", "Решение исполнено: нужен результат повторного контроля", business,
			[]string{"Статус несоответствия: решение принято → проверено", "1С: возврат из брака в работу (после переделки или ремонта)", hist}
	case dom.ActClose:
		return "Закрыть несоответствие", "Решение по изделию исполнено",
			[]string{"Несоответствие по изделию закрыто — задачи по нему снимаются",
				"Системное расследование (если открыто) не закрывается — у него свой статус"},
			[]string{"Статус несоответствия: → закрыто", "1С: без изменений", hist}
	case dom.ActDisposition:
		return dispositionTexts(variant, f, incidents())
	}
	return op, "", []string{}, []string{hist}
}

// dispositionTexts — решение по изделию (FR-53, FR-54): что будет с
// изделием и маршрутом, кому уйдёт исполнение и чьё ещё решение нужно.
func dispositionTexts(d string, f DecisionFacts, incidents []string) (label, why string, business, technical []string) {
	var pre []string
	switch f.Approvals {
	case dom.ApprovalsPending:
		pre = append(pre, "Сначала — подписи маршрута решения: до них решение записано, но не исполняется (режим 4)")
	case dom.ApprovalsDemoStub:
		pre = append(pre, "Демо: подписи маршрута решения не проверяются — решение исполняется сразу")
	}
	if f.Commission {
		pre = append(pre, "Решение принимает комиссия (специальный процесс): одно решение — всем изделиям окна нарушения")
	}
	hist := "История: необратимое решение по изделию с подписью уровня 2 (режим 4 — маршрут подписей); прежние записи не меняются"
	status := "Статус несоответствия: подтверждено → решение принято"
	why = "Несоответствие подтверждено, решения по изделию ещё нет"
	switch d {
	case "rework":
		label = "Переделка"
		business = []string{
			"Изделие возвращается на операцию " + f.op() + ": " + f.reworkWhat("переделка") + " — новое выполнение со ссылкой на прежнее",
			"Исполнение — мастеру участка: исполнителю задание на переделку; изоляция и блок по этому несоответствию снимаются",
		}
		if f.ReworkLimit > 0 {
			business = append(business, "Лимит переделок операции — "+strconv.Itoa(f.ReworkLimit)+"; сверх лимита — только ремонт по разрешению на отклонение или списание")
		}
		business = append(business, "После переделки — повторный контроль и подтверждение исполнения контролёром")
		technical = []string{status, "Решение по изделию: переделка", "1С: перевод в брак (переделка); после подтверждения исполнения — возврат в работу", hist}
	case "repair":
		label = "Ремонт по разрешению на отклонение"
		if f.Concession != "" {
			label += " " + f.Concession
			why += "; есть действующее разрешение на отклонение " + f.Concession
		}
		pre = append([]string{"Нужно действующее разрешение на отклонение (решение режима 5, свой маршрут подписей)"}, pre...)
		business = []string{
			"Изделие возвращается на операцию " + f.op() + ": " + f.reworkWhat("ремонт") + " по разрешению на отклонение",
			"Расходуется 1 из лимита разрешения" + opt(" ", f.Concession),
			"Исполнение — мастеру участка; изоляция и блок по этому несоответствию снимаются",
			"После ремонта и повторного контроля — «годно по разрешению на отклонение», не «годно»",
		}
		technical = []string{status, "Решение по изделию: ремонт", "1С: перевод в брак (переделка); после подтверждения исполнения — возврат в работу", hist}
	case "use_as_is":
		label = "Использовать как есть по разрешению"
		if f.Concession != "" {
			label += " " + f.Concession
			why += "; есть действующее разрешение на отклонение " + f.Concession
		}
		pre = append([]string{"Нужно действующее разрешение на отклонение (решение режима 5, свой маршрут подписей)"}, pre...)
		business = []string{
			"Изделие идёт дальше по маршруту без переделки" + opt(" — на ", f.Next) + "; качество — «годно по разрешению на отклонение»",
			"Расходуется 1 из лимита разрешения" + opt(" ", f.Concession),
			"Мастеру участка — продолжить маршрут; изоляция и блок по этому несоответствию снимаются",
		}
		technical = []string{status, "Качество: годно по разрешению на отклонение", "1С: перевода в брак нет; при выпуске — с номером разрешения", hist}
	case "scrap":
		label = "Списать (брак)"
		business = []string{
			"Изделие списывается: дальше по маршруту не идёт",
			"Мастеру участка — убрать изделие в брак; замена в заказе — через планирование и 1С",
			"Несоответствие закрывается после исполнения",
		}
		technical = []string{status, "Решение по изделию: списать", "1С: перевод в брак (списание)", hist}
	case "return_to_supplier":
		label = "Вернуть поставщику"
		why = "Только для необработанного изделия (входной контроль)"
		business = []string{
			"Изделие возвращается поставщику — только необработанное; основание претензии — в решении",
			"Мастеру участка (кладовой) — отгрузить поставщику",
		}
		technical = []string{status, "Решение по изделию: вернуть поставщику", "1С: возврат поставщику", hist}
	default:
		return d, "", []string{}, []string{hist}
	}
	business = append(pre, business...)
	business = append(business, incidents...)
	return label, why, business, technical
}

func opt(prefix, s string) string {
	if s == "" {
		return ""
	}
	return prefix + s
}

// days — срок в рабочих днях словами.
func days(n int) string {
	if n <= 0 {
		n = 3
	}
	w := "рабочих дня"
	switch {
	case n%10 == 1 && n%100 != 11:
		w = "рабочий день"
	case n%10 >= 5 || n%10 == 0 || (n%100 >= 11 && n%100 <= 14):
		w = "рабочих дней"
	}
	return strconv.Itoa(n) + " " + w
}

// RunLabel — выполнение операции для людей: «RUN-SV-017-1» → «СВ-017-1»
// (коды операций — кириллицей, как в сопроводительной карте).
func RunLabel(runID string) string {
	id := strings.TrimPrefix(runID, "RUN-")
	for lat, cyr := range map[string]string{"SV-": "СВ-", "MO-": "МО-", "SB-": "СБ-", "IS-": "ИС-", "KP-": "КП-"} {
		if strings.HasPrefix(id, lat) {
			return cyr + strings.TrimPrefix(id, lat)
		}
	}
	return id
}

// decisionFacts — факты для текстов решений из свёртки изделия.
func (s *Service) decisionFacts(ctx context.Context, v *itemView, n dom.NC, c NCCard) DecisionFacts {
	st := v.State()
	f := DecisionFacts{NCNumber: n.Number, ItemLabel: c.ItemLabel, Where: where(v), Commission: n.Commission, Isolated: st.Isolated(),
		Containment: st.ContainmentLevel(), Incidents: incidentsOf(st), DecisionDays: 3}
	if op := c.Happened.Operation; op != nil {
		f.Operation = first(stepName(v, op.StepKey), op.Label)
		f.RunLabel = RunLabel(op.OperationRunID)
		if def := v.Env.Process.Def; def != nil {
			if nd := def.ByStep(op.StepKey); nd != nil {
				if l, ok := nd.ReworkLimit(); ok {
					f.ReworkLimit = l
				}
			}
		}
	}
	if def := v.Env.Process.Def; def != nil {
		if tv, ok := v.Snap.Process.Primary(v.Env.Process); ok {
			if l := nextStepLabel(def, tv.StepKey); l != nil {
				f.Next = "«" + *l + "»"
			}
		}
	}
	if a, err := s.d.Routes.Approvals(ctx, dom.ActDisposition, "ncd-"+n.ID); err == nil {
		f.Approvals = a
	}
	return f
}

// concessionFor — действующее разрешение на отклонение вида kind ("" —
// любое) в области изделия: номер для людей и id для команды.
func (s *Service) concessionFor(ctx context.Context, itemID, kind string, now time.Time) (number, id string) {
	book, err := s.concessionBook(ctx, platform.Moment{})
	if err != nil {
		return "", ""
	}
	for _, c := range book.Sorted() {
		if c.InScope(itemID) && dom.ConcessionGuard(&c, c.GrantedEventID, itemID, kind, now) == nil {
			number, id = first(c.Grant.Number, c.Grant.ConcessionID), c.Grant.ConcessionID
		}
	}
	return number, id
}

// cardActions — решения карточки (to_decide.decisions) с доступностью для
// вошедшего и последствиями.
func (s *Service) cardActions(ctx context.Context, v *itemView, n dom.NC, c NCCard) []NCDecisionAction {
	actor := platform.PrincipalFrom(ctx).PersonID
	now := s.at(ctx, platform.Moment{}, v.RunID)
	st := v.State()
	f := s.decisionFacts(ctx, v, n, c)
	stream := "item:" + v.ItemID
	guard := func(action string, payload any) error {
		return dom.Guard(st, v.Env, v.Upstream(), kernel.Command{Action: action, Actor: actor, Object: stream, BasisSeq: v.BasisSeq,
			GuardStreams: []string{stream}, OccurredAt: now, SignatureLevel: 2, Payload: payload})
	}
	probe := dom.Reason{Text: "проверка доступности"}
	var out []NCDecisionAction
	add := func(op, variant string, err error, policy *string) *NCDecisionAction {
		a := NCDecisionAction{Operation: op, PolicyRef: policy}
		if op == dom.ActDisposition {
			d := variant
			a.Disposition = &d
		}
		a.Label, a.WhyAvailable, a.Consequences, a.TechnicalConsequences = DecisionTexts(op, variant, f)
		a.Allowed = err == nil
		if err != nil {
			a.WhyAvailable = refusalText(err)
		}
		out = append(out, a)
		return &out[len(out)-1]
	}
	for _, op := range c.ToDecide.Decisions {
		switch op {
		case dom.ActConfirm:
			add(op, "", guard(op, dom.ConfirmedData{NCID: n.ID, SignalIDs: n.Draft.SignalIDs, Severity: n.Severity, Reason: probe}), nil)
		case dom.ActRejectSignal:
			add(op, "", guard(op, dom.SignalRejectedData{SignalIDs: n.Draft.SignalIDs, Reason: probe}), nil)
		case dom.ActRecheck:
			add(op, "", guard(op, dom.RecheckData{Method: "visual", Reason: probe}), nil)
		case dom.ActIsolate:
			add(op, "", guard(op, dom.IsolatedData{Reason: probe}), nil)
		case dom.ActContainmentRelease:
			var keys []string
			for _, x := range st.Containment {
				if !x.Released {
					keys = append(keys, x.Key)
				}
			}
			add(op, "", guard(op, dom.ContainmentReleasedData{ReleasedEventIDs: keys, Reason: probe}), nil)
		case dom.ActVerify:
			add(op, n.Disposition, guard(op, dom.DispositionVerifiedData{NCID: n.ID}), nil)
		case dom.ActClose:
			add(op, "", guard(op, dom.ClosedData{NCID: n.ID}), nil)
		case dom.ActPresentation:
			out = append(out, s.cardPresentationActions(ctx, v)...)
		case dom.ActDisposition:
			policy := "полномочие «" + AuthorityNCDisposition + "» (решение по несоответствию, рамки по тяжести)"
			var authErr error
			if s.d.Authorities != nil && actor != "" && !s.d.Authorities.HasAuthority(actor, AuthorityNCDisposition) {
				authErr = kernel.Refuse(errcodes.AccessSignatureRequired, "who", "обладатель полномочия «"+AuthorityNCDisposition+"» (технолог, начальник ОТК, комиссия)")
			}
			for _, d := range Dispositions {
				data := dom.DispositionSetData{NCID: n.ID, Disposition: d, Reason: probe}
				ff := f
				if d == "repair" || d == "use_as_is" {
					ff.Concession, data.ConcessionID = s.concessionFor(ctx, v.ItemID, d, now)
				}
				err := authErr
				if err == nil {
					err = guard(op, data)
				}
				a := NCDecisionAction{Operation: op, Disposition: &d, PolicyRef: &policy}
				a.Label, a.WhyAvailable, a.Consequences, a.TechnicalConsequences = DecisionTexts(op, d, ff)
				if data.ConcessionID != "" {
					id := data.ConcessionID
					a.ConcessionID = &id
				}
				a.Allowed = err == nil
				if err != nil {
					a.WhyAvailable = refusalText(err)
				}
				out = append(out, a)
			}
		}
	}
	return out
}

// cardPresentationActions — решения на ждущем предъявлении из карточки: те
// же, что у presentation.read (гарды и тексты resolveActions).
func (s *Service) cardPresentationActions(ctx context.Context, v *itemView) []NCDecisionAction {
	pr := v.State().PendingPresentation()
	if pr == nil {
		return nil
	}
	p := NCPresentationPoint{EventID: pr.EventID, StepKey: pr.StepKey, ClosingPoint: pr.ClosingPoint, PresentationNo: pr.PresentationNo,
		MethodEventIDs: slices.Clone(v.State().Inspections)}
	if def := v.Env.Process.Def; def != nil {
		p.NextStepLabel = nextStepLabel(def, p.StepKey)
	}
	var out []NCDecisionAction
	for _, a := range s.resolveActions(ctx, v, p) {
		out = append(out, NCDecisionAction{Operation: a.Operation, Resolution: a.Resolution, Label: a.Label, Allowed: a.Allowed,
			WhyAvailable: a.WhyAvailable, Consequences: a.Consequences, TechnicalConsequences: a.TechnicalConsequences, PolicyRef: a.PolicyRef})
	}
	return out
}

// RoleLabel — роль исполнителя в дательном падеже («мастеру участка»).
func RoleLabel(role string) string {
	switch role {
	case "site_foreman":
		return "мастеру участка"
	case "technologist":
		return "технологу"
	case "quality_inspector":
		return "контролёру ОТК"
	case "head_of_qc":
		return "начальнику ОТК"
	case "head_of_workshop":
		return "начальнику цеха"
	case "production_manager":
		return "руководителю производства"
	case "approver":
		return "согласующим по маршруту подписей"
	}
	return role
}

// HandoffStatusLabel — состояние исполнения словами.
func HandoffStatusLabel(status string) string {
	switch status {
	case "in_progress":
		return "исполняется"
	case "done":
		return "исполнено"
	}
	return "ожидает исполнения"
}

// HandoffTitle — что поручено по решению d словами.
func HandoffTitle(d string, f DecisionFacts) string {
	switch d {
	case "rework":
		return "Переделка" + f.reworkWhat("")
	case "repair":
		return "Ремонт" + f.reworkWhat("") + opt(" по разрешению ", f.Concession)
	case "use_as_is":
		return "Продолжить маршрут по разрешению на отклонение" + opt(" ", f.Concession)
	case "scrap":
		return "Убрать изделие в брак (списание)"
	case "return_to_supplier":
		return "Вернуть изделие поставщику"
	}
	return dom.DispositionLabel(d)
}

// handoff — кому передано исполнение решения по изделию (интерфейс 7): задача
// notifications, порождённая решением, а без неё — по состоянию процесса:
// выполнение операции несоответствия после решения — «исполняется» или
// «исполнено»; подтверждение исполнения или закрытие — «исполнено».
func (s *Service) handoff(ctx context.Context, v *itemView, n dom.NC, c NCCard) *NCHandoff {
	if n.DispositionEventID == "" {
		return nil
	}
	st := v.State()
	var at time.Time
	for _, d := range st.Decisions {
		if d.EventID == n.DispositionEventID {
			at = d.At
		}
	}
	h := &NCHandoff{DecisionEventID: n.DispositionEventID, RoleID: "site_foreman", Status: "waiting", Since: at}
	f := s.decisionFacts(ctx, v, n, c)
	if n.ConcessionID != "" {
		f.Concession = n.ConcessionID
		if book, err := s.concessionBook(ctx, platform.Moment{}); err == nil {
			if x, ok := book.Items[n.ConcessionID]; ok && x.Grant.Number != "" {
				f.Concession = x.Grant.Number
			}
		}
	}
	h.TaskTitle = HandoffTitle(n.Disposition, f)
	if !n.Executed {
		// Режим 4: подписи маршрута не собраны — исполнение ещё никому не передано.
		h.RoleID, h.TaskTitle = "approver", "Подписи решения «"+dom.DispositionLabel(n.Disposition)+"» — до них решение не исполняется"
	} else {
		for _, t := range v.Snap.Notifications.Tasks {
			caused := false
			for _, x := range t.Causes {
				caused = caused || x.EventID == n.DispositionEventID
			}
			if !caused {
				continue
			}
			id := t.ID
			h.RoleID, h.TaskTitle, h.TaskID = t.Role, t.Title, &id
			if t.Person != "" {
				p := t.Person
				h.Person = &p
			}
			if t.Closed != nil {
				h.Status, h.Since = "done", t.Closed.At
			}
		}
		if h.Status != "done" {
			switch {
			case n.Status == dom.StatusVerified || n.Status == dom.StatusClosed:
				h.Status = "done"
				for _, d := range st.Decisions {
					if d.EventID == n.VerifiedEventID || d.EventID == n.ClosedEventID {
						h.Since = d.At
					}
				}
			case n.Disposition == "rework" || n.Disposition == "repair":
				if op := c.Happened.Operation; op != nil {
					if r, ok := reworkRun(v, op.StepKey, op.OperationRunID, at); ok {
						h.Status, h.Since = "in_progress", r.StartedAt
						if r.Operator != "" {
							p := r.Operator
							h.Person = &p
						}
						if r.FinishedAt != nil {
							h.Status, h.Since = "done", *r.FinishedAt
						}
					}
				}
			}
		}
	}
	h.RoleLabel = RoleLabel(h.RoleID)
	h.StatusLabel = HandoffStatusLabel(h.Status)
	return h
}

// reworkRun — последнее выполнение операции stepKey, начатое после решения
// (at) и не равное прежнему выполнению prev: из состояния процесса, а без
// описания процесса — по записям начала и конца выполнения во входе.
func reworkRun(v *itemView, stepKey, prev string, at time.Time) (dp.Run, bool) {
	var best dp.Run
	found := false
	for _, id := range sortedKeys(v.Snap.Process.Runs) {
		r := v.Snap.Process.Runs[id]
		if r.StepKey == stepKey && r.RunID != prev && !r.StartedAt.Before(at) && (!found || !r.StartedAt.Before(best.StartedAt)) {
			best, found = r, true
		}
	}
	if found {
		return best, true
	}
	runs := map[string]*dp.Run{}
	for _, r := range v.Input {
		var d runData
		switch r.Type {
		case catalog.OperationRunStarted:
			_ = json.Unmarshal(r.Data, &d)
			if d.StepKey != stepKey || d.OperationRunID == prev || r.OccurredAt.Before(at) {
				continue
			}
			x := &dp.Run{RunID: d.OperationRunID, StepKey: d.StepKey, StartedAt: r.OccurredAt}
			if d.OperatorID != nil {
				x.Operator = *d.OperatorID
			}
			runs[d.OperationRunID] = x
			if !found || !x.StartedAt.Before(best.StartedAt) {
				best, found = *x, true
			}
		case catalog.OperationRunFinished:
			_ = json.Unmarshal(r.Data, &d)
			if x, ok := runs[d.OperationRunID]; ok {
				t := r.OccurredAt
				x.FinishedAt = &t
				if best.RunID == x.RunID {
					best = *x
				}
			}
		}
	}
	return best, found
}

// RefusalText — отказ гарда словами (для заготовок: те же тексты, что у live).
func RefusalText(err error) string { return refusalText(err) }
