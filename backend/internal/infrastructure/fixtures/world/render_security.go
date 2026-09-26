package world

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"ant/internal/application/platform"
	securityapp "ant/internal/application/security"
	"ant/internal/contracts/catalog"
	secdom "ant/internal/domain/security"
	"ant/internal/infrastructure/fixtures/loader"
)

// Стол Аудитора ИБ на заготовках (интерфейс 6; FR-73…FR-77, FR-118; AD-9,
// AD-24, AD-28, AD-46): отчёты верификатора, журнал критических действий и
// шина безопасности — из мира заготовок теми же типами, что отдаёт live.
//
//   - отчёты верификатора — по расписанию (каждые 6 часов) и на часах каждого
//     шага; до потери связи со шлюзом ИС-2 — «цело», пока записи шлюза не
//     пришли и после объявленной потери — «цело с оговорками», после подмены
//     записи EV-WS2-0412 (S09) — «нарушено» с местом нарушения, как находит
//     верификатор после `make tamper` (атака 1: звено не сходится);
//   - журнал критических действий — записи мира с признаком критичности по
//     каталогу (AD-40) и критические события шины (отказ в допуске, выдача
//     полномочия): CA-‹n› по порядку записи, подписанты с классом ключа;
//   - шина безопасности — входы, отказы в доступе и допуске, выдача прав,
//     тревоги присутствия по СКУД (эпик 37), отчёты и нарушение целостности.

// auditReportEvery — период отчётов верификатора «по расписанию» в мире
// заготовок (реальный интервал — integrity.interval_minutes; все отчёты
// каждые 15 минут раздули бы заготовки).
const auditReportEvery = 6 * time.Hour

// auditListLimit — сколько последних отчётов в списке.
const auditListLimit = 30

// verifierBuild — хеш бинарника верификатора мира заготовок.
var verifierBuild = Digest([]byte("fixtures/verifier-build/1"))

// integrityAt — индикатор целостности на часах t: статус и время отчёта
// (как security.integrity.read, renderSecurity).
func (m *Model) integrityAt(t time.Time) (status string, checked time.Time) {
	iv := m.Spec.Integrity.IntervalMinutes
	if iv == 0 {
		iv = 15
	}
	status, checked = "ok", t.Truncate(time.Duration(iv)*time.Minute)
	for _, v := range m.Spec.Integrity.Violations {
		if !v.At.Time().After(t) {
			status, checked = "violated", v.At.Time()
		}
	}
	return status, checked
}

// reportDigest — отпечаток отчёта: тот же, что ReportRef индикатора.
func reportDigest(status string, checked time.Time) string {
	return Digest([]byte(fmt.Sprintf("verifier-report/%s/%s", status, checked.UTC().Format(time.RFC3339))))
}

// reportTimes — часы отчётов верификатора, видимых на шаге n: по расписанию
// и на часах шагов; по возрастанию, без повторов.
func (c *Ctx) reportTimes() []time.Time {
	var out []time.Time
	start := c.M.Steps[0].Add(-24 * time.Hour).Truncate(auditReportEvery)
	for t := start; !t.After(c.T); t = t.Add(auditReportEvery) {
		out = append(out, t)
	}
	for k := 0; k <= c.N; k++ {
		out = append(out, c.M.Steps[k])
	}
	slices.SortFunc(out, func(a, b time.Time) int { return a.Compare(b) })
	return slices.CompactFunc(out, func(a, b time.Time) bool { return a.Equal(b) })
}

