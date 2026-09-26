package federation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"uuid"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	appmaterials "ant/internal/application/materials"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/federation"
	"ant/internal/domain/kernel"
	"ant/internal/domain/signing"
)

// Service — реализация live ведущих портов модуля federation (AD-36, AD-19;
// FR-131, FR-132): партнёры и выписки читаются из журнала по типу записи,
// регистрация партнёра — решение в журнале, приём выписки — самостоятельная
// проверка подписей и запись federation.extract.received, отправка —
// подписанный ключом шлюза пакет в хранилище материалов и запись
// federation.message.sent (отправляет роль outbox). Без зависимостей (Deps
// пуст) — прежняя заглушка 501.
type Service struct {
	Unimplemented
	d    Deps
	live bool
}

// Deps — ведомые порты live-реализации.
type Deps struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
	// Materials — хранилище пакетов выписок (AD-23); nil — пакет только в записи не хранится.
	Materials appmaterials.MaterialStore
	// Crypto — проверка одной подписи (profiles.Verify).
	Crypto dom.CryptoFunc
	// Sign — подпись пакета ключом шлюза предприятия (уровень 0); nil — исходящая
	// выписка без подписи шлюза (подписи людей — маршрутом документа).
	Sign func(ctx context.Context, payloadType string, payload []byte) ([]byte, error)
	// Enterprise — наш код предприятия (глобальные ID «код:локальный_id»).
	Enterprise string
	// Passport — содержимое выписки по изделию или партии (паспорт, эпик 18);
	// nil — только предмет и ссылки на документ.
	Passport func(ctx context.Context, subject platform.DrillRef) (dom.Extract, error)
	Now      func() time.Time
	Log      *slog.Logger
	// DomainBuild, Partitions — как у прочих модулей (записи решений).
	DomainBuild string
	Partitions  int
}

// NewService создаёт реализацию live; без аргумента — заглушка 501.
func NewService(deps ...Deps) *Service {
	s := &Service{}
	if len(deps) > 0 && deps[0].Journal != nil && deps[0].Codec != nil {
		s.d, s.live = deps[0], true
		if s.d.Now == nil {
			s.d.Now = time.Now
		}
		if s.d.Log == nil {
			s.d.Log = slog.Default()
		}
	}
	return s
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// SourceAPI — source_id решений операций API.
const SourceAPI = "ant-api"

func partnerStream(code string) string { return "partner:" + code }

// records — записи типа t (все страницы).
func (s *Service) records(ctx context.Context, t catalog.Type) ([]engineapp.Decoded, error) {
	var out []engineapp.Decoded
	after := int64(0)
	for {
		page, err := s.d.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(t), AfterSeq: after, Limit: 500})
		if err != nil {
			return nil, err
		}
		for _, e := range page {
			d, err := s.d.Codec.Decode(ctx, e)
			if err != nil {
				return nil, err
			}
			out = append(out, d)
		}
		if len(page) < 500 {
			return out, nil
		}
		after = int64(page[len(page)-1].Seq)
	}
}

// visible — запись видна на момент m (AD-22): по времени записи в журнал.
func visible(d engineapp.Decoded, m platform.Moment) bool {
	return m.AsOf == nil || !d.Record.RecordedAt.After(*m.AsOf)
}

// partners — действующие партнёры: последний акт по каждому коду.
func (s *Service) partners(ctx context.Context, m platform.Moment) ([]Partner, error) {
	ds, err := s.records(ctx, catalog.FederationPartnerRegistered)
	if err != nil {
		return nil, err
	}
	last := map[string]Partner{}
	for _, d := range ds {
		if !visible(d, m) {
			continue
		}
		var x ev.FederationPartnerRegisteredV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, err
		}
		p := Partner{PartnerCode: x.PartnerCode, Name: x.Name, DocumentID: string(x.DocumentID), RegisteredAt: d.Record.RecordedAt, Channel: "unknown"}
		for _, f := range x.RootFingerprints {
			p.RootFingerprints = append(p.RootFingerprints, string(f))
		}
		if x.Endpoint != nil {
			p.Endpoint = *x.Endpoint
		}
		last[p.PartnerCode] = p
	}
	out := make([]Partner, 0, len(last))
	for _, p := range last {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PartnerCode < out[j].PartnerCode })
	return out, nil
}

