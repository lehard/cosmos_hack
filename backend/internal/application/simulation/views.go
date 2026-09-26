package simulation

import (
	"time"

	"ant/internal/application/platform"
)

// Формы ответов пульта тестовых сценариев (PRD §3a «Тестовые сценарии»,
// FR-104…FR-108, FR-129, FR-152; AD-26, AD-37, AD-38). Экран — эпик 14.

// Scenario — определение сценария из scenarios/definitions: 8 ситуаций §4.2,
// 9 проверок §5.1, демо-сценарии, сбои (AD-26).
type Scenario struct {
	ScenarioID   string   `json:"scenario_id" doc:"Идентификатор сценария (S01…S13, демо, сбой)."`
	Version      string   `json:"version" doc:"Версия определения сценария."`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	CaseRefs     []string `json:"case_refs,omitempty" doc:"Ссылки на кейс: «§4.2 ситуация 3», «§5.1 проверка 7»."`
	DefaultSeed  int64    `json:"default_seed" minimum:"0"`
	DefaultItems int      `json:"default_items" minimum:"0" doc:"Изделий в прогоне по умолчанию (плюс фоновые)."`
	Assertions   int      `json:"assertions" minimum:"0" doc:"Число утверждений в scenarios/expected — строк табло."`
	Decisions    int      `json:"decisions" minimum:"0" doc:"Сколько раз сценарий останавливается на решении человека."`
}

// ScenarioList — сценарии пульта.
type ScenarioList struct {
	Items []Scenario `json:"items"`
}

// Run — прогон сценария: отдельное пространство имён run_id (AD-38).
type Run struct {
	RunID           string     `json:"run_id"`
	ScenarioID      string     `json:"scenario_id"`
	ScenarioVersion string     `json:"scenario_version"`
	Seed            int64      `json:"seed" minimum:"0"`
	Mode            string     `json:"mode" enum:"interactive,autocheck,load" doc:"interactive — ждёт решения на столах ролей; autocheck — решения подписывает demo-signer (AD-26)."`
	State           string     `json:"state" enum:"running,paused,waiting_for_decision,completed,stopped,failed"`
	Speed           int        `json:"speed" minimum:"1" maximum:"1000" doc:"Ускорение доменных часов ×1…×1000."`
	Step            int        `json:"step" minimum:"0" doc:"Номер шага сценария (на заготовках — шаг курсора, AD-36)."`
	StepTitle       string     `json:"step_title,omitempty" doc:"Название текущего шага сценария."`
	Steps           int        `json:"steps" minimum:"0" doc:"Всего шагов."`
	ClockAt         time.Time  `json:"clock_at" doc:"Доменное «сейчас» прогона — последняя запись time.clock.ticked (AD-37)."`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `nullable:"true" json:"finished_at" doc:"null — прогон идёт."`
	WaitingFor      *RunWait   `json:"waiting_for,omitempty" doc:"На каком решении человека стоит сценарий."`
	Items           int        `json:"items" minimum:"0" doc:"Изделий сценария (без фоновых)."`
	BoardPassed     int        `json:"board_passed" minimum:"0" doc:"Утверждений табло совпало."`
	BoardTotal      int        `json:"board_total" minimum:"0"`
	BasisSeq        int64      `json:"basis_seq" doc:"seq, на котором построен ответ (AD-39)."`
}

// RunWait — решение человека, которого ждёт интерактивный прогон.
type RunWait struct {
	Role     string `json:"role" doc:"Роль стола, где ждут решения."`
	Action   string `json:"action" doc:"x-ant-action id ожидаемой операции."`
	ObjectID string `json:"object_id" doc:"Объект решения (несоответствие, изделие…)."`
	Title    string `json:"title,omitempty" doc:"Что ждёт сценарий, по-русски: «подтвердить сигнал Ф-017»."`
}

// StartedRun — квитанция запуска прогона: команда записана, прогон создан
// под своим run_id (AD-38).
type StartedRun struct {
	platform.Receipt
	RunID string
}

// RunList — прогоны.
type RunList struct {
	Items      []Run  `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// BoardRow — строка табло «ожидалось → получилось» (AD-26, кейс §5.1):
// утверждение scenarios/expected над operationId и путём ответа.
type BoardRow struct {
	AssertionID string  `json:"assertion_id"`
	Title       string  `json:"title" doc:"Что проверяется, по-русски."`
	OperationID string  `json:"operation_id" doc:"Операция API, которой проверяется утверждение (те же Queries)."`
	Path        string  `json:"path" doc:"JSON Pointer в ответе операции."`
	Expected    string  `json:"expected" doc:"Ожидаемое значение (JSON)."`
	Actual      *string `nullable:"true" json:"actual" doc:"Полученное значение (JSON); null — ещё не проверено."`
	Status      string  `json:"status" enum:"pending,passed,failed,not_reached" doc:"not_reached — сценарий не дошёл до шага."`
	Step        int     `json:"step" minimum:"0" doc:"Шаг сценария, после которого проверяется."`
}

// Board — табло прогона.
type Board struct {
	RunID    string     `json:"run_id"`
	Passed   int        `json:"passed" minimum:"0"`
	Failed   int        `json:"failed" minimum:"0"`
	Pending  int        `json:"pending" minimum:"0"`
	Rows     []BoardRow `json:"rows"`
	BasisSeq int64      `json:"basis_seq"`
}

// Injection — кнопка цифрового стенда (FR-152, AD-26): инъекция поверх идущего прогона.
type Injection struct {
	Injection   string `json:"injection" enum:"duplicate_event,late_event,corrupt_frame,machine_fault,data_loss,tamper_outside"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty" doc:"Что произойдёт и где это видно (столы, табло)."`
	Available   bool   `json:"available" doc:"Доступна в текущем состоянии прогона и профиле (tamper_outside — только fixtures и demo)."`
	NeedsTarget bool   `json:"needs_target" doc:"Нужна целевая запись (target_event_id)."`
}

// InjectionList — кнопки цифрового стенда.
type InjectionList struct {
	Items []Injection `json:"items"`
}