// verifierReport — отчёт верификатора на часах t (AD-9): проверки по записям
// мира, известным к t; находки — место нарушения или оговорки.
func (c *Ctx) verifierReport(t time.Time) securityapp.VerifierReport {
	status, checked := c.M.integrityAt(t)
	var upTo int64
	var n, signed, decisions, reactions, late int
	classes := map[string]int{"device": 0, "personal": 0, "paper": 0, "partner": 0, "scenario": 0, "genesis": 0, "server_attested": 0}
	for _, e := range c.M.Events {
		if e.Recorded.After(t) || e.Step > c.N {
			continue
		}
		n++
		upTo = max(upTo, e.Seq)
		classes[e.Provenance]++
		switch e.Kind {
		case "decision":
			decisions++
			signed++
		case "reaction":
			reactions++
		case "fact":
			signed++
		}
		if e.Late {
			late++
		}
	}
	row := func(check string, count int, notes ...securityapp.VerifierFinding) securityapp.VerifierCheckRow {
		r := securityapp.VerifierCheckRow{Check: check, Status: "intact", Count: count}
		for _, f := range notes {
			switch {
			case f.Status == "rejected":
				r.Status = "rejected"
			case f.Status == "unverifiable" && r.Status != "rejected":
				r.Status = "unverifiable"
			}
			r.Findings = append(r.Findings, f)
		}
		var details []string
		for _, f := range r.Findings {
			details = append(details, f.Detail)
		}
		r.Details = strings.Join(details, "\n")
		return r
	}
	var chains, sourceSeq, lateWrite, reacts []securityapp.VerifierFinding
	// Шлюз ИС-2: записи копятся у шлюза (оговорка), после восстановления — объявленная потеря.
	for _, s := range c.M.Spec.Sources {
		if s.Lost.IsZero() || t.Before(s.Lost.Time()) {
			continue
		}
		if s.Restored.IsZero() || t.Before(s.Restored.Time()) {
			sourceSeq = append(sourceSeq, securityapp.VerifierFinding{Code: "source_seq_gap.pending", Status: "unverifiable",
				Detail: fmt.Sprintf("источник %s: записей с %s нет ни в журнале, ни в записи о карантине; потеря ещё не объявлена (шлюз копит журнал)", s.ID, s.Lost.Time().In(c.M.clk.loc).Format("02.01 15:04"))})
			continue
		}
		if s.Batch.Lost > 0 {
			sourceSeq = append(sourceSeq, securityapp.VerifierFinding{Code: "source_seq_gap.declared", Status: "unverifiable",
				Detail: fmt.Sprintf("источник %s: номера %s объявлены потерянными (потеря данных источника, %d записей)", s.ID, s.Batch.Gap, s.Batch.Lost)})
		}
		if s.Batch.Late > 0 {
			lateWrite = append(lateWrite, securityapp.VerifierFinding{Code: "late_write.flagged", Status: "note",
				Detail: fmt.Sprintf("источник %s: %d фактов зафиксированы позже события (до %d ч) — флаг «задержка записи»", s.ID, s.Batch.Late, s.Batch.MaxDelayHours)})
		}
	}
	// Подмена записи в обход системы (S09, make tamper, атака 1).
	for _, v := range c.M.Spec.Integrity.Violations {
		if v.At.Time().After(t) {
			continue
		}
		e := c.M.lateRecord(v.Record)
		if e == nil {
			continue
		}
		item := ""
		if e.Item != nil {
			item = ", изделие " + e.Item.Label
		}
		chains = append(chains, securityapp.VerifierFinding{Code: "chain_link_mismatch", Status: "rejected", Chain: "main", Seq: e.Seq, EventID: e.ID,
			Detail: fmt.Sprintf("цепочка main, запись seq %d (%s%s): звено не сходится — запись изменена в обход системы. %s", e.Seq, e.Type, item, v.Note)})
		reacts = append(reacts, securityapp.VerifierFinding{Code: "reactions.content", Status: "rejected", Chain: "main", Seq: e.Seq, EventID: e.ID,
			Detail: fmt.Sprintf("реакции по записи seq %d расходятся со свёрткой входа — содержимое входа изменено (запись %s источника %s)", e.Seq, v.Record, e.Source)})
	}
	checks := []securityapp.VerifierCheckRow{
		row("chains", n, chains...),
		row("checkpoints", max(1, int(t.Sub(c.M.Steps[0].Add(-24*time.Hour))/(30*time.Second)))),
		row("signatures", signed),
		row("signing_moment", decisions),
		row("authority", decisions),
		row("bpmn_quorum", 3),
		row("reactions", reactions, reacts...),
		row("coverage", len(c.M.Items)),
		row("source_seq", len(c.M.Spec.Sources), sourceSeq...),
		row("rendering", decisions),
		row("late_write", late, lateWrite...),
		row("build", 1),
		row("projections", len(c.M.Items)),
		row("genesis", 1),
	}
	verdict := "intact"
	var headline string
	for _, r := range checks {
		switch r.Status {
		case "rejected":
			verdict, headline = "violated", r.Findings[0].Detail
		case "unverifiable":
			if verdict == "intact" {
				verdict, headline = "intact_with_reservations", r.Findings[0].Detail
			}
		}
	}
	if status == "violated" {
		verdict = "violated"
	}
	return securityapp.VerifierReport{
		Summary: securityapp.VerifierReportSummary{ReportDigest: reportDigest(status, checked), Verdict: verdict, CheckedUpToSeq: upTo,
			CheckedAt: checked.UTC(), VerifierBuild: verifierBuild, ServerSide: true, Headline: headline},
		Checks: checks, SignatureClasses: classes, PaperDecisions: []platform.DrillRef{}, VirtualTime: true, SignedBy: "verifier@1",
	}
}

