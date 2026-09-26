package analysis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/analysis"
	"ant/internal/domain/engine"
)

// Предложения — фабрика генераторов (FR-63, эпик 42). Генератор — ведомый
// порт Generator: получает срез состояния модуля (инциденты, разборы, меры,
// ограничение линии) и возвращает предложения; сам ничего не пишет. Сервис
// записывает каждое новое предложение фактом incident.suggestion.recorded
// (ключ повторения — одно открытое предложение на ключ), дальше — решения
// людей. Новый генератор подключается без изменения ядра: адаптер вызывает
// RegisterGenerator в init своего пакета (или передаётся в Config.Generators).

// Generator — ведомый порт генератора предложений (FR-63).
type Generator interface {
	// ID — имя генератора (порт + адаптер), пишется в предложение: `rules.bottleneck`.
	ID() string
	// Kind — вид предложений (перечисление контракта).
	Kind() string
	// Generate — предложения по срезу состояния; ничего не применяет и не пишет.
	Generate(ctx context.Context, in GeneratorInput) ([]dom.Proposal, error)
}

// GeneratorInput — срез состояния для генераторов.
type GeneratorInput struct {
	// Now — доменное «сейчас» (AD-37).
	Now time.Time
	// Incidents — инциденты с версиями области, решениями и мерами.
	Incidents []dom.IncidentRecord
	// Analyses — разборы обстоятельств всех несоответствий (FR-58).
	Analyses []dom.Analysis
	// Items — изделие каждого несоответствия (nc_id → item_id).
	Items map[string]string
	// Actions — корректирующие меры.
	Actions []dom.CorrectiveAction
	// Line — ограничение линии; nil — порт LineSource не подключён.
	Line *dom.LineBottleneck
}

// LineSource — ведомый порт «ограничение линии» (счётчики узлов analytics,
// FR-5): адаптер собирает cmd/ant над analytics.Queries.
type LineSource interface {
	Bottleneck(ctx context.Context) (*dom.LineBottleneck, error)
}

// ruleGenerator — встроенный генератор на правилах (адаптер порта Generator).
type ruleGenerator struct {
	id, kind string
	run      func(GeneratorInput) []dom.Proposal
}

func (g ruleGenerator) ID() string   { return g.id }
func (g ruleGenerator) Kind() string { return g.kind }
func (g ruleGenerator) Generate(_ context.Context, in GeneratorInput) ([]dom.Proposal, error) {
	return g.run(in), nil
}

// RuleGenerators — встроенные генераторы на правилах: ограничение линии,
// область риска, кандидаты в правила реакции, адаптация VisionQC, карта
// дефицита данных (FR-63, FR-143).
func RuleGenerators() []Generator {
	return []Generator{
		ruleGenerator{"rules.bottleneck", dom.SuggestBottleneck, func(in GeneratorInput) []dom.Proposal {
			if in.Line == nil {
				return nil
			}
			if p, ok := dom.BottleneckProposal(*in.Line); ok {
				return []dom.Proposal{p}
			}
			return nil
		}},
		ruleGenerator{"rules.risk_scope", dom.SuggestRiskScope, func(in GeneratorInput) []dom.Proposal { return dom.RiskScopeProposals(in.Incidents) }},
		ruleGenerator{"rules.reaction_rules", dom.SuggestReactionRule, func(in GeneratorInput) []dom.Proposal {
			return dom.ReactionRuleCandidates(in.Incidents)
		}},
		ruleGenerator{"rules.vision_adaptation", dom.SuggestAnalyzer, func(in GeneratorInput) []dom.Proposal {
			return dom.AnalyzerAdaptationProposals(in.Analyses)
		}},
		ruleGenerator{"rules.data_deficit", dom.SuggestDataDeficit, func(in GeneratorInput) []dom.Proposal {
			return dom.DeficitProposals(dom.BuildDeficitMap(in.Analyses, in.Incidents, in.Items))
		}},
	}
}

var (
	pluginMu sync.RWMutex
	plugins  []Generator
)

// RegisterGenerator подключает генератор-адаптер (FR-63, NFR «новый генератор
// без изменения ядра»): вызывается из init пакета адаптера, который cmd/ant
// импортирует; генератор на ИИ — такой же адаптер (его предложения — факты
// источника-генератора, AD-3).
func RegisterGenerator(g Generator) {
	pluginMu.Lock()
	defer pluginMu.Unlock()
	plugins = append(plugins, g)
}