// Partners — предприятия-партнёры (federation.partner.list).
func (s *Service) Partners(ctx context.Context, m platform.Moment) (PartnerList, error) {
	if !s.live {
		return s.Unimplemented.Partners(ctx, m)
	}
	ps, err := s.partners(ctx, m)
	return PartnerList{Items: ps}, err
}

// extracts — входящие и исходящие выписки из журнала.
func (s *Service) extracts(ctx context.Context, m platform.Moment) ([]PassportExtract, error) {
	var out []PassportExtract
	in, err := s.records(ctx, catalog.FederationExtractReceived)
	if err != nil {
		return nil, err
	}
	for _, d := range in {
		if !visible(d, m) {
			continue
		}
		var x ev.FederationExtractReceivedV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, err
		}
		e := PassportExtract{ExtractDigest: string(x.ExtractDigest), Direction: "incoming", PartnerCode: x.PartnerCode,
			OriginStatus: string(x.OriginStatus), MaterialAddress: string(x.MaterialAddress), At: d.Record.RecordedAt}
		if x.HeatNo != nil {
			e.HeatNo = *x.HeatNo
		}
		if x.LotID != nil {
			e.Subject = &platform.DrillRef{Entity: platform.EntityLot, ID: string(*x.LotID)}
			e.Label = string(*x.LotID)
		}
		out = append(out, e)
	}
	acks := map[string]bool{}
	ak, err := s.records(ctx, catalog.FederationMessageAcknowledged)
	if err != nil {
		return nil, err
	}
	for _, d := range ak {
		var x ev.FederationMessageAcknowledgedV1
		if json.Unmarshal(d.Record.Data, &x) == nil && visible(d, m) {
			acks[string(x.MessageID)] = x.Outcome == ev.FederationMessageAcknowledgedV1OutcomeReceived
		}
	}
	sent, err := s.records(ctx, catalog.FederationMessageSent)
	if err != nil {
		return nil, err
	}
	for _, d := range sent {
		var x ev.FederationMessageSentV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, err
		}
		if !visible(d, m) || x.Kind != ev.FederationMessageSentV1KindPassportExtract || x.ExtractDigest == nil {
			continue
		}
		e := PassportExtract{ExtractDigest: string(*x.ExtractDigest), Direction: "outgoing", PartnerCode: x.PartnerCode,
			OriginStatus: dom.OriginNotApplicable, MessageID: string(x.MessageID), At: d.Record.RecordedAt}
		if x.MaterialAddress != nil {
			e.MaterialAddress = string(*x.MaterialAddress)
		}
		if x.DocumentID != nil {
			e.DocumentID = string(*x.DocumentID)
		}
		if x.SubjectID != nil {
			e.Subject = &platform.DrillRef{Entity: platform.EntityItem, ID: *x.SubjectID}
			e.Label = *x.SubjectID
		}
		if a, ok := acks[e.MessageID]; ok {
			e.Acknowledged = &a
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, nil
}

// Extracts — выписки (federation.extract.list).
func (s *Service) Extracts(ctx context.Context, direction, partnerCode string, m platform.Moment, p platform.Page) (PassportExtractList, error) {
	if !s.live {
		return s.Unimplemented.Extracts(ctx, direction, partnerCode, m, p)
	}
	all, err := s.extracts(ctx, m)
	if err != nil {
		return PassportExtractList{}, err
	}
	return PassportExtractList{Items: Filter(all, direction, partnerCode)}, nil
}

// Filter — выписки по направлению и партнёру (пусто — все).
func Filter(all []PassportExtract, direction, partnerCode string) []PassportExtract {
	out := []PassportExtract{}
	for _, e := range all {
		if (direction == "" || e.Direction == direction) && (partnerCode == "" || e.PartnerCode == partnerCode) {
			out = append(out, e)
		}
	}
	return out
}

