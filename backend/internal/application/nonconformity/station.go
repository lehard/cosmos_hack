package nonconformity

import (
	"cmp"
	"context"
	"slices"
	"time"

	"ant/internal/application/platform"
	dom "ant/internal/domain/nonconformity"
)

// Окно операции участка (интерфейс 6, стол мастера и начальника цеха; FR-49,
// Д-70): чтение по образцу nonconformity.presentation.read — шаг процесса,
// счётчики, действующие остановки точки процесса, предложение системы
// (остановить или снять — решает человек) и действия с доступностью,
// причиной и последствиями, вычисленными сервером. Кнопки окна — только
// actions этого ответа (nonconformity.process_hold.set / release).

// StationCounters — счётчики шага на момент чтения.
type StationCounters struct {
	Queue           int `json:"queue" minimum:"0" doc:"Ждут на шаге."`
	InProgress      int `json:"in_progress" minimum:"0" doc:"В работе."`
	Passed          int `json:"passed" minimum:"0" doc:"Прошли шаг."`
	Defects         int `json:"defects" minimum:"0" doc:"Физические дефекты за период, не наблюдения (соглашение «Дефект»)."`
	Nonconformities int `json:"nonconformities" minimum:"0" doc:"Открытые несоответствия шага."`
}

// StationHold — действующая остановка точки процесса (FR-49).
type StationHold struct {
	HoldID           string    `json:"hold_id"`
	Reason           string    `json:"reason" doc:"Почему остановлено — словами."`
	Since            time.Time `json:"since" doc:"С какого момента остановлено."`
	Level            string    `json:"level" enum:"process_point_stop,critical_stop" doc:"Стоп точки процесса или критическая остановка."`
	ReleaseCondition string    `json:"release_condition,omitempty" doc:"Условие снятия (что должно случиться)."`
	EquipmentID      string    `json:"equipment_id,omitempty" doc:"Оборудование точки."`
	IncidentID       string    `json:"incident_id,omitempty" doc:"Инцидент, из-за которого остановка."`
	SetEventID       string    `json:"set_event_id,omitempty" doc:"Запись остановки в журнале (переход к записи)."`
}

// StationSuggestion — предложение системы по точке: остановить или снять
// остановку, и почему (решает человек, FR-50 режим 3).
type StationSuggestion struct {
	Outcome string   `json:"outcome" enum:"stop,release" doc:"stop — остановить точку, release — снять остановку."`
	Why     []string `json:"why" doc:"Почему система это предлагает — словами."`
}

// StationAction — действие окна операции участка: операция, доступность для
// вошедшего, почему и последствия.
type StationAction struct {
	Operation    string   `json:"operation" enum:"nonconformity.process_hold.set,nonconformity.process_hold.release" doc:"Операция API."`
	Label        string   `json:"label" doc:"Надпись кнопки."`
	Allowed      bool     `json:"allowed" doc:"Пройдёт гарды для вошедшего (остановка уже действует / не действует, полномочие снятия)."`
	WhyAvailable string   `json:"why_available" doc:"Почему доступно или почему нет — словами."`
	Consequences []string `json:"consequences" doc:"Что произойдёт: изделия, маршрут, точка чистоты, журнал."`
	HoldID       string   `json:"hold_id,omitempty" doc:"Остановка, которую снимает release (путь операции)."`
	Level        string   `json:"level,omitempty" enum:"process_point_stop,critical_stop" doc:"Уровень, с которым set остановит точку (поле команды)."`
	EquipmentID  string   `json:"equipment_id,omitempty" doc:"Оборудование — поле команды set."`
	StepKey      string   `json:"step_key,omitempty" doc:"Шаг — поле команды set."`
}

// StationView — окно операции участка (nonconformity.station.read).
type StationView struct {
	StepKey     string             `json:"step_key"`
	StepLabel   string             `json:"step_label,omitempty" doc:"Имя шага по описанию процесса."`
	EquipmentID string             `json:"equipment_id,omitempty" doc:"Оборудование шага, если известно."`
	Counters    *StationCounters   `json:"counters,omitempty" doc:"Счётчики шага; нет — источник счётчиков недоступен (карточка узла process.node.read)."`
	ActiveHolds []StationHold      `json:"active_holds" doc:"Действующие остановки точки процесса (FR-49)."`
	Suggestion  *StationSuggestion `json:"suggestion,omitempty" doc:"Предложение системы: остановить или снять — решает человек."`
	Actions     []StationAction    `json:"actions" doc:"Действия окна: только их показывает интерфейс."`
	BasisSeq    int64              `json:"basis_seq" doc:"seq, на котором построен ответ."`
}

