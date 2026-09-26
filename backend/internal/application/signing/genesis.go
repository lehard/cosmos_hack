package signing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/signing"
)

// Блок генезиса доверия (AD-33, FR-10, FR-109; критерий Т1): сборка, запись и
// проверка. Состав фиксирован спайном: криптопрофили и формат цепочки, ключи
// (движок, шлюзы, хранитель, верификатор, устройства, демо-персоны, корни
// партнёра), код предприятия, режим часов, стартовая политика, справочники,
// нормативный слой v1 с подписями кворума ключами из этого же блока. Все
// записи — класса genesis, пакет каждой подписан ключом-якорем hybrid (и
// кворумом, где он нужен); после подписи якорь уничтожается — последняя
// запись блока «якорь уничтожен». Второй генезис — нарушение; повторный
// ant init ничего не делает (EnsureGenesis).

// GenesisRecord — запись блока генезиса между заголовком и «якорь уничтожен».
type GenesisRecord struct {
	Type catalog.Type
	// Stream — поток записи (`key:‹id›`, `policy:‹область›`, …); пусто — global.
	Stream string
	// Data — data записи по схеме contracts/events/‹семейство›/‹тип›.v1.json.
	Data any
	// Quorum — подписи кворума сверх якоря: key_ref ключей, зарегистрированных
	// в этом же блоке (нормативный слой v1, паспорта анализаторов; AD-33, FR-23).
	Quorum []string
}

// GenesisSpec — состав блока генезиса.
type GenesisSpec struct {
	EnterpriseCode string
	// NormativeVersionHash — хеш стартовой версии процесса (заголовок).
	NormativeVersionHash string
	// Anchor — открытые ключи якоря (anchor-gost@1, anchor-pq@1).
	Anchor []dom.AnchorKey
	// Records — записи по порядку (режим часов, профили, ключи, политика,
	// справочники, нормативный слой).
	Records []GenesisRecord
}

// GenesisConfig — параметры записей блока.
type GenesisConfig struct {
	DomainBuild string
	// Partitions — P движка: записи вне изделия — в партиции P (как у приёма).
	Partitions int
	// Now — InfraClock (received_at записей блока); nil — time.Now.
	Now func() time.Time
	// OccurredAt — occurred_at записей блока (доменное время начала действия
	// стартовой политики и справочников); ноль — Now.
	OccurredAt time.Time
}

// GenesisBlock — подписанный блок, готовый к записи.
type GenesisBlock struct {
	Pending []journal.Pending
	Header  dom.GenesisHeader
	// Digest — отпечаток генезиса (trust-anchors, первая контрольная точка).
	Digest string
}

// GenesisPayloadType — payloadType пакетов блока (класс genesis, AD-10).
var GenesisPayloadType = dom.PayloadType(dom.ClassGenesis, 1)

// GenesisEventID — event_id записи n блока с якорем anchorFP (UUIDv5 от NS_ANT):
// у каждого домена доверия свои, повтор той же записи — дубль.
func GenesisEventID(anchorFP string, n int) string {
	return kernel.UUIDv5(constants.NsAnt, "genesis\x1f"+anchorFP+"\x1f"+strconv.Itoa(n))
}

