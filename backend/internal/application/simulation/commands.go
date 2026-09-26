package simulation

import "ant/internal/application/platform"

// StartRun — запустить прогон сценария (simulation.run.started, AD-38):
// seed, скорость, число изделий, режим.
type StartRun struct {
	platform.CommandHeader
	Seed  *int64 `json:"seed,omitempty" minimum:"0" doc:"Seed генератора; пусто — из определения сценария."`
	Speed int    `json:"speed,omitempty" minimum:"1" maximum:"1000" doc:"Ускорение ×1…×1000; пусто — ×1."`
	Items int    `json:"items,omitempty" minimum:"0" maximum:"10000" doc:"Изделий сценария; пусто — по определению."`
	Mode  string `json:"mode" enum:"interactive,autocheck,load" doc:"interactive — решения на столах ролей; autocheck — demo-signer (AD-26)."`
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
	Injection     string `json:"injection" enum:"duplicate_event,late_event,corrupt_frame,machine_fault,data_loss,tamper_outside"`
	TargetEventID string `json:"target_event_id,omitempty" format:"uuid" doc:"Целевое событие (повтор, опоздание)."`
}