// generators — встроенные (или из Config.Generators) и подключённые адаптеры.
func (s *Service) generators() []Generator {
	gs := s.cfg.Generators
	if gs == nil {
		gs = RuleGenerators()
	}
	pluginMu.RLock()
	defer pluginMu.RUnlock()
	return append(slices.Clone(gs), plugins...)
}

// ConnectLine подключает порт ограничения линии (cmd/ant, над analytics).
func (s *Service) ConnectLine(l LineSource) { s.cfg.Line = l }

// ── запись факта предложения ──

// Fact — факт модуля analysis от имени сервера (предложение генератора, AD-3).
type Fact struct {
	Type      catalog.Type
	EventID   string
	Stream    string
	Data      any
	Generator string
	At        time.Time
}

// FactWriter — ведомый порт записи фактов модуля (предложения): одна пачка
// journal.Append; повтор с тем же event_id — дубль (AD-7).
type FactWriter interface {
	WriteFact(ctx context.Context, f Fact) (platform.Receipt, error)
}

// SourceSuggestions — source_id фактов генераторов предложений.
const SourceSuggestions = "ant-suggestions"

var _ FactWriter = JournalDecisions{}

// WriteFact — факт предложения в журнал: вид «факт», класс происхождения
// server_attested (каталог incident.suggestion.recorded). Конверт — DSSE без
// подписей, как у решений демо (Д-30); доверие — пересчёт, а не ключ (AD-3).
func (w JournalDecisions) WriteFact(ctx context.Context, f Fact) (platform.Receipt, error) {
	info, ok := catalog.Lookup(f.Type)
	if !ok || info.Emitter != string(dom.Module) || info.Kind != catalog.KindFact {
		return platform.Receipt{}, errors.New("analysis: факт чужого типа или не факт: " + string(f.Type))
	}
	data, err := json.Marshal(f.Data)
	if err != nil {
		return platform.Receipt{}, err
	}
	occurred := engineapp.FormatTime(f.At)
	env := map[string]any{
		"event_id": f.EventID, "event_type": string(f.Type), "schema_version": info.CurrentVersion, "source_id": SourceSuggestions,
		"occurred_at": occurred, "correlation_id": f.EventID, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{"engine@1"}},
		"data":      json.RawMessage(data),
	}
	canon, err := engine.Canonical(env)
	if err != nil {
		return platform.Receipt{}, err
	}
	sealed, _ := json.Marshal(struct {
		PayloadType string   `json:"payloadType"`
		Payload     string   `json:"payload"`
		Signatures  []string `json:"signatures"`
	}{engineapp.PayloadTypeEvent, base64.StdEncoding.EncodeToString(canon), []string{}})
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindFact, EventType: string(f.Type),
		SchemaVersion: info.CurrentVersion, EventID: f.EventID, SourceID: SourceSuggestions, Stream: f.Stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now()), CorrelationID: f.EventID,
		ProvenanceClass: jc.JournalEntryProvenanceClassServerAttested, DomainBuild: w.DomainBuild,
	}
	res, err := w.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}})
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	rc := platform.Receipt{CommandID: f.EventID, EventIDs: []string{f.EventID}, RecordedAt: res.Committed}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[0]
	}
	return rc, nil
}

// ── ведущие порты предложений ──

// SuggestionQueries — чтение предложений, мер и карты дефицита (FR-63, FR-64, FR-138, FR-143).
type SuggestionQueries interface {
	// Suggestions — предложения и генераторы (analysis.suggestion.list).
	Suggestions(ctx context.Context, m platform.Moment) (SuggestionList, error)
	// CorrectiveActions — меры с планами, флагами и взглядом руководителя по качеству (analysis.action.list).
	CorrectiveActions(ctx context.Context, m platform.Moment) (CorrectiveActionList, error)
	// DataDeficit — карта дефицита данных (analysis.data_deficit.read).
	DataDeficit(ctx context.Context, m platform.Moment) (DataDeficitMap, error)
}

// SuggestionCommands — команды по предложениям.
type SuggestionCommands interface {
	// GenerateSuggestions — прогнать генераторы и записать новые предложения (analysis.suggestion.generate).
	GenerateSuggestions(ctx context.Context, in GenerateSuggestions) (platform.Receipt, error)
	// ForwardSuggestion — передать ответственному (analysis.suggestion.forward).
	ForwardSuggestion(ctx context.Context, suggestionID string, in ForwardSuggestion) (platform.Receipt, error)
	// ResolveSuggestion — принять в работу или отклонить (analysis.suggestion.resolve).
	ResolveSuggestion(ctx context.Context, suggestionID string, in ResolveSuggestion) (platform.Receipt, error)
}

