package ingest

// Seen — принятое ранее сообщение в реестре идемпотентности (AD-7, FR-31):
// ключ — source_id + event_id, содержимое — отпечаток payload.
type Seen struct {
	SourceID    string
	EventID     string
	Fingerprint string
	// Seq — позиция записи в журнале (прежний ответ отправителю).
	Seq int64
	// SourceSeq — номер у источника.
	SourceSeq int64
	// SignatureVerified — подпись проверена (false — профиль demo без подписи).
	SignatureVerified bool
}

// Repeat — вердикт по повтору ключа.
type Repeat string

// Вердикты повтора.
const (
	// RepeatNew — ключ не встречался.
	RepeatNew Repeat = "new"
	// RepeatDuplicate — тот же ключ и то же содержимое: подтвердить прежним
	// ответом, не учитывать повторно (счётчик дублей растёт).
	RepeatDuplicate Repeat = "duplicate"
	// RepeatConflict — тот же ключ, другое содержимое: конфликт целостности —
	// запись CA и событие security (AD-7).
	RepeatConflict Repeat = "conflict"
)

// CheckRepeat — вердикт идемпотентности: prior — запись реестра по ключу
// (nil — не было), fingerprint — отпечаток нового содержимого (ContentFingerprint).
// FR-31, кейс §4.5: повтор не учитывается; тот же ключ с другим содержимым —
// конфликт целостности.
func CheckRepeat(prior *Seen, fingerprint string) Repeat {
	switch {
	case prior == nil:
		return RepeatNew
	case prior.Fingerprint == fingerprint:
		return RepeatDuplicate
	default:
		return RepeatConflict
	}
}
