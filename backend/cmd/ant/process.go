package main

import (
	notificationsapp "ant/internal/application/notifications"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"io/fs"
	"time"

	analyticsapp "ant/internal/application/analytics"
	documentsapp "ant/internal/application/documents"
	engineapp "ant/internal/application/engine"
	ingestapp "ant/internal/application/ingest"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	referenceapp "ant/internal/application/reference"
	dp "ant/internal/domain/process"
	"ant/internal/infrastructure/fixtures/world"
)

// Модуль process (эпик 17) на ядре процесса: хранилище версий в схеме
// process, нормативный слой изделия для воркера, запросов на момент и
// пересборки (Bundles, AD-17), live-реализация операций process.

// processSeedFile — стартовый процесс (FR-10): его отпечаток закрепляет блок
// генезиса (normative.version.loaded, эпик 05), байты — встроенная копия
// normative/; без генезиса (разработка) — копия world.Inputs.
const processSeedFile = "normative/process/flange-process.bpmn"

// bundleSource — нормативный слой изделия для воркера, запросов на момент,
// пересборки и гардов команд: версия процесса, закреплённая при запуске
// изделия (эпик 17), слои quality (эпик 20), item (эпик 18) и notifications
// (эпик 24: описание процесса для сроков окон BPMN и точек предъявления) поверх неё.
func (c *core) bundleSource() engineapp.BundleSource {
	// Внешний слой — срез справочников на basis_seq изделия (эпик 19, AD-31):
	// поверка и квалификации для предусловий, производственный календарь сроков;
	// под ним documents (эпик 28): шаблоны документов, срез политики, названия шагов.
	return &referenceapp.Bundles{Next: c.documentsBundles(notificationsapp.Bundles{Next: c.itemBundles(c.qualityBundles(c.bundles))}), Source: c.refSource}
}

// states — запросы состояния изделия на момент с тем же нормативным слоем (AD-22).
func (c *core) states() engineapp.StateQueries {
	return engineapp.StateQueries{Codec: c.codec, Bundles: c.bundleSource()}
}

// ensureProcessSeed — стартовая версия на чистой базе без ручных шагов
// (FR-10); демо-трек: подписана генезисом (AD-33). Без схемы process
// (migrate ещё не прошёл) — предупреждение, повтор при следующем старте роли.
func (c *core) ensureProcessSeed(ctx context.Context, env *environment) {
	c.seedOnce.Do(func() {
		xml, err := genesisProcessXML(ctx, c, env, func() ([]byte, error) { return fs.ReadFile(world.Inputs(), processSeedFile) })
		if err != nil {
			env.log.Error("процесс: нет стартовой версии", "err", err)
			return
		}
		v, err := processapp.EnsureSeed(ctx, c.versions, xml, time.Now().UTC())
		if err != nil {
			env.log.Warn("процесс: стартовая версия не загружена — жду migrate", "err", err)
			return
		}
		env.log.Info("процесс: стартовая версия", "version_id", v.ID, "hash", v.Hash)
	})
}

// processLive — live-реализация операций process для роли api: живая карта и
// карточки узлов по проекциям движка, версии из схемы process, команды
// исполнителя — гард над свёрткой изделия и факт через приём (ingest).
func processLive(ctx context.Context, env *environment, ingest *ingestapp.Service, analytics *analyticsapp.Service, documents *documentsapp.Service) (*processapp.Service, error) {
	c, err := env.readyCore(ctx)
	if err != nil {
		return nil, err
	}
	c.ensureProcessSeed(ctx, env)
	svc := &processapp.LiveService{Store: c.engine, States: c.states(), Library: c.versions, Bundles: c.bundles,
		Clock:    c.domainClock().Now,
		Recorder: &processapp.Recorder{Journal: c.journal, DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions, Now: c.codec.Now}}
	if ingest != nil {
		svc.Facts = ingestFacts{ingest}
	}
	if analytics != nil {
		svc.Counters = analyticsCounters{analytics}
	}
	if documents != nil {
		svc.Approvals = approvalDocs{documents}
	}
	return svc, nil
}

// approvalDocs — лист утверждения версии процесса над documents (эпики 28,
// 39; FR-23, AD-43): документ по шаблону process-version-approval с маршрутом
// кворума, подписи — операциями documents; засчитанные подписи этапов —
// подписи кворума версии.
type approvalDocs struct{ s *documentsapp.Service }

func (a approvalDocs) Request(ctx context.Context, v processapp.VersionRecord, decision, comment string, meta platform.CommandMeta) (string, error) {
	in := documentsapp.RequestVersion{SubjectRef: "process_version:" + v.ID, TemplateRef: processapp.ApprovalTemplate, Decision: decision, Comment: comment}
	// Своя команда документа: id записи листа не должен совпасть с id решения
	// normative.version.submitted (оно — с command_id клиента); повтор даёт тот же id.
	in.CommandHeader = platform.CommandHeader{CommandID: derivedCommandID(meta.CommandID, "approval"), BasisSeq: meta.BasisSeq, PolicySeq: meta.PolicySeq}
	r, err := a.s.RequestVersion(ctx, in)
	if err != nil {
		return "", err
	}
	return r.DocumentID, nil
}

