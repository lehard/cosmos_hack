package cad

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/cad"
)

// Config — зависимости живой реализации ведущих портов (AD-36, режим live).
type Config struct {
	// Reader — адаптер формата файла (integration/cad/kompas).
	Reader Reader
	// Intake — обычный приём фактов (AD-18).
	Intake Intake
	// Projections — проекции cad.* движка (роль projector).
	Projections engineapp.ProjectionStore
	// Env — нормативный слой для перевода связей (EnvFromBPMN версии процесса).
	Env dom.Env
	// Nomenclature — номенклатура учётной системы (nil — сверка не выполняется).
	Nomenclature Nomenclature
	// Clock — доменное «сейчас» при приёме команды (AD-37); nil — системное.
	Clock appjournal.DomainClock
	// MaxFile — предел размера файла (0 — 4 МиБ).
	MaxFile int
}

// Service — реализация live ведущих портов модуля cad (AD-36): импорт
// условной сборки через обычный приём и чтение проекций. Без зависимостей
// (NewService) — заглушка 501.
type Service struct {
	Unimplemented
	cfg Config
}

// NewService создаёт заглушку live (все операции — 501); живую реализацию
// собирает NewLive.
func NewService() *Service { return &Service{} }

// NewLive создаёт живую реализацию.
func NewLive(cfg Config) *Service { return &Service{cfg: cfg} }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// ImportAssembly — cad.assembly.import (FR-94): файл → сборка на нашем языке
// → зоны и ограничения нормативного слоя → факты через обычный приём:
// cad.assembly.imported и соответствия обозначений КД нашим типам
// (reference.external_id.mapped, система kompas). Файл не по контракту —
// 400 api.validation_failed с полем; тот же файл повторно — прежний ответ
// (дубль по отпечатку файла, AD-7).
func (s *Service) ImportAssembly(ctx context.Context, in ImportAssembly) (platform.Receipt, error) {
	if s.cfg.Reader == nil || s.cfg.Intake == nil {
		return s.Unimplemented.ImportAssembly(ctx, in)
	}
	limit := s.cfg.MaxFile
	if limit <= 0 {
		limit = 4 << 20
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.Content))
	if err != nil {
		return platform.Receipt{}, invalid("/content", "не base64: "+err.Error())
	}
	if len(raw) > limit {
		return platform.Receipt{}, invalid("/content", fmt.Sprintf("файл %d байт — больше предела %d", len(raw), limit))
	}
	a, err := s.cfg.Reader.Read(ctx, in.FileName, raw)
	if err != nil {
		var fe *FormatError
		if errors.As(err, &fe) {
			return platform.Receipt{}, invalid(fe.Field, fe.Detail)
		}
		return platform.Receipt{}, err
	}
	var nom *dom.Nomenclature
	if s.cfg.Nomenclature != nil {
		if nom, err = s.cfg.Nomenclature.ItemTypes(ctx); err != nil {
			return platform.Receipt{}, err
		}
	}
	res, problems := dom.Translate(a, s.cfg.Env, nom)
	if len(problems) > 0 {
		return platform.Receipt{}, invalid(problems[0].Field, problems[0].Detail)
	}
	at := time.Now().UTC()
	if s.cfg.Clock != nil {
		if t, err := s.cfg.Clock.Now(ctx); err == nil && !t.IsZero() {
			at = t.UTC()
		}
	}
	facts := Facts(a, res, at)
	envs := make([][]byte, 0, len(facts))
	signer := ""
	if p := platform.PrincipalFrom(ctx); p.PersonID != "" {
		signer = p.PersonID + "@1"
	}
	for _, f := range facts {
		b, err := Envelope(dom.SourceKompas, signer, f)
		if err != nil {
			return platform.Receipt{}, err
		}
		envs = append(envs, b)
	}
	outs, err := s.cfg.Intake.Submit(ctx, dom.SourceKompas, envs)
	if err != nil {
		return platform.Receipt{}, err
	}
	if len(outs) == 0 {
		return platform.Receipt{}, errors.New("cad: приём не вернул итог импорта")
	}
	first := outs[0]
	switch first.Outcome {
	case "quarantined", "conflict":
		code := errcodes.Code(first.Code)
		if code == "" {
			code = errcodes.IngestSchemaViolation
		}
		e := platform.Fail(code, "field", first.Field, "reason", first.Detail, "event_type", string(catalog.CadAssemblyImported), "version", "1")
		e.Detail = "Импорт сборки не принят приёмом: " + first.Detail
		return platform.Receipt{}, e
	}
	rc := platform.Receipt{CommandID: in.CommandID, Seq: first.Seq, RecordedAt: at, Replayed: first.Outcome == "duplicate"}
	for i, o := range outs {
		if o.Outcome == "quarantined" || o.Outcome == "conflict" {
			continue
		}
		id := o.EventID
		if id == "" && i < len(facts) {
			id = facts[i].EventID
		}
		rc.EventIDs = append(rc.EventIDs, id)
	}
	return rc, nil
}

// Facts — факты импорта: cad.assembly.imported (event_id от отпечатка файла)
// и соответствия обозначений КД нашим типам (FR-95).
func Facts(a dom.Assembly, r dom.Result, at time.Time) []Fact {
	out := []Fact{{Type: catalog.CadAssemblyImported, EventID: dom.ImportEventID(a.Digest), OccurredAt: at, Data: dom.Fact(a, r)}}
	for _, m := range r.Mappings {
		out = append(out, Fact{Type: catalog.ReferenceExternalIdMapped, EventID: dom.MappingEventID(m), OccurredAt: at, Data: dom.MappingFact(m)})
	}
	return out
}

func invalid(field, reason string) error {
	if field == "" {
		field = "/"
	}
	e := platform.Fail(errcodes.ApiValidationFailed, "field", field, "reason", reason)
	e.Detail = "Файл условной сборки: " + field + ": " + reason
	return e
}

// Assemblies — импортированные сборки (cad.assembly.list), последние — первыми.
func (s *Service) Assemblies(ctx context.Context, m platform.Moment) (CadAssemblyList, error) {
	if s.cfg.Projections == nil {
		return s.Unimplemented.Assemblies(ctx, m)
	}
	out := CadAssemblyList{Items: []CadAssembly{}}
	var ds []string
	raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionAssemblyIndex, "all")
	if err != nil {
		return out, err
	}
	if ok {
		if err := json.Unmarshal(raw, &ds); err != nil {
			return out, err
		}
	}
	for i := len(ds) - 1; i >= 0; i-- {
		raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionAssembly, ds[i])
		if err != nil {
			return out, err
		}
		if !ok {
			continue
		}
		var im Imported
		if err := json.Unmarshal(raw, &im); err != nil {
			return out, err
		}
		if m.AsOf != nil && im.At.After(*m.AsOf) {
			continue
		}
		out.Items = append(out.Items, View(im))
	}
	return out, nil
}