// BuildGenesis собирает и подписывает блок: заголовок (seq 1), записи spec,
// «якорь уничтожен» (seq k). Подпись — порт Signer ключами якоря и кворума
// (в демо — ключи в памяти ant init; якорь на диск не пишется).
func BuildGenesis(ctx context.Context, spec GenesisSpec, signer Signer, cfg GenesisConfig) (GenesisBlock, error) {
	if len(spec.Anchor) != len(dom.AnchorRefs) {
		return GenesisBlock{}, fmt.Errorf("генезис: ключей якоря %d, нужно %d", len(spec.Anchor), len(dom.AnchorRefs))
	}
	now := time.Now
	if cfg.Now != nil {
		now = cfg.Now
	}
	received := now().UTC().Truncate(time.Millisecond).Format(TimeLayout)
	at := received
	if !cfg.OccurredAt.IsZero() {
		at = cfg.OccurredAt.UTC().Truncate(time.Millisecond).Format(TimeLayout)
	}
	fp := dom.AnchorFingerprint(spec.Anchor)
	headerID := GenesisEventID(fp, 1)
	type item struct {
		rec  GenesisRecord
		id   string
		body []byte
	}
	items := make([]item, 0, len(spec.Records)+2)
	for _, r := range append(slices.Clone(spec.Records), GenesisRecord{Type: catalog.JournalAnchorDestroyed,
		Data: map[string]any{"anchor_fingerprint": fp}}) {
		items = append(items, item{rec: r, id: GenesisEventID(fp, len(items)+2)})
	}
	var payloads [][]byte
	for i := range items {
		b, err := genesisPayload(items[i].rec, items[i].id, headerID, at)
		if err != nil {
			return GenesisBlock{}, err
		}
		items[i].body = b
		payloads = append(payloads, b)
	}
	h := dom.GenesisHeader{EnterpriseCode: spec.EnterpriseCode, ChainFormatVersion: dom.ChainFormatVersion, AnchorFingerprint: fp,
		BlockSize: len(items) + 1, NormativeVersionHash: spec.NormativeVersionHash, AnchorKeys: slices.Clone(spec.Anchor),
		BlockDigest: dom.BlockDigest(payloads)}
	head, err := genesisPayload(GenesisRecord{Type: catalog.JournalGenesisRecorded, Data: h}, headerID, "", at)
	if err != nil {
		return GenesisBlock{}, err
	}
	all := append([]item{{rec: GenesisRecord{Type: catalog.JournalGenesisRecorded}, id: headerID, body: head}}, items...)
	out := GenesisBlock{Header: h, Digest: dom.GenesisDigest(head)}
	for _, it := range all {
		refs := append(slices.Clone(dom.AnchorRefs), it.rec.Quorum...)
		env, err := signer.Sign(ctx, GenesisPayloadType, it.body, strings.Join(refs, ","))
		if err != nil {
			return GenesisBlock{}, fmt.Errorf("генезис: подпись %s: %w", it.rec.Type, err)
		}
		info, _ := catalog.Lookup(it.rec.Type)
		e := jc.JournalEntry{Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKind(info.Kind), EventType: string(it.rec.Type),
			SchemaVersion: info.CurrentVersion, EventID: it.id, SourceID: dom.GenesisSource, Stream: streamOr(it.rec.Stream),
			Partition: cfg.Partitions, OccurredAt: at, ReceivedAt: received, CorrelationID: headerID,
			ProvenanceClass: jc.JournalEntryProvenanceClassGenesis, DomainBuild: cfg.DomainBuild}
		if it.id != headerID {
			c := headerID
			e.CausationID = &c
		}
		out.Pending = append(out.Pending, journal.Pending{Entry: e, Envelope: env})
	}
	return out, nil
}

func streamOr(s string) string {
	if s == "" {
		return "global"
	}
	return s
}

// genesisPayload — канонический конверт события записи блока (подписываемые
// байты, AD-10): integrity — профиль hybrid, подписанты — якорь и кворум.
func genesisPayload(r GenesisRecord, id, headerID, at string) ([]byte, error) {
	info, ok := catalog.Lookup(r.Type)
	if !ok {
		return nil, fmt.Errorf("генезис: тип %q не из каталога", r.Type)
	}
	if !slices.Contains(info.Provenance, dom.ProvGenesis) {
		return nil, fmt.Errorf("генезис: у типа %s нет происхождения genesis в каталоге", r.Type)
	}
	data, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}
	var cause any
	corr := id
	if headerID != "" {
		cause, corr = headerID, headerID
	}
	ev := map[string]any{
		"event_id": id, "event_type": string(r.Type), "schema_version": info.CurrentVersion, "source_id": dom.GenesisSource,
		"source_kind": "import", "occurred_at": at, "correlation_id": corr, "causation_id": cause,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": dom.ProfileHybrid,
			"signers": append(slices.Clone(dom.AnchorRefs), r.Quorum...)},
		"data": json.RawMessage(data),
	}
	// Блока command нет: записи генезиса — не команды человека (до seq 1
	// журнала нет ни basis_seq, ни policy_seq; AD-39).
	return dom.CanonicalOf(ev)
}

// Ошибки записи генезиса.
var (
	// ErrJournalNotEmpty — в журнале уже есть записи, а генезиса нет: блок
	// генезиса — только seq 1…k (AD-33); окружение нужно сбросить.
	ErrJournalNotEmpty = errors.New("генезис: журнал не пуст, а блока генезиса нет — сбросьте окружение (AD-33)")
)