func (a approvalDocs) Route(ctx context.Context, documentID string) (processapp.ApprovalRoute, error) {
	if documentID == "" {
		return processapp.ApprovalRoute{}, platform.Fail("api.not_found", "object", "лист утверждения", "id", "")
	}
	d, err := a.s.Document(ctx, documentID, 0, platform.Moment{})
	if err != nil {
		return processapp.ApprovalRoute{}, err
	}
	out := processapp.ApprovalRoute{Closed: d.RouteClosed != nil || d.Status == "route_closed"}
	if d.RouteClosed != nil {
		out.RouteClosedEventID = *d.RouteClosed
	}
	for _, st := range d.Route {
		out.Need += st.Required
		n := 0
		for _, sg := range st.Signatures {
			if !sg.Counted {
				continue
			}
			n++
			out.Signatures = append(out.Signatures, dp.Signature{Role: st.Role, Person: sg.SignerID, Authority: st.AuthorityID, Valid: true})
		}
		out.Have += min(n, st.Required)
	}
	return out, nil
}

// ingestFacts — FactWriter над ручным вводом приёма (FR-137, FR-141): схема,
// дубли (event_id = command_id), журнал, пометка источника «ручной ввод».
type ingestFacts struct{ s *ingestapp.Service }

func (f ingestFacts) Submit(ctx context.Context, x processapp.Fact) (platform.Receipt, error) {
	r, err := f.s.SubmitManual(ctx, ingestapp.Cmd[ingestapp.ManualInput]{Meta: x.Meta, Body: ingestapp.ManualInput{
		EventType: string(x.EventType), ItemID: x.ItemID, OccurredAt: x.OccurredAt, Data: x.Data}})
	if err != nil {
		return platform.Receipt{}, err
	}
	if r.Code != "" && r.Outcome != ingestapp.OutcomeAccepted && r.Outcome != ingestapp.OutcomeAcceptedWithFlag && r.Outcome != ingestapp.OutcomeDuplicate {
		e := platform.Fail(r.Code, "field", r.Field, "value", r.Value, "reason", r.Detail)
		e.Detail = r.Detail
		return platform.Receipt{}, e
	}
	return platform.Receipt{CommandID: x.Meta.CommandID, Seq: r.Seq, EventIDs: []string{r.EventID}, Replayed: r.Outcome == ingestapp.OutcomeDuplicate}, nil
}

// analyticsCounters — счётчики узлов живой карты от analytics (эпик 25, AD-21).
type analyticsCounters struct{ s *analyticsapp.Service }

func (a analyticsCounters) NodeCounters(ctx context.Context, versionID string, q processapp.LiveMapQuery, m platform.Moment) (processapp.NodeCounterSet, error) {
	set, err := a.s.NodeCounters(ctx, versionID, analyticsapp.PeriodQuery{Kind: q.Period, From: q.From, To: q.To}, m)
	if err != nil {
		return processapp.NodeCounterSet{}, err
	}
	out := processapp.NodeCounterSet{DataGaps: set.DataGaps, BasisSeq: set.BasisSeq}
	for _, c := range set.Counters {
		out.Counters = append(out.Counters, processapp.MapNodeCounters{StepKey: c.StepKey, Queue: c.Queue, InProgress: c.InProgress,
			Passed: c.Passed, Defects: c.Defects, Nonconformities: c.Nonconformities})
	}
	if b := set.Bottleneck; b != nil {
		out.Bottleneck = &processapp.MapBottleneck{StepKey: b.StepKey}
		if b.Wait != nil {
			out.Bottleneck.Wait = *b.Wait
		}
	}
	for _, an := range set.Anomalies {
		x := processapp.MapNodeAnomaly{StepKey: an.StepKey, Kind: an.Kind}
		if an.Threshold != nil {
			x.Threshold = *an.Threshold
		}
		out.Anomalies = append(out.Anomalies, x)
	}
	return out, nil
}

// derivedCommandID — детерминированный UUID производной команды из id
// команды клиента (повтор команды — тот же производный id, AD-7); пусто — пусто.
func derivedCommandID(commandID, purpose string) string {
	if commandID == "" {
		return ""
	}
	h := sha256.Sum256([]byte(strings.ToLower(commandID) + "/" + purpose))
	h[6] = (h[6] & 0x0f) | 0x70
	h[8] = (h[8] & 0x3f) | 0x80
	x := hex.EncodeToString(h[:16])
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}
