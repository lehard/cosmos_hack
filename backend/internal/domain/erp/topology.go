package erp

// Lane — цех процесса: дорожка BPMN со своим складом (FR-130; свойства
// дорожки `ant:properties/@workshop`, `@warehouse`) и шаги, которые на ней
// лежат (step_key).
type Lane struct {
	// ID — id дорожки BPMN.
	ID string `json:"id"`
	// Workshop — цех (место справочника), Warehouse — его склад.
	Workshop  string `json:"workshop"`
	Warehouse string `json:"warehouse"`
	// Steps — ключи шагов дорожки.
	Steps []string `json:"steps"`
}

// Topology — цеха процесса и их склады в порядке дорожек BPMN (порядок
// laneSet — порядок прохождения цехов). По ней модуль erp определяет склады
// учётного действия: смена склада при передаче изделия между цехами — склад
// цеха-получателя и склад, где изделие числится по прежним сообщениям (FR-130).
// Получается из нормативного слоя (application/erp.TopologyFromBPMN).
type Topology struct {
	Lanes []Lane `json:"lanes"`
}

// LaneOf — номер дорожки шага (ok = false — шаг вне дорожек).
func (t Topology) LaneOf(step string) (int, bool) {
	for i, l := range t.Lanes {
		for _, s := range l.Steps {
			if s == step {
				return i, true
			}
		}
	}
	return 0, false
}

// Warehouse — склад дорожки i (пусто — дорожки нет).
func (t Topology) Warehouse(i int) string {
	if i < 0 || i >= len(t.Lanes) {
		return ""
	}
	return t.Lanes[i].Warehouse
}

// FirstWarehouse — склад первой дорожки (приёмка и входной контроль).
func (t Topology) FirstWarehouse() string { return t.Warehouse(0) }

// StepWarehouse — склад цеха, на дорожке которого лежит шаг.
func (t Topology) StepWarehouse(step string) string {
	if i, ok := t.LaneOf(step); ok {
		return t.Warehouse(i)
	}
	return ""
}