// GenerateSuggestions — команда «сформировать предложения».
type GenerateSuggestions struct {
	platform.CommandHeader
}

// ForwardSuggestion — передать предложение ответственному (UJ-1).
type ForwardSuggestion struct {
	platform.CommandHeader
	ResponsibleID   string `json:"responsible_id" maxLength:"128" doc:"Псевдоним ответственного."`
	ResponsibleRole string `json:"responsible_role,omitempty" maxLength:"64" doc:"Роль: site_foreman — по персоналу, technologist — если нужна новая версия процесса."`
	Note            string `json:"note,omitempty" maxLength:"4000"`
}

// ResolveSuggestion — решение по предложению.
type ResolveSuggestion struct {
	platform.CommandHeader
	Resolution string `json:"resolution" enum:"accepted,rejected" doc:"Принято в работу / отклонено."`
	Reason     Reason `json:"reason"`
}

// UnimplementedSuggestions — заглушка портов предложений (501).
type UnimplementedSuggestions struct{}

func (UnimplementedSuggestions) Suggestions(context.Context, platform.Moment) (SuggestionList, error) {
	return SuggestionList{}, ni("analysis.suggestion.list")
}
func (UnimplementedSuggestions) CorrectiveActions(context.Context, platform.Moment) (CorrectiveActionList, error) {
	return CorrectiveActionList{}, ni("analysis.action.list")
}
func (UnimplementedSuggestions) DataDeficit(context.Context, platform.Moment) (DataDeficitMap, error) {
	return DataDeficitMap{}, ni("analysis.data_deficit.read")
}
func (UnimplementedSuggestions) GenerateSuggestions(context.Context, GenerateSuggestions) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.suggestion.generate")
}
func (UnimplementedSuggestions) ForwardSuggestion(context.Context, string, ForwardSuggestion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.suggestion.forward")
}
func (UnimplementedSuggestions) ResolveSuggestion(context.Context, string, ResolveSuggestion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.suggestion.resolve")
}

var (
	_ SuggestionQueries  = UnimplementedSuggestions{}
	_ SuggestionCommands = UnimplementedSuggestions{}
	_ SuggestionQueries  = (*Service)(nil)
	_ SuggestionCommands = (*Service)(nil)
)

// ── реализация live ──

// input — срез состояния для генераторов и карты дефицита.
func (s *Service) input(ctx context.Context, withLine bool) (GeneratorInput, error) {
	now, err := s.now(ctx)
	if err != nil {
		return GeneratorInput{}, err
	}
	in := GeneratorInput{Now: now, Items: map[string]string{}}
	ids, err := s.list(ctx, ProjectionIncident)
	if err != nil {
		return in, err
	}
	for _, id := range ids {
		if v, err := s.incident(ctx, id); err == nil {
			in.Incidents = append(in.Incidents, v)
		}
	}
	ncIDs, err := s.list(ctx, ProjectionNC)
	if err != nil {
		return in, err
	}
	items := map[string]ItemView{}
	for _, id := range ncIDs {
		n, err := s.nc(ctx, id)
		if err != nil {
			continue
		}
		iv, ok := items[n.ItemID]
		if !ok {
			if iv, err = s.item(ctx, n.ItemID); err != nil {
				return in, err
			}
			items[n.ItemID] = iv
		}
		a, ok := dom.Analyze(iv.State, n.ItemID, id, iv.Equipment)
		if !ok {
			continue
		}
		if a.Profile.DefectType == "" {
			a.Profile.DefectType = n.DefectType
		}
		in.Analyses = append(in.Analyses, a)
		in.Items[id] = n.ItemID
	}
	if in.Actions, err = s.actions(ctx); err != nil {
		return in, err
	}
	if withLine && s.cfg.Line != nil {
		// Ограничение линии — необязательный вход: сбой analytics не мешает остальным генераторам.
		if b, err := s.cfg.Line.Bottleneck(ctx); err == nil {
			in.Line = b
		}
	}
	return in, nil
}

// suggestion — предложение из проекции analysis.suggestion.
func (s *Service) suggestion(ctx context.Context, id string) (dom.SuggestionRecord, error) {
	var v dom.SuggestionRecord
	ok, err := s.get(ctx, ProjectionSuggestion, id, &v)
	if err != nil {
		return v, err
	}
	if !ok || v.SuggestionID == "" {
		return v, notFound("Предложение", id)
	}
	return v, nil
}