// Extract — выписка с подписями (federation.extract.read): пакет из хранилища
// материалов проверяется заново корнями партнёра на сейчас.
func (s *Service) Extract(ctx context.Context, digest string) (PassportExtractView, error) {
	if !s.live {
		return s.Unimplemented.Extract(ctx, digest)
	}
	all, err := s.extracts(ctx, platform.Moment{})
	if err != nil {
		return PassportExtractView{}, err
	}
	i := slices.IndexFunc(all, func(e PassportExtract) bool { return e.ExtractDigest == digest })
	if i < 0 {
		return PassportExtractView{}, platform.Fail(errcodes.ApiNotFound, "object", "Выписка паспорта", "id", digest)
	}
	e := all[i]
	raw, err := s.material(ctx, e.MaterialAddress)
	if err != nil || raw == nil {
		return PassportExtractView{Extract: e, Content: map[string]any{}, Signatures: []PartnerSignature{},
			OriginReason: "пакет выписки недоступен в хранилище материалов"}, nil
	}
	ps, err := s.partners(ctx, platform.Moment{})
	if err != nil {
		return PassportExtractView{}, err
	}
	partner, roots := e.PartnerCode, RootsOf(ps, e.PartnerCode)
	if e.Direction == "outgoing" {
		partner, roots = "", nil
	}
	v, verr := dom.VerifyExtract(raw, partner, roots, s.d.Crypto)
	if verr != nil && e.Direction == "incoming" {
		v.Reason = verr.Error()
	}
	view := ViewOf(v, raw, e.Direction, e.PartnerCode, e.MaterialAddress, e.At, e.Subject)
	view.Extract.MessageID, view.Extract.Acknowledged, view.Extract.DocumentID = e.MessageID, e.Acknowledged, e.DocumentID
	if e.Direction == "incoming" {
		view.Extract.OriginStatus = e.OriginStatus // статус на момент приёма (запись журнала)
	}
	return view, nil
}

// RootsOf — отпечатки корней партнёра code из акта регистрации.
func RootsOf(ps []Partner, code string) []string {
	for _, p := range ps {
		if p.PartnerCode == code {
			return p.RootFingerprints
		}
	}
	return nil
}

func (s *Service) material(ctx context.Context, addr string) ([]byte, error) {
	if s.d.Materials == nil || addr == "" {
		return nil, nil
	}
	r, _, err := s.d.Materials.Get(ctx, addr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func (s *Service) put(ctx context.Context, raw []byte) (string, error) {
	if s.d.Materials == nil {
		return signing.Digest(raw), nil
	}
	return s.d.Materials.Put(ctx, bytes.NewReader(raw), appmaterials.Meta{ContentType: "application/json", Size: int64(len(raw)), Kind: "attachment"})
}

// RegisterPartner — акт регистрации партнёра и его корней (AD-19, AD-11).
func (s *Service) RegisterPartner(ctx context.Context, in RegisterPartner) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.RegisterPartner(ctx, in)
	}
	if err := CheckRegister(in); err != nil {
		return platform.Receipt{}, err
	}
	data := ev.FederationPartnerRegisteredV1{PartnerCode: in.PartnerCode, Name: strings.TrimSpace(in.Name), DocumentID: ev.ObjectID(in.DocumentID)}
	for _, f := range in.RootFingerprints {
		data.RootFingerprints = append(data.RootFingerprints, ev.Digest(f))
	}
	if in.Endpoint != "" {
		ep := in.Endpoint
		data.Endpoint = &ep
	}
	return s.decide(ctx, catalog.FederationPartnerRegistered, partnerStream(in.PartnerCode), data, in.CommandMeta())
}

// CheckRegister — гард акта регистрации: отпечатки корней — streebog256, без повторов.
func CheckRegister(in RegisterPartner) error {
	seen := map[string]bool{}
	for _, f := range in.RootFingerprints {
		if _, err := signing.ParseDigest(f); err != nil || seen[f] {
			e := platform.Fail(errcodes.ApiValidationFailed, "field", "root_fingerprints")
			e.Detail = "Отпечаток корня партнёра — streebog256:‹64 hex›, без повторов: " + f
			return e
		}
		seen[f] = true
	}
	if strings.TrimSpace(in.DocumentID) == "" {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "document_id")
		e.Detail = "Нужен акт регистрации партнёра (документ с маршрутом «администратор безопасности → начальник ОТК»)"
		return e
	}
	return nil
}