// AuthProcessHold — полномочие снятия остановки точки процесса (normative/policy grants.authorities).
const AuthProcessHold = "process_hold"

// Надписи и последствия действий окна — одни на live и заготовках.
const (
	LabelHoldSet     = "Остановить точку"
	LabelHoldRelease = "Снять остановку"
)

// HoldSetConsequences — что произойдёт при остановке точки процесса.
func HoldSetConsequences(step string) []string {
	return []string{
		"Операции шага «" + step + "» не начинаются: терминалы исполнителей показывают остановку",
		"Изделия в работе на шаге доделываются, новые встают в очередь перед точкой",
		"Остановка — отдельный объект сдерживания процесса (не блок изделий); запись в журнал и журнал критических действий",
	}
}

// HoldReleaseConsequences — что произойдёт при снятии остановки.
func HoldReleaseConsequences(cleanPoint string) []string {
	return []string{
		"Шаг снова принимает изделия",
		"Точка чистоты: " + cmp.Or(cleanPoint, "первые изделия после снятия") + " — под усиленным контролем",
		"Снятие — разрешающее действие уполномоченного: запись в журнал и журнал критических действий",
	}
}

// StationActions — действия окна по действующим остановкам: остановить,
// если остановки нет; снять каждую действующую — у обладателя полномочия.
func StationActions(step, stepLabel, equipment string, holds []StationHold, canRelease bool) []StationAction {
	out := []StationAction{}
	if len(holds) == 0 {
		out = append(out, StationAction{Operation: dom.ActProcessHoldSet, Label: LabelHoldSet, Allowed: true, Level: "process_point_stop",
			EquipmentID: equipment, StepKey: step, WhyAvailable: "Точка работает — остановить может мастер, начальник цеха и руководитель производства (FR-49)",
			Consequences: HoldSetConsequences(cmp.Or(stepLabel, step))})
	} else {
		out = append(out, StationAction{Operation: dom.ActProcessHoldSet, Label: LabelHoldSet, Allowed: false, Level: "process_point_stop",
			EquipmentID: equipment, StepKey: step, WhyAvailable: "Точка уже остановлена (" + holds[0].HoldID + ")",
			Consequences: HoldSetConsequences(cmp.Or(stepLabel, step))})
	}
	for _, h := range holds {
		a := StationAction{Operation: dom.ActProcessHoldRelease, Label: LabelHoldRelease, HoldID: h.HoldID, Allowed: canRelease,
			Consequences: HoldReleaseConsequences("")}
		if canRelease {
			a.WhyAvailable = "У вас полномочие «" + AuthProcessHold + "»; условие снятия: " + cmp.Or(h.ReleaseCondition, "не задано")
		} else {
			a.WhyAvailable = "Снять остановку может обладатель полномочия «" + AuthProcessHold + "» (начальник цеха, руководитель производства)"
		}
		out = append(out, a)
	}
	return out
}

// Station — окно операции участка (nonconformity.station.read): live —
// действующие остановки из журнала (стадия holdBook) на шаге step_key или на
// оборудовании equipment_id; счётчики — у карточки узла process.node.read.
func (s *Service) Station(ctx context.Context, stepKey, equipmentID string, m platform.Moment) (StationView, error) {
	if !s.live() {
		return s.Unimplemented.Station(ctx, stepKey, equipmentID, m)
	}
	book, err := s.holdBook(ctx)
	if err != nil {
		return StationView{}, err
	}
	v := StationView{StepKey: stepKey, EquipmentID: equipmentID, ActiveHolds: []StationHold{}}
	for _, h := range book.Holds {
		// Остановка точки: по шагу или по оборудованию поста (мастер останавливает ИС-2).
		if !h.Active() || !(h.StepKey == stepKey || equipmentID != "" && h.EquipmentID == equipmentID) {
			continue
		}
		v.ActiveHolds = append(v.ActiveHolds, StationHold{HoldID: h.ID, Reason: h.Reason, Since: h.SetAt, Level: h.Level,
			ReleaseCondition: h.ReleaseCondition, EquipmentID: h.EquipmentID, IncidentID: h.IncidentID, SetEventID: h.SetEventID})
		v.EquipmentID = cmp.Or(v.EquipmentID, h.EquipmentID)
	}
	slices.SortFunc(v.ActiveHolds, func(a, b StationHold) int { return a.Since.Compare(b.Since) })
	actor := platform.PrincipalFrom(ctx).PersonID
	canRelease := s.d.Authorities == nil || s.d.Authorities.HasAuthority(actor, AuthProcessHold)
	v.Actions = StationActions(stepKey, v.StepLabel, v.EquipmentID, v.ActiveHolds, canRelease)
	return v, nil
}