// Suggestions — предложения (новые сверху) и генераторы.
func (s *Service) Suggestions(ctx context.Context, m platform.Moment) (SuggestionList, error) {
	if !s.live() {
		return UnimplementedSuggestions{}.Suggestions(ctx, m)
	}
	out := SuggestionList{Items: []Suggestion{}, Generators: []GeneratorInfo{}}
	ids, err := s.list(ctx, ProjectionSuggestion)
	if err != nil {
		return out, err
	}
	for _, id := range ids {
		v, err := s.suggestion(ctx, id)
		if err != nil {
			continue
		}
		if m.AsOf != nil && v.RecordedAt.After(*m.AsOf) {
			continue
		}
		out.Items = append(out.Items, suggestionView(v))
		out.BasisSeq = max(out.BasisSeq, v.BasisSeq)
	}
	slices.SortStableFunc(out.Items, func(a, b Suggestion) int { return b.RecordedAt.Compare(a.RecordedAt) })
	for _, g := range s.generators() {
		gi := GeneratorInfo{ID: g.ID(), Kind: g.Kind(), Connected: true}
		if g.Kind() == dom.SuggestBottleneck && s.cfg.Line == nil {
			gi.Connected = false
		}
		out.Generators = append(out.Generators, gi)
	}
	return out, nil
}