// ReceiveExtract — принять выписку партнёра (FR-132): проверка подписей
// корнями партнёра; изменённая — 422; принятая — federation.extract.received.
func (s *Service) ReceiveExtract(ctx context.Context, in ReceiveExtract) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.ReceiveExtract(ctx, in)
	}
	ps, err := s.partners(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	partner := in.PartnerCode
	if RootsOf(ps, partner) == nil {
		partner = "" // не зарегистрирован — «происхождение не подтверждено»
	}
	raw := []byte(in.Envelope)
	v, err := dom.VerifyExtract(raw, partner, RootsOf(ps, in.PartnerCode), s.d.Crypto)
	if err != nil {
		s.d.Log.Warn("federation: выписка отклонена", "partner", in.PartnerCode, "err", err)
		return platform.Receipt{}, RejectTampered(err)
	}
	addr, err := s.put(ctx, raw)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := ev.FederationExtractReceivedV1{PartnerCode: in.PartnerCode, ExtractDigest: ev.Digest(v.Digest),
		OriginStatus: ev.FederationExtractReceivedV1OriginStatus(v.Origin), MaterialAddress: ev.Digest(addr)}
	if h := v.Extract.Origin.HeatNo; h != "" {
		data.HeatNo = &h
	}
	if sub := SubjectOf(v.Extract, "incoming"); sub != nil && sub.Entity == platform.EntityLot {
		l := ev.ObjectID(sub.ID)
		data.LotID = &l
	}
	// Идемпотентность приёма (AD-7): ключ — партнёр + отпечаток выписки.
	id := kernel.UUIDv5(constants.NsAnt, string(catalog.FederationExtractReceived)+"\x1f"+in.PartnerCode+"\x1f"+v.Digest)
	return s.record(ctx, engineapp.Out{EventID: id, Type: catalog.FederationExtractReceived, Kind: catalog.KindFact,
		Stream: partnerStream(in.PartnerCode), OccurredAt: s.d.Now().UTC(), RunID: appjournal.RunFrom(ctx), Data: data})
}

// SendExtract — отправить выписку паспорта партнёру (FR-131): содержимое —
// паспорт предмета (порт Passport), подпись ключом шлюза; пакет — в
// хранилище материалов; запись federation.message.sent забирает роль outbox.
func (s *Service) SendExtract(ctx context.Context, in SendExtract) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.SendExtract(ctx, in)
	}
	ps, err := s.partners(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if RootsOf(ps, in.PartnerCode) == nil {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "Партнёр", "id", in.PartnerCode)
	}
	data := ev.FederationMessageSentV1{PartnerCode: in.PartnerCode, Kind: ev.FederationMessageSentV1Kind(in.Kind)}
	if in.DocumentID != "" {
		doc := ev.ObjectID(in.DocumentID)
		data.DocumentID = &doc
	}
	sid := in.Subject.ID
	data.SubjectID = &sid
	if in.Kind == "passport_extract" {
		raw, digest, err := s.buildExtract(ctx, in)
		if err != nil {
			return platform.Receipt{}, err
		}
		addr, err := s.put(ctx, raw)
		if err != nil {
			return platform.Receipt{}, err
		}
		d, a := ev.Digest(digest), ev.Digest(addr)
		data.ExtractDigest, data.MaterialAddress = &d, &a
	}
	key := strings.ToLower(in.CommandID)
	if key == "" {
		key = s.d.Now().UTC().Format(time.RFC3339Nano)
	}
	id := kernel.UUIDv5(constants.NsAnt, string(catalog.FederationMessageSent)+"\x1f"+in.PartnerCode+"\x1f"+key)
	data.MessageID = ev.UUID(id)
	return s.record(ctx, engineapp.Out{EventID: id, Type: catalog.FederationMessageSent, Kind: catalog.KindService,
		Stream: partnerStream(in.PartnerCode), OccurredAt: s.d.Now().UTC(), RunID: appjournal.RunFrom(ctx), Data: data})
}