// ─────────────────────────── события шины (синтетические) ───────────────────────────

// auditEv — событие шины безопасности мира заготовок вне записей мира: вход,
// отказ, выдача прав, тревога присутствия по СКУД.
type auditEv struct {
	key                string
	at                 time.Time
	typ                string
	person, workplace  string
	summary, severity  string
	actor, authorityID string
}

// auditEvents — события шины по часам мира (без отчётов верификатора).
func (m *Model) auditEvents() []auditEv {
	at := m.clk.at
	evs := []auditEv{
		{key: "auth-1", at: at(18, 7, 52), typ: string(catalog.SecurityAuthFailed), person: "INS-02", severity: "warning",
			summary: "Неудачный вход: неверный PIN ключа в браузере (Контролёр ОТК 2); следующая попытка — успешно"},
		{key: "grant-1", at: at(18, 9, 30), typ: string(catalog.PolicyAuthorityGranted), person: "INS-02", severity: "info", actor: "ADM-01", authorityID: "first_article",
			summary: "Полномочие выдано: first_article (первое изделие) в области ent01/b1/ac — Контролёр ОТК 2; выдал Администратор безопасности, вторая подпись — Начальник ОТК"},
		{key: "admission-1", at: at(21, 8, 12), typ: string(catalog.SecurityAdmissionDenied), person: "W22", workplace: "WP-WELD-2", severity: "warning",
			summary: "Отказ в допуске к рабочему месту: Сварщик W22 не назначен на пост «Пост сварки 2» в первую смену (not_assigned)"},
		{key: "denied-1", at: at(22, 9, 14), typ: string(catalog.SecurityAccessDenied), person: "W21", severity: "warning",
			summary: "Отказ в доступе: nonconformity.presentation.resolve — у роли «Исполнитель» нет права решения на точке предъявления (Сварщик W21)"},
		{key: "role-1", at: at(23, 8, 5), typ: string(catalog.PolicyRoleAssigned), person: "HWS-AC", severity: "info", actor: "ADM-01",
			summary: "Роль назначена: head_of_workshop в области ent01/b1/ac — Начальник сборочно-испытательного цеха; выдал Администратор безопасности"},
		{key: "auth-2", at: at(23, 22, 41), typ: string(catalog.SecurityAuthFailed), severity: "warning",
			summary: "Неудачный вход: три неверных пароля подряд для учётной записи adm-01 — вход временно ограничен (rate_limited)"},
		{key: "role-2", at: at(24, 9, 0), typ: string(catalog.PolicyRoleUnassigned), person: "O18", severity: "warning", actor: "ADM-01",
			summary: "Роль снята: performer в области ent01/b1/mc — Оператор ЧПУ O18 (перевод в другой цех); снял Администратор безопасности"},
	}
	// Тревоги присутствия по СКУД (эпик 37): «ключ вставлен, владельца нет в зоне».
	for _, p := range m.Spec.People {
		if p.Workplace == "" {
			continue
		}
		c := m.ctx(len(m.Steps) - 1)
		for i, e := range c.postEvents(p.Workplace) {
			if e.kind != "presence_deviation" || e.person != p.Person {
				continue
			}
			evs = append(evs, auditEv{key: fmt.Sprintf("presence-%s-%d", p.Person, i), at: e.at, typ: string(catalog.SecurityPresenceDeviation), person: p.Person,
				workplace: p.Workplace, severity: "alarm",
				summary: fmt.Sprintf("Ключ вставлен, владельца нет в зоне: %s — %s; допуск снят (выход из зоны цеха по СКУД)", m.personName(p.Person), workplaceTitle[p.Workplace])})
		}
	}
	slices.SortStableFunc(evs, func(a, b auditEv) int { return a.at.Compare(b.at) })
	return evs
}

// auditSeq — номер синтетического события в журнале мира заготовок: в конце
// диапазона шага (записи мира занимают его начало).
func (m *Model) auditSeq(t time.Time, i int) int64 {
	return int64(m.StepOf(t))*loader.SeqPerStep + 8000 + int64(i)
}

// ─────────────────────────── журнал критических действий ───────────────────────────

// keyStorageOf — класс хранения ключа человека в мире заготовок (Д-72):
// контролёр ОТК 2, главный сварщик и рабочие — ключ в браузере, остальные — физический ключ.
func keyStorageOf(person string) string {
	switch person {
	case "INS-02", "CWL-01", "W21", "W22", "O17", "O18", "K16", "A31", "T41", "STK-51":
		return "software_browser"
	}
	return "hardware_token"
}

