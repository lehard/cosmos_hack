package nonconformity

import (
	"strings"

	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Детерминированные идентификаторы модуля (AD-4, «Соглашения/Идентификаторы»):
// UUIDv5 от NS_ANT — одинаковы у воркера, воспроизведения и верификатора.

// WindowNCID — id группового несоответствия окна нарушения специального
// процесса (FR-151): один на окно, общий для всех выполнений и изделий окна
// (решение комиссии одно — NC-G1, S05).
func WindowNCID(windowEventID string) string {
	return kernel.UUIDv5(constants.NsAnt, "nc\x1fwindow\x1f"+windowEventID)
}

// DraftNCID — id несоответствия-черновика изделия по его сигналам (FR-51):
// одни и те же сигналы дают тот же id при пересвёртке.
func DraftNCID(itemID string, signalIDs []string) string {
	return kernel.UUIDv5(constants.NsAnt, "nc\x1fdraft\x1f"+itemID+"\x1f"+strings.Join(signalIDs, "\x1e"))
}

// NCNumber — номер несоответствия для людей: «НС-» и первые восемь знаков id.
// Сквозной номер журнала регистрации — проекция; в свёртке изделия номер
// выводится из id, чтобы не зависеть от других изделий (AD-5).
func NCNumber(ncID string) string {
	h := strings.ToUpper(strings.ReplaceAll(ncID, "-", ""))
	if len(h) > 8 {
		h = h[:8]
	}
	return "НС-" + h
}
