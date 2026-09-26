package main

import (
	"context"

	cadapp "ant/internal/application/cad"
	ingestapp "ant/internal/application/ingest"
	referenceapp "ant/internal/application/reference"
	caddom "ant/internal/domain/cad"
	"ant/internal/infrastructure/integration/cad/kompas"
)

// Эпик 31: модуль cad — импорт условной сборки КОМПАС-3D (роль api,
// операция cad.assembly.import) через обычный приём и проекция cad.assembly
// (роль projector, engine.go). Файл читает адаптер порта cad.Reader
// (kompas.Reader); перевод связей W-1, J-1, S-1 в зоны и ограничения — по
// версии процесса (cadapp.EnvFromBPMN); сверка обозначений КД — с
// номенклатурой учётной системы порта учёта.

// cadLive — живая реализация операций cad.* для роли api.
func cadLive(ctx context.Context, env *environment, ingest *ingestapp.Service) (*cadapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	xml, err := processXML(ctx, c, env)
	if err != nil {
		return nil, err
	}
	cenv, err := cadapp.EnvFromBPMN(xml)
	if err != nil {
		return nil, err
	}
	cfg := cadapp.Config{Reader: kompas.Reader{}, Projections: c.engine, Env: cenv, Clock: c.domainClock()}
	if ingest != nil {
		cfg.Intake = cadIntake{ingest}
	}
	if sys := ledgerSystem(env); sys != "" {
		cfg.Nomenclature = cadNomenclature{src: c.refSource, system: sys}
	}
	return cadapp.NewLive(cfg), nil
}

// cadIntake — порт входа cad (application/cad.Intake) над живым приёмом:
// итог по каждому конверту в том же порядке.
type cadIntake struct{ s *ingestapp.Service }

func (i cadIntake) Submit(ctx context.Context, source string, envs [][]byte) ([]cadapp.IntakeOutcome, error) {
	r, err := submitEnvelopes(ctx, i.s, source, envs)
	if err != nil {
		return nil, err
	}
	out := make([]cadapp.IntakeOutcome, len(envs))
	for _, it := range r.Items {
		if it.Index < 0 || it.Index >= len(out) {
			continue
		}
		o := cadapp.IntakeOutcome{Outcome: it.Status}
		switch {
		case it.Status == "accepted" && len(it.Flags) > 0:
			o.Outcome = "accepted_with_flag"
		case it.Status == "rejected":
			o.Outcome = "conflict" // отказ без карантина: импорт не принят, как при конфликте
		}
		if it.EventID != nil {
			o.EventID = *it.EventID
		}
		if it.Seq != nil {
			o.Seq = *it.Seq
		}
		if it.Code != nil {
			o.Code = *it.Code
		}
		out[it.Index] = o
	}
	return out, nil
}

// cadNomenclature — номенклатура учётной системы для сверки обозначений КД
// (FR-95): наши типы изделий с действующим соответствием системы (записи
// reference.external_id.mapped, вид item_type). Соответствий нет —
// номенклатура ещё не получена (nil), сверка не выполняется.
type cadNomenclature struct {
	src    *referenceapp.JournalSource
	system string
}

func (n cadNomenclature) ItemTypes(ctx context.Context) (*caddom.Nomenclature, error) {
	b, err := n.src.Book(ctx, 0)
	if err != nil {
		return nil, err
	}
	types := map[string]bool{}
	for _, v := range b.MappingsView(n.system) {
		if v.Effective && string(v.Data.ObjectKind) == "item_type" {
			types[string(v.Data.InternalID)] = true
		}
	}
	if len(types) == 0 {
		return nil, nil
	}
	return &caddom.Nomenclature{System: n.system, ItemTypes: types}, nil
}