// GenesisState — генезис в журнале: есть ли, сколько заголовков.
type GenesisState struct {
	Present bool
	Count   int
	// Header — запись заголовка (seq 1).
	Header jc.JournalEntry
}

// FindGenesis — есть ли в журнале блок генезиса (заголовки journal.genesis.recorded).
func FindGenesis(ctx context.Context, j journal.JournalStore) (GenesisState, error) {
	es, err := j.Read(ctx, journal.ReadQuery{EventType: string(catalog.JournalGenesisRecorded), Limit: 2})
	if err != nil {
		return GenesisState{}, err
	}
	st := GenesisState{Present: len(es) > 0, Count: len(es)}
	if st.Present {
		st.Header = es[0]
	}
	return st, nil
}

// EnsureGenesis — AD-33: генезис уже есть — ничего не делает (written = false);
// журнал пуст — пишет блок одной транзакцией и проверяет, что он лёг на
// seq 1…k; журнал не пуст без генезиса — ErrJournalNotEmpty. build
// вызывается, только если блок действительно нужно записать (ключи и якорь
// не создаются зря).
func EnsureGenesis(ctx context.Context, j journal.JournalStore, build func() (GenesisBlock, error)) (written bool, err error) {
	st, err := FindGenesis(ctx, j)
	if err != nil || st.Present {
		return false, err
	}
	any1, err := j.Read(ctx, journal.ReadQuery{Limit: 1})
	if err != nil {
		return false, err
	}
	if len(any1) > 0 {
		return false, ErrJournalNotEmpty
	}
	b, err := build()
	if err != nil {
		return false, err
	}
	_, err = j.Append(ctx, journal.AppendRequest{Batch: b.Pending, Project: func(_ context.Context, res journal.AppendResult) error {
		// Блок генезиса — только seq 1…k: кто-то записал раньше — откат.
		if len(res.Seqs) == 0 || res.Seqs[0] != 1 {
			return ErrJournalNotEmpty
		}
		return nil
	}})
	if errors.Is(err, journal.ErrDuplicate) {
		return false, nil
	}
	return err == nil, err
}

// RawVerifier — криптографическая проверка одной подписи открытым ключом
// (порт Crypto.VerifyRaw; адаптер — infrastructure/security/profiles.Verifier).
type RawVerifier interface {
	VerifyRaw(profile string, pub []byte, payloadType string, payload, sig []byte) bool
}

// GenesisReport — итог проверки блока генезиса.
type GenesisReport struct {
	Header dom.GenesisHeader
	// Digest — отпечаток генезиса (H(payload заголовка)).
	Digest string
	// Entries — записи блока seq 1…k.
	Entries []jc.JournalEntry
}

// VerifyGenesis — проверка блока генезиса в журнале «целиком по якорю»
// (AD-33, AD-9): подписи якоря и кворума каждой записи, отпечаток блока,
// закреплённый якорь pinned (из trust-anchors; пусто — без сверки), один
// генезис в журнале. Ключи кворума — только зарегистрированные в самом блоке.
func VerifyGenesis(ctx context.Context, j journal.JournalStore, v RawVerifier, pinned string) (GenesisReport, error) {
	return verifyGenesis(ctx, j, v, pinned, false)
}

// VerifyGenesisDigest — быстрая проверка блока (ядро при старте, повтор ant
// init): подписи заголовка, «якоря уничтожен» и кворума проверяются
// криптографически, остальные записи — через block_digest под подписью
// якоря (dom.GenesisCheck.DigestOnly).
func VerifyGenesisDigest(ctx context.Context, j journal.JournalStore, v RawVerifier, pinned string) (GenesisReport, error) {
	return verifyGenesis(ctx, j, v, pinned, true)
}

