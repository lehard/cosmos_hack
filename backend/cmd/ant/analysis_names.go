package main

import (
	"context"
	"time"

	accessapp "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	referenceapp "ant/internal/application/reference"
)

// analysisNames — порт analysis.Names (стол технолога): названия для людей
// из тех же справочников, что у карточки НС и аналитики — имена людей из
// каталога доступа, оборудование и партии — справочник (эпик 19), виды
// дефектов — классификатор нормативного слоя, шаги — узлы BPMN действующей
// версии процесса. Кода нет в справочнике — названия нет (не выдумываем).
type analysisNames struct {
	dir     *accessapp.Directory
	ref     *referenceapp.JournalSource
	bundles engineapp.BundleSource
	steps   *activeProcess
}

// PersonName — имя человека по псевдониму.
func (n analysisNames) PersonName(_ context.Context, id string) (string, bool) {
	if n.dir == nil || id == "" {
		return "", false
	}
	p, ok := n.dir.Persona(id)
	return p.Name, ok && p.Name != ""
}

// FactorLabel — значение общего фактора словами по его виду.
func (n analysisNames) FactorLabel(ctx context.Context, factor, value string) (string, bool) {
	switch factor {
	case "machine", "tool", "fixture":
		return referenceapp.EquipmentNames{Source: n.ref}.EquipmentName(ctx, value, time.Now())
	case "performer":
		return n.PersonName(ctx, value)
	case "material_batch":
		if n.ref == nil || value == "" {
			return "", false
		}
		b, err := n.ref.Book(ctx, 0)
		if err != nil {
			return "", false
		}
		for _, l := range b.Lots {
			if string(l.Data.LotID) == value && l.Data.ExternalNumber != "" {
				return "Партия " + l.Data.ExternalNumber, true
			}
		}
	}
	return "", false
}

// DefectLabel — вид дефекта по-русски из классификатора нормативного слоя.
func (n analysisNames) DefectLabel(ctx context.Context, code string) (string, bool) {
	if n.bundles == nil || code == "" {
		return "", false
	}
	b, _, err := n.bundles.Bundle(ctx, "", nil)
	if err != nil {
		return "", false
	}
	for _, d := range b.Quality.Classifier.DefectTypes {
		if d.Code == code && d.Name != "" {
			return d.Name, true
		}
	}
	return "", false
}

// StepName — имя узла BPMN действующей версии процесса.
func (n analysisNames) StepName(ctx context.Context, stepKey string) (string, bool) {
	if n.steps == nil {
		return "", false
	}
	names, err := n.steps.StepNames(ctx)
	if err != nil {
		return "", false
	}
	s, ok := names[stepKey]
	return s, ok && s != ""
}
