package simulation

import "ant/internal/application/platform"

// StartRun — запустить прогон сценария (simulation.run.started, AD-38):
// seed, скорость, число изделий, режим.
type StartRun struct {
	platform.CommandHeader
	Seed  *int64 `json:"seed,omitempty" minimum:"0" doc:"Seed генератора; пусто — из определения сценария."`
	Speed int    `json:"speed,omitempty" minimum:"1" maximum:"1000" doc:"Ускорение ×1…×1000; пусто — скорость определения прогона (показ SHOW-IS2 — ×60, MS-1 — ×1000), на заготовках — ×1."`
	Items int    `json:"items,omitempty" minimum:"0" maximum:"10000" doc:"Не используется: изделия задаёт определение прогона (scenarios/definitions/runs); поле оставлено для совместимости и игнорируется."`
	Mode  string `json:"mode" enum:"interactive,autocheck,load" doc:"interactive — решения на столах ролей; autocheck — demo-signer (AD-26)."`
	// Start — откуда начинать прогон: с начала или с точки старта сценария
	// (start_step заготовок).
	Start string `json:"start,omitempty" enum:"beginning,start_step" doc:"Откуда начинать: beginning — с самого начала (по умолчанию); start_step — с точки старта сценария (start_step в simulation.scenario.list): история сразу «у катастрофы», шаги до точки считаются пройденными. Пока только заготовки (профиль fixtures)."`
	// FromStep — явный шаг старта (только заготовки): сильнее Start.
	FromStep *int `json:"from_step,omitempty" minimum:"0" doc:"Шаг старта на заготовках (профиль fixtures): курсор сразу на этом шаге, шаги до него пройдены; сильнее start. Живой прогон шагов не нумерует — отказ api.validation_failed."`
}

// RunControl — пауза, продолжение, остановка прогона.
type RunControl struct {
	platform.CommandHeader
	Note string `json:"note,omitempty" maxLength:"500"`
}

// SetSpeed — изменить скорость доменных часов прогона.
type SetSpeed struct {
	platform.CommandHeader
	Speed int `json:"speed" minimum:"1" maximum:"1000"`
}

// ApplyInjection — нажать кнопку цифрового стенда (simulation.injection.applied, FR-152).
type ApplyInjection struct {
	platform.CommandHeader
	Injection     string `json:"injection" enum:"duplicate_event,late_event,corrupt_frame,machine_fault,data_loss,tamper_outside,light_change"`
	TargetEventID string `json:"target_event_id,omitempty" format:"uuid" doc:"Целевое событие (повтор, опоздание)."`
}