func verifyGenesis(ctx context.Context, j journal.JournalStore, v RawVerifier, pinned string, digestOnly bool) (GenesisReport, error) {
	st, err := FindGenesis(ctx, j)
	if err != nil {
		return GenesisReport{}, err
	}
	if !st.Present {
		return GenesisReport{}, fmt.Errorf("%w: генезиса в журнале нет (ant init)", dom.ErrGenesis)
	}
	first, err := j.Read(ctx, journal.ReadQuery{Limit: 1})
	if err != nil {
		return GenesisReport{}, err
	}
	if len(first) == 0 {
		return GenesisReport{}, fmt.Errorf("%w: журнал пуст", dom.ErrGenesis)
	}
	_, headPayload, err := openGenesis(ctx, j, first[0])
	if err != nil {
		return GenesisReport{}, err
	}
	h, err := dom.ParseGenesisHeader(headPayload)
	if err != nil {
		return GenesisReport{}, err
	}
	if h.BlockSize < 2 || h.BlockSize > 100000 {
		return GenesisReport{}, fmt.Errorf("%w: block_size %d", dom.ErrGenesis, h.BlockSize)
	}
	var entries []jc.JournalEntry
	for after := int64(0); len(entries) < h.BlockSize; {
		es, err := j.Read(ctx, journal.ReadQuery{AfterSeq: after, Limit: min(1000, h.BlockSize-len(entries))})
		if err != nil {
			return GenesisReport{}, err
		}
		if len(es) == 0 {
			break
		}
		entries = append(entries, es...)
		after = int64(es[len(es)-1].Seq)
	}
	pubs := map[string][2]any{}
	for _, k := range h.AnchorKeys {
		pubs[k.KeyRef] = [2]any{k.ProfileID, dom.AnchorKeyBytes(k)}
	}
	block := make([]dom.GenesisEntry, 0, len(entries))
	for _, e := range entries {
		env, payload, err := openGenesis(ctx, j, e)
		if err != nil {
			return GenesisReport{}, err
		}
		ge := dom.GenesisEntry{Seq: int64(e.Seq), EventType: e.EventType, Provenance: string(e.ProvenanceClass),
			PayloadType: env.PayloadType, Payload: payload}
		// Быстрая проверка: только заголовок, последняя запись и подписи сверх якоря (кворум).
		skip := digestOnly && e.Seq != 1 && e.Seq != h.BlockSize && len(env.Signatures) <= len(dom.AnchorRefs)
		for _, s := range env.Signatures {
			ge.Present = append(ge.Present, s.KeyID)
			if skip {
				continue
			}
			c := dom.CryptoCheck{KeyRef: s.KeyID, Result: dom.CryptoUnavailable}
			if k, ok := pubs[s.KeyID]; ok {
				sig, _ := base64.StdEncoding.DecodeString(s.Sig)
				c.Result = dom.CryptoBad
				if v.VerifyRaw(k[0].(string), k[1].([]byte), env.PayloadType, payload, sig) {
					c.Result = dom.CryptoOK
				}
			}
			ge.Crypto = append(ge.Crypto, c)
		}
		block = append(block, ge)
		// Ключи, зарегистрированные блоком, подписывают кворум в последующих записях.
		if e.EventType == string(catalog.KeyRegistrationRecorded) {
			if d, err := EventData(env.Marshal()); err == nil {
				if g, err := RegistrationFromData(d); err == nil {
					pubs[g.KeyRef] = [2]any{g.ProfileID, g.PublicKey()}
				}
			}
		}
	}
	h, err = dom.CheckGenesisBlock(dom.GenesisCheck{Block: block, Pinned: pinned, GenesisCount: st.Count, DigestOnly: digestOnly})
	if err != nil {
		return GenesisReport{}, err
	}
	return GenesisReport{Header: h, Digest: dom.GenesisDigest(headPayload), Entries: entries}, nil
}

// openGenesis — конверт DSSE записи и подписанное содержимое.
func openGenesis(ctx context.Context, j journal.JournalStore, e jc.JournalEntry) (dom.Envelope, []byte, error) {
	env, err := j.Open(ctx, e)
	if err != nil {
		return dom.Envelope{}, nil, fmt.Errorf("%w: seq %d: конверт: %v", dom.ErrGenesis, e.Seq, err)
	}
	d, payload, err := dom.ParseEnvelope(env.Raw)
	if err != nil {
		return dom.Envelope{}, nil, fmt.Errorf("%w: seq %d: %v", dom.ErrGenesis, e.Seq, err)
	}
	return d, payload, nil
}

// GenesisData — data записей типа t внутри блока генезиса (по порядку seq):
// так модули берут из генезиса стартовый нормативный слой, паспорта и
// политику, а не из встроенной копии.
func GenesisData(ctx context.Context, j journal.JournalStore, rep GenesisReport, t catalog.Type) ([]json.RawMessage, error) {
	var out []json.RawMessage
	for _, e := range rep.Entries {
		if e.EventType != string(t) {
			continue
		}
		env, err := j.Open(ctx, e)
		if err != nil {
			return nil, err
		}
		d, err := EventData(env.Raw)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}
