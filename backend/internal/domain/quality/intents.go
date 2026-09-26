package quality

import (
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
)

// IntentSetQuality — имя намерения смены состояния качества.
const IntentSetQuality = "set_quality"

// SetQuality — функция-намерение quality (AD-30, AD-40): подтверждение
// несоответствия контролёром — решение nonconformity, а ось «состояние
// качества» меняет только quality по этому намерению.
func SetQuality(from kernel.Module, value statuses.Quality, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentSetQuality, value, causes...)
}
