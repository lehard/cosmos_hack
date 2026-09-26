package security

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// BusTypes — типы шины безопасности (семейство security без журнала CA).
var BusTypes = []catalog.Type{
	catalog.SecurityAuthFailed, catalog.SecuritySignatureInvalid, catalog.SecurityIdempotencyConflict,
	catalog.SecurityAccessDenied, catalog.SecurityAdmissionDenied, catalog.SecurityKeyAlert, catalog.SecurityKeeperAlert,
	catalog.SecurityIntegrityViolated, catalog.SecurityIntegrityChecked, catalog.SecurityPresenceDeviation,
}

// ToSecurityEvent — событие шины безопасности для интерфейса и подписчиков:
// тяжесть и краткое описание по-русски из data записи.
func ToSecurityEvent(e jc.JournalEntry, ev Event) SecurityEvent {
	at, _ := dj.ParseTime(e.OccurredAt)
	out := SecurityEvent{EventID: e.EventID, EventType: e.EventType, Seq: int64(e.Seq), OccurredAt: at.UTC().Truncate(time.Millisecond),
		Severity: "info", Data: ev.Data}
	var d map[string]any
	_ = json.Unmarshal(ev.Data, &d)
	str := func(k string) string {
		v, _ := d[k].(string)
		return v
	}
	out.SourceID = str("source_id")
	switch catalog.Type(e.EventType) {
	case catalog.SecurityAuthFailed:
		out.Severity, out.Summary = "warning", "Неудачный вход"
	case catalog.SecuritySignatureInvalid:
		out.Severity, out.Summary = "alarm", fmt.Sprintf("Недействительная подпись (%s)%s", str("failure"), suffix(" — источник ", str("source_id")))
	case catalog.SecurityIdempotencyConflict:
		out.Severity, out.Summary = "alarm", fmt.Sprintf("Конфликт целостности: событие %s источника %s пришло с другим содержимым", str("event_id"), str("source_id"))
	case catalog.SecurityAccessDenied:
		out.Severity, out.Summary = "warning", "Отказ в доступе"+suffix(": ", str("action"))
	case catalog.SecurityAdmissionDenied:
		out.Severity, out.Summary = "warning", "Отказ в допуске к рабочему месту"+suffix(": ", str("reason_code"))
	case catalog.SecurityKeyAlert:
		out.Severity, out.Summary = "alarm", "Тревога по ключу"+suffix(": ", str("alert"))
	case catalog.SecurityKeeperAlert:
		out.Severity, out.Summary = "alarm", "Тревога хранителя: "+keeperAlertText(str("alert"))+suffix(" — ", str("detail"))
	case catalog.SecurityIntegrityViolated:
		out.Severity, out.Summary = "alarm", "Нарушение целостности: "+str("detail")
		out.CARef = str("ca_ref")
	case catalog.SecurityIntegrityChecked:
		out.Summary = "Отчёт верификатора: " + VerdictText(str("verdict"))
		if str("verdict") == "violated" {
			out.Severity = "alarm"
		}
	case catalog.SecurityPresenceDeviation:
		out.Severity, out.Summary = "warning", "Отклонение присутствия"
	default:
		out.Summary = e.EventType
	}
	if kind, id, ok := strings.Cut(e.Stream, ":"); ok && id != "" {
		for _, k := range platform.EntityKinds {
			if string(k) == kind {
				out.Object = &platform.DrillRef{Entity: k, ID: id}
			}
		}
	}
	return out
}

func suffix(sep, v string) string {
	if v == "" {
		return ""
	}
	return sep + v
}

// VerdictText — вердикт верификатора по-русски.
func VerdictText(v string) string {
	switch v {
	case "intact":
		return "цело"
	case "intact_with_reservations":
		return "цело с оговорками"
	case "violated":
		return "нарушено"
	}
	return v
}

func keeperAlertText(a string) string {
	switch a {
	case "heads_silent":
		return "головы цепочек не приходили дольше 2N секунд"
	case "fork_attempt":
		return "звенья не сходятся с принятой головой — попытка переписать цепочку"
	case "rollback_attempt":
		return "номер головы меньше принятого — откат"
	case "checkpoint_gap":
		return "разрыв между контрольными точками"
	}
	return a
}
