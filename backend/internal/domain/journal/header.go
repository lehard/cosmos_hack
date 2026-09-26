package journal

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
)

// TimeLayout — формат времени записи: RFC 3339 UTC, ровно три знака после
// секунд (соглашения спайна, «Время»).
const TimeLayout = "2006-01-02T15:04:05.000Z"

// FormatTime — время в формате записи (усечение до миллисекунд, UTC).
func FormatTime(t time.Time) string { return t.UTC().Truncate(time.Millisecond).Format(TimeLayout) }

// ParseTime разбирает время записи (RFC 3339; источники могут прислать
// другую точность — заголовок хранит исходную строку, столбцы — момент).
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// ItemStream — поток изделия `item:‹item_id›` (AD-39).
func ItemStream(itemID string) string { return "item:" + itemID }

// StreamKind — вид потока: часть до первого «:» (`item`, `lot`, `policy`…);
// у `global` — сам `global`.
func StreamKind(stream string) (kind, id string) {
	k, rest, ok := strings.Cut(stream, ":")
	if !ok {
		return stream, ""
	}
	return k, rest
}

// Class — производные пометки записи по каталогу (AD-40): их хранит журнал в
// столбцах, чтобы проверки AD-39 и подача работы воркеру не толковали data.
type Class struct {
	// GuardRelevant — запись меняет версию своего потока для гардов (AD-39).
	GuardRelevant bool
	// Trigger — запись потока изделия запускает пересвёртку (AD-5): факт,
	// решение, адресованная запись межизделийной стадии; записи роли worker
	// (реакции самой свёртки) — не триггер (AD-3).
	Trigger bool
	// Known — тип есть в каталоге.
	Known bool
}

// Classify — пометки записи по каталогу. Неизвестный тип (например, версия
// каталога новее сборки) считается guard_relevant и триггером: лишняя
// пересвёртка и лишний 409 безопаснее пропущенных.
func Classify(e jc.JournalEntry) Class {
	info, ok := catalog.Lookup(catalog.Type(e.EventType))
	hasItem := e.ItemID != nil && *e.ItemID != ""
	if !ok {
		return Class{GuardRelevant: true, Trigger: hasItem}
	}
	return Class{GuardRelevant: info.GuardRelevant, Trigger: hasItem && info.Role != "worker", Known: true}
}

var (
	reEventType = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)
	reUUID      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	reSource    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]{0,127}$`)
)

// ErrInvalid — запись не проходит проверку заголовка перед записью.
var ErrInvalid = errors.New("journal: заголовок записи не соответствует entry.schema.json")

// ValidatePending проверяет открытые поля, которые ставит вызывающий
// (seq, chain, committed_at, commit и link ставит Append). Полная проверка
// data по схемам — у приёма (AD-20); здесь — то, без чего запись нельзя
// поставить в цепочку и найти.
func ValidatePending(e jc.JournalEntry) error {
	var errs []string
	if !reEventType.MatchString(e.EventType) {
		errs = append(errs, "event_type "+e.EventType)
	}
	if !reUUID.MatchString(e.EventID) {
		errs = append(errs, "event_id "+e.EventID)
	}
	if !reSource.MatchString(e.SourceID) {
		errs = append(errs, "source_id "+e.SourceID)
	}
	switch e.EntryKind {
	case jc.JournalEntryEntryKindFact, jc.JournalEntryEntryKindReaction, jc.JournalEntryEntryKindDecision, jc.JournalEntryEntryKindService:
	default:
		errs = append(errs, "entry_kind "+string(e.EntryKind))
	}
	switch e.ProvenanceClass {
	case jc.JournalEntryProvenanceClassDevice, jc.JournalEntryProvenanceClassPersonal, jc.JournalEntryProvenanceClassPaper,
		jc.JournalEntryProvenanceClassPartner, jc.JournalEntryProvenanceClassServerAttested, jc.JournalEntryProvenanceClassScenario,
		jc.JournalEntryProvenanceClassGenesis:
	default:
		errs = append(errs, "provenance_class "+string(e.ProvenanceClass))
	}
	if e.SchemaVersion < 1 || e.SchemaVersion > 999 {
		errs = append(errs, fmt.Sprintf("schema_version %d", e.SchemaVersion))
	}
	if e.Stream == "" {
		errs = append(errs, "stream пуст")
	}
	if e.Partition < 0 {
		errs = append(errs, "partition < 0")
	}
	if e.CorrelationID == "" {
		errs = append(errs, "correlation_id пуст")
	}
	if _, err := ParseDigest(e.DomainBuild); err != nil {
		errs = append(errs, "domain_build "+e.DomainBuild)
	}
	for _, f := range [...][2]string{{"occurred_at", e.OccurredAt}, {"received_at", e.ReceivedAt}} {
		if _, err := ParseTime(f[1]); err != nil {
			errs = append(errs, f[0]+" "+f[1])
		}
	}
	if e.RecordedAt != "" {
		if _, err := ParseTime(e.RecordedAt); err != nil {
			errs = append(errs, "recorded_at "+e.RecordedAt)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalid, strings.Join(errs, "; "))
	}
	return nil
}

// SealedPlain — пометка dek_id блока, который пока не зашифрован.
// TODO(29): шифрование при хранении (AD-23, DEK/KEK) — эпик 29; до него
// блок sealed несёт JCS(plain_block) открыто, base64 в ciphertext_b64.
const SealedPlain = "plain"

// PlainSealed — блок sealed демо-трека: plain_block {salt_b64, envelope}
// открыто (TODO(29)).
func PlainSealed(salt, envelope []byte) (jc.SealedBlock, error) {
	env, err := Canonical(envelope)
	if err != nil {
		return jc.SealedBlock{}, err
	}
	pb, err := json.Marshal(struct {
		SaltB64  string          `json:"salt_b64"`
		Envelope json.RawMessage `json:"envelope"`
	}{base64.StdEncoding.EncodeToString(salt), env})
	if err != nil {
		return jc.SealedBlock{}, err
	}
	pb, err = Canonical(pb)
	if err != nil {
		return jc.SealedBlock{}, err
	}
	return jc.SealedBlock{DekID: SealedPlain, CiphertextB64: base64.StdEncoding.EncodeToString(pb)}, nil
}

// ErrSealed — блок зашифрован, а ключа нет (или формат блока неизвестен).
var ErrSealed = errors.New("journal: блок записи зашифрован — нужен KEK (эпик 29)")

// OpenPlain раскрывает блок sealed демо-трека и сверяет commit (AD-23:
// после расшифрования commit проверяется всегда). Возвращает соль и JCS конверта.
func OpenPlain(e jc.JournalEntry) (salt, envelope []byte, err error) {
	if e.Sealed.DekID != SealedPlain {
		return nil, nil, ErrSealed
	}
	pb, err := base64.StdEncoding.DecodeString(e.Sealed.CiphertextB64)
	if err != nil {
		return nil, nil, fmt.Errorf("sealed: %w", err)
	}
	var blk struct {
		SaltB64  string          `json:"salt_b64"`
		Envelope json.RawMessage `json:"envelope"`
	}
	if err := json.Unmarshal(pb, &blk); err != nil {
		return nil, nil, fmt.Errorf("sealed: %w", err)
	}
	salt, err = base64.StdEncoding.DecodeString(blk.SaltB64)
	if err != nil {
		return nil, nil, fmt.Errorf("sealed: salt: %w", err)
	}
	envelope, err = Canonical(blk.Envelope)
	if err != nil {
		return nil, nil, err
	}
	if err := VerifyCommit(e, salt, envelope); err != nil {
		return nil, nil, err
	}
	return salt, envelope, nil
}