// caAuthority — полномочие, под которым записано решение (normative/policy grants.authorities).
var caAuthority = map[string]string{
	"decision.presentation.resolved": "qc_acceptance", "decision.presentation.reviewed": "qc_acceptance", "decision.disposition.set": "nc_disposition",
	"decision.containment.released": "containment_release", "decision.process_hold.set": "process_hold", "incident.cause.concluded": "cause_confirmation",
	"incident.scope.narrowed": "risk_scope_narrowing", "decision.lot.resolved": "qc_acceptance",
}

// caRow — строка журнала CA мира: момент записи и представление.
type caRow struct {
	at   time.Time
	step int
	v    securityapp.CriticalAction
}

// criticalActions — журнал критических действий мира по порядку записи
// (CA-1 … CA-n); видимость — по шагу записи.
func (m *Model) criticalActions() []caRow {
	var rows []caRow
	signer := func(person, source, provenance string) []securityapp.CriticalActionSigner {
		if person != "" {
			return []securityapp.CriticalActionSigner{{SignerID: person, Display: m.personName(person), KeyClass: "personal",
				KeyStorage: keyStorageOf(person), KeyID: strings.ToLower(person) + "@1"}}
		}
		if provenance == "" {
			provenance = "server_attested"
		}
		return []securityapp.CriticalActionSigner{{SignerID: source, Display: map[bool]string{true: "система (заверено сервером)", false: source}[source == "ant"],
			KeyClass: provenance, KeyID: source + "@1"}}
	}
	for _, e := range m.Events {
		a, info, ok := secdom.ActionFor(catalog.Type(e.Type))
		if !ok {
			continue
		}
		v := securityapp.CriticalAction{CAGroup: info.CAGroup, ActionType: e.Type, ActionName: a.Name, ObjectRef: e.Stream,
			Before: a.Before, After: a.After, ActorID: e.Author, AuthorityID: caAuthority[e.Type], BasisEventIDs: []string{},
			MainEventID: e.ID, MainCommit: Digest([]byte("commit/" + e.ID)), RecordedAt: e.Recorded.UTC(), PolicySeq: ptr(int64(1)),
			Signers: signer(e.Author, e.Source, e.Provenance)}
		if e.Author != "" {
			v.ActorDisplay = m.personName(e.Author)
		}
		if e.Summary != "" {
			v.After = a.After + " — " + e.Summary
		}
		switch {
		case e.Entity.Entity != "":
			v.Object = platform.DrillRef{Entity: platform.EntityKind(e.Entity.Entity), ID: e.Entity.ID}
		case e.Item != nil:
			v.Object = platform.DrillRef{Entity: platform.EntityItem, ID: FullID(e.Item.ID)}
		default:
			v.Object = platform.DrillRef{Entity: platform.EntityIntegrity, ID: "global"}
		}
		rows = append(rows, caRow{at: e.Recorded, step: e.Step, v: v})
	}
	for _, x := range m.auditEvents() {
		a, info, ok := secdom.ActionFor(catalog.Type(x.typ))
		if !ok {
			continue
		}
		id := m.eventID("security/" + x.key)
		v := securityapp.CriticalAction{CAGroup: info.CAGroup, ActionType: x.typ, ActionName: a.Name, ObjectRef: "person:" + x.person,
			Object: platform.DrillRef{Entity: platform.EntityPerson, ID: x.person}, Before: a.Before, After: x.summary, ActorID: x.actor,
			AuthorityID: x.authorityID, BasisEventIDs: []string{}, MainEventID: id, MainCommit: Digest([]byte("commit/" + id)),
			RecordedAt: x.at.UTC(), PolicySeq: ptr(int64(1)), Signers: signer(x.actor, "ant", "server_attested")}
		if x.actor != "" {
			v.ActorDisplay = m.personName(x.actor)
		}
		if x.typ == string(catalog.PolicyAuthorityGranted) {
			v.Signers = append(v.Signers, securityapp.CriticalActionSigner{SignerID: "HQC-01", Display: m.personName("HQC-01"), KeyClass: "personal",
				KeyStorage: keyStorageOf("HQC-01"), KeyID: "hqc-01@1"})
		}
		rows = append(rows, caRow{at: x.at, step: m.StepOf(x.at), v: v})
	}
	slices.SortStableFunc(rows, func(a, b caRow) int { return a.at.Compare(b.at) })
	for i := range rows {
		rows[i].v.CANo = int64(i + 1)
		rows[i].v.CARef = secdom.Ref(int64(i + 1))
	}
	return rows
}

// caCache — номера CA записей мира (event_id → CA-‹n›) по модели.
var caCache sync.Map