// GenerateSuggestions — прогон всех генераторов (analysis.suggestion.generate):
// квитанция перечисляет event_id новых предложений; уже записанные не повторяются.
func (s *Service) GenerateSuggestions(ctx context.Context, in GenerateSuggestions) (platform.Receipt, error) {
	if !s.live() {
		return UnimplementedSuggestions{}.GenerateSuggestions(ctx, in)
	}
	run, err := s.Generate(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if len(run.Recorded) == 0 && len(run.Failed) > 0 {
		return platform.Receipt{}, errors.New("analysis: генераторы предложений: " + strings.Join(run.Failed, "; "))
	}
	now, _ := s.now(ctx)
	return platform.Receipt{CommandID: in.CommandID, Seq: run.Seq, EventIDs: run.EventIDs, RecordedAt: now}, nil
}

// Generate — прогон всех генераторов: каждое предложение с новым ключом
// повторения записывается фактом; уже записанное (тот же id) — пропускается.
// Сбой одного генератора не останавливает остальные. Тот же сценарий может
// вызывать планировщик.
func (s *Service) Generate(ctx context.Context) (SuggestionRun, error) {
	fw, ok := s.cfg.Decisions.(FactWriter)
	if !ok {
		return SuggestionRun{}, ni("analysis.suggestion.generate")
	}
	src, err := s.input(ctx, true)
	if err != nil {
		return SuggestionRun{}, err
	}
	run := SuggestionRun{Recorded: []string{}, EventIDs: []string{}, Failed: []string{}}
	for _, g := range s.generators() {
		ps, err := g.Generate(ctx, src)
		if err != nil {
			run.Failed = append(run.Failed, g.ID()+": "+err.Error())
			continue
		}
		for _, p := range ps {
			if p.Generator == "" {
				p.Generator = g.ID()
			}
			if p.Kind == "" {
				p.Kind = g.Kind()
			}
			id := dom.SuggestionID(p.Generator, p.DedupKey)
			if _, err := s.suggestion(ctx, id); err == nil {
				run.Skipped++
				continue
			}
			eventID := dom.SuggestionEventID(p.Generator, p.DedupKey)
			rc, err := fw.WriteFact(ctx, Fact{Type: catalog.IncidentSuggestionRecorded, EventID: eventID,
				Stream: dom.SuggestionStream(id), Data: suggestionData(id, p), Generator: p.Generator, At: src.Now})
			if errors.Is(err, appjournal.ErrDuplicate) {
				run.Skipped++
				continue
			}
			if err != nil {
				if pe, ok := platform.AsError(err); ok && pe.Code == errcodes.JournalDuplicate {
					run.Skipped++
					continue
				}
				run.Failed = append(run.Failed, g.ID()+": "+err.Error())
				continue
			}
			run.Recorded = append(run.Recorded, id)
			run.EventIDs = append(run.EventIDs, eventID)
			run.Seq = max(run.Seq, rc.Seq)
		}
	}
	return run, nil
}

// suggestionData — data факта incident.suggestion.recorded.
func suggestionData(id string, p dom.Proposal) map[string]any {
	basis := p.Basis
	if basis == nil {
		basis = []string{}
	}
	d := map[string]any{"suggestion_id": id, "generator": p.Generator, "kind": p.Kind, "statement": trim(p.Statement, 4000), "basis": sortedIDs(basis)}
	put := func(k, v string, n int) {
		if v != "" {
			d[k] = trim(v, n)
		}
	}
	put("title", p.Title, 256)
	put("estimate", p.Estimate, 1000)
	put("responsible_role", p.ResponsibleRole, 64)
	put("step_key", p.StepKey, 256)
	put("incident_id", p.IncidentID, 128)
	put("missing_kind", p.MissingKind, 64)
	put("dedup_key", p.DedupKey, 256)
	return d
}

// ForwardSuggestion — руководитель передаёт предложение ответственному; задачу
// ответственному ставит notifications по этой записи.
func (s *Service) ForwardSuggestion(ctx context.Context, suggestionID string, in ForwardSuggestion) (platform.Receipt, error) {
	if !s.live() {
		return UnimplementedSuggestions{}.ForwardSuggestion(ctx, suggestionID, in)
	}
	v, err := s.suggestion(ctx, suggestionID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := dom.GuardSuggestionOpen(v, "передать можно только открытое предложение"); err != nil {
		return platform.Receipt{}, refusal(err)
	}
	if in.ResponsibleID == "" {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "responsible_id", "reason", "не указан ответственный")
		e.Detail = "Укажите, кому передать предложение"
		return platform.Receipt{}, e
	}
	data := map[string]any{"suggestion_id": suggestionID, "responsible_id": in.ResponsibleID}
	role := in.ResponsibleRole
	if role == "" {
		role = v.ResponsibleRole
	}
	if role != "" {
		data["responsible_role"] = role
	}
	if v.Title != "" {
		data["title"] = trim(v.Title, 256)
	}
	if in.Note != "" {
		data["note"] = in.Note
	}
	return s.decideSuggestion(ctx, catalog.IncidentSuggestionForwarded, suggestionID, in.CommandMeta(), data)
}

// ResolveSuggestion — принять в работу или отклонить; с основанием.
func (s *Service) ResolveSuggestion(ctx context.Context, suggestionID string, in ResolveSuggestion) (platform.Receipt, error) {
	if !s.live() {
		return UnimplementedSuggestions{}.ResolveSuggestion(ctx, suggestionID, in)
	}
	v, err := s.suggestion(ctx, suggestionID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := dom.GuardSuggestionOpen(v, "решение по предложению уже принято"); err != nil {
		return platform.Receipt{}, refusal(err)
	}
	if in.Reason.Text == "" {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "reason.text", "reason", "нужно основание")
		e.Detail = "Укажите основание решения"
		return platform.Receipt{}, e
	}
	data := map[string]any{"suggestion_id": suggestionID, "resolution": in.Resolution, "reason": reasonOf(in.Reason)}
	return s.decideSuggestion(ctx, catalog.IncidentSuggestionResolved, suggestionID, in.CommandMeta(), data)
}

// decideSuggestion — решение в поток suggestion:‹id› с проверкой basis_seq (AD-39).
func (s *Service) decideSuggestion(ctx context.Context, t catalog.Type, id string, meta platform.CommandMeta, data any) (platform.Receipt, error) {
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := dom.SuggestionStream(id)
	return s.cfg.Decisions.Write(ctx, Decision{Type: t, Stream: stream, Data: data, Meta: meta, Actor: platform.PrincipalFrom(ctx).PersonID,
		OccurredAt: now, GuardStreams: []string{stream}})
}

func suggestionView(v dom.SuggestionRecord) Suggestion {
	out := Suggestion{SuggestionID: v.SuggestionID, Generator: v.Generator, Kind: v.Kind, Title: v.Title, Statement: v.Statement,
		Estimate: strp(v.Estimate), ResponsibleRole: strp(v.ResponsibleRole), ResponsibleID: strp(v.ResponsibleID), StepKey: strp(v.StepKey),
		IncidentID: strp(v.IncidentID), MissingKind: strp(v.MissingKind), Basis: append([]string{}, v.Basis...), Status: v.Status,
		RecordedAt: v.RecordedAt, EventID: v.EventID, BasisSeq: v.BasisSeq, History: []SuggestionHistory{}}
	for _, h := range v.History {
		out.History = append(out.History, SuggestionHistory{Type: h.Type, Actor: strp(h.Actor), At: h.At, ResponsibleID: strp(h.ResponsibleID),
			ResponsibleRole: strp(h.Role), Text: strp(h.Text), EventID: h.EventID})
	}
	return out
}

func trim(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func sortedIDs(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return slices.Compact(out)
}