// buildExtract — пакет исходящей выписки: канонический JSON и подпись шлюза.
func (s *Service) buildExtract(ctx context.Context, in SendExtract) ([]byte, string, error) {
	gid := in.Subject.ID
	if !strings.Contains(gid, ":") {
		gid = dom.GlobalID(s.d.Enterprise, gid)
	}
	x := dom.Extract{Subject: dom.Subject{Kind: string(in.Subject.Entity), GlobalID: gid}}
	if s.d.Passport != nil {
		p, err := s.d.Passport(ctx, in.Subject)
		if err != nil {
			return nil, "", err
		}
		x = p
	}
	x.FormatVersion, x.CryptoProfile, x.Sender, x.Recipient = 1, "gost", s.d.Enterprise, in.PartnerCode
	x.IssuedAt = s.d.Now().UTC().Format(time.RFC3339)
	if x.ExtractNo == "" {
		x.ExtractNo = s.d.Enterprise + "-" + in.Subject.ID
	}
	if in.DocumentID != "" {
		x.Sources = append(x.Sources, "document:"+in.DocumentID)
	}
	payload, err := signing.CanonicalOf(x)
	if err != nil {
		return nil, "", err
	}
	raw := signing.Seal(dom.ExtractPayloadType, payload).Marshal()
	if s.d.Sign != nil {
		if raw, err = s.d.Sign(ctx, dom.ExtractPayloadType, payload); err != nil {
			return nil, "", err
		}
	}
	return raw, dom.ExtractDigest(payload), nil
}

// record — служебная запись или факт модуля ключом движка (Codec, AD-10).
func (s *Service) record(ctx context.Context, o engineapp.Out) (platform.Receipt, error) {
	pend, err := s.d.Codec.Encode(ctx, o)
	if err != nil {
		return platform.Receipt{}, err
	}
	r, err := s.d.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pend}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return platform.Receipt{CommandID: o.EventID, EventIDs: []string{o.EventID}, Replayed: true}, nil
	}
	if err != nil {
		return platform.Receipt{}, err
	}
	s.d.Log.Info("federation: запись", "event_type", string(o.Type), "event_id", o.EventID)
	rc := platform.Receipt{CommandID: o.EventID, EventIDs: []string{o.EventID}, RecordedAt: r.Committed}
	if len(r.Seqs) > 0 {
		rc.Seq = r.Seqs[0]
	}
	return rc, nil
}

// decide — решение человека операцией API (как у ops): конверт без подписи
// сервера (подписанный пакет команды — в блоке command, Д-30), source ant-api.
func (s *Service) decide(ctx context.Context, t catalog.Type, stream string, payload any, meta platform.CommandMeta) (platform.Receipt, error) {
	info, _ := catalog.Lookup(t)
	id := strings.ToLower(meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := strings.ToLower(platform.PrincipalFrom(ctx).PersonID)
	if actor == "" {
		actor = "anonymous"
	}
	occurred := engineapp.FormatTime(s.d.Now())
	run := appjournal.RunFrom(ctx)
	env := map[string]any{
		"event_id": id, "event_type": string(t), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"occurred_at": occurred, "correlation_id": id, "causation_id": nil,
		"command":   map[string]any{"command_id": id, "basis_seq": meta.BasisSeq, "guard_streams": []string{stream}, "policy_seq": meta.PolicySeq},
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{actor + "@1"}},
		"data":      json.RawMessage(data),
	}
	if run != "" {
		env["run_id"] = run
	}
	canon, err := engine.Canonical(env)
	if err != nil {
		return platform.Receipt{}, err
	}
	sealed, _ := json.Marshal(struct {
		PayloadType string   `json:"payloadType"`
		Payload     string   `json:"payload"`
		Signatures  []string `json:"signatures"`
	}{engineapp.PayloadTypeEvent, base64.StdEncoding.EncodeToString(canon), []string{}})
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(t),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(s.d.Now()), CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: s.d.DomainBuild,
	}
	if run != "" {
		e.RunID = &run
	}
	res, err := s.d.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return platform.Receipt{CommandID: id, EventIDs: []string{id}, Replayed: true}, nil
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	s.d.Log.Info("federation: решение записано", "event_id", id, "event_type", string(t))
	rc := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: res.Committed}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[0]
	}
	return rc, nil
}