// caOf — номер критического действия записи мира по event_id.
func (m *Model) caOf() map[string]string {
	if v, ok := caCache.Load(m); ok {
		return v.(map[string]string)
	}
	out := map[string]string{}
	for _, r := range m.criticalActions() {
		out[r.v.MainEventID] = r.v.CARef
	}
	caCache.Store(m, out)
	return out
}

// renderAudit — ответы стола Аудитора ИБ на шаге.
func renderAudit(c *Ctx) []loader.Response {
	var out []loader.Response
	// Отчёты верификатора: список — последние auditListLimit, отчёт целиком — каждый видимый.
	var reports []securityapp.VerifierReport
	seen := map[string]bool{}
	for _, t := range c.reportTimes() {
		r := c.verifierReport(t)
		if seen[r.Summary.ReportDigest] {
			continue
		}
		seen[r.Summary.ReportDigest] = true
		reports = append(reports, r)
	}
	list := securityapp.VerifierReportList{Items: []securityapp.VerifierReportSummary{}}
	for i := len(reports) - 1; i >= 0 && len(list.Items) < auditListLimit; i-- {
		list.Items = append(list.Items, reports[i].Summary)
		out = append(out, resp("security.verifier_report.read", reports[i], "report_digest", reports[i].Summary.ReportDigest))
	}
	out = append(out, resp("security.verifier_report.list", list))

	// Журнал критических действий на часах шага, новые сверху.
	cas := securityapp.CriticalActionList{Items: []securityapp.CriticalAction{}}
	caOf := map[string]string{}
	rows := c.M.criticalActions()
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if r.step > c.N || r.at.After(c.T) {
			continue
		}
		cas.Items = append(cas.Items, r.v)
		caOf[r.v.MainEventID] = r.v.CARef
		out = append(out, resp("security.critical_action.read", r.v, "ca_ref", r.v.CARef))
	}
	out = append(out, resp("security.critical_action.list", cas))

	// Шина безопасности на часах шага, новые сверху.
	var bus []securityapp.SecurityEvent
	for i, x := range c.M.auditEvents() {
		if x.at.After(c.T) {
			continue
		}
		id := c.M.eventID("security/" + x.key)
		e := securityapp.SecurityEvent{EventID: id, EventType: x.typ, Seq: c.M.auditSeq(x.at, i), OccurredAt: x.at.UTC(), Severity: x.severity,
			SourceID: "ant-security", Summary: x.summary, PersonID: x.person, WorkplaceID: x.workplace, CARef: caOf[id]}
		if x.person != "" {
			e.PersonDisplay = c.M.personName(x.person)
			e.Object = &platform.DrillRef{Entity: platform.EntityPerson, ID: x.person}
		}
		bus = append(bus, e)
	}
	for _, r := range reports {
		sev := map[string]string{"intact": "info", "intact_with_reservations": "warning", "violated": "alarm"}[r.Summary.Verdict]
		sum := "Отчёт верификатора: " + securityapp.VerdictText(r.Summary.Verdict)
		if r.Summary.Headline != "" {
			sum += " — " + r.Summary.Headline
		}
		bus = append(bus, securityapp.SecurityEvent{EventID: c.M.eventID("security/report/" + r.Summary.ReportDigest), EventType: string(catalog.SecurityIntegrityChecked),
			Seq: c.M.auditSeq(r.Summary.CheckedAt, 500+len(bus)), OccurredAt: r.Summary.CheckedAt, Severity: sev, SourceID: "ant-security", Summary: sum,
			Object: &platform.DrillRef{Entity: platform.EntityIntegrity, ID: "global"}})
	}
	for _, e := range c.Visible() {
		if e.Type != string(catalog.SecurityIntegrityViolated) || e.Occurred.After(c.T) {
			continue
		}
		bus = append(bus, securityapp.SecurityEvent{EventID: e.ID, EventType: e.Type, Seq: e.Seq, OccurredAt: e.Occurred.UTC(), Severity: "alarm",
			SourceID: "ant-security", Summary: e.Summary, Object: &platform.DrillRef{Entity: platform.EntityIntegrity, ID: "global"}})
	}
	slices.SortStableFunc(bus, func(a, b securityapp.SecurityEvent) int {
		if x := b.OccurredAt.Compare(a.OccurredAt); x != 0 {
			return x
		}
		return int(b.Seq - a.Seq)
	})
	if len(bus) > 200 {
		bus = bus[:200]
	}
	out = append(out, resp("security.event.list", securityapp.SecurityEventList{Items: append([]securityapp.SecurityEvent{}, bus...)}))
	return out
}
