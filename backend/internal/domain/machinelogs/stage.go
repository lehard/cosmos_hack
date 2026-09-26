package machinelogs

import "ant/internal/domain/kernel"

// StageState — состояние модуля machinelogs в межизделийной стадии (AD-42).
type StageState struct{}

// Stage — функция модуля machinelogs (временная линия оборудования, окна нарушений специального процесса, FR-151), подключаемая к межизделийной стадии (AD-42,
// точка подключения; порядок вызова задаёт domain/crossitem.Fold). Вход —
// запись стадии: факт без изделия, команда над объектом вне изделия или запись
// изделия с пометкой publish_stage. Выход — адресованные записи в потоки
// изделий и объектов (kernel.NewAddressed с собственными типами модуля),
// occurred_at — наибольший среди причин. Без нового факта стадия не реагирует
// на записи, вызванные её же адресованными записями (тест неподвижной точки).
func Stage(s StageState, r kernel.Record) (StageState, []kernel.Addressed) {
	_ = r
	return s, nil
}
