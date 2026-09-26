package signing

import (
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// PRD §11.16: «годен» уровнем 1 не подписывается; факт исполнителя — можно.
func TestLevels(t *testing.T) {
	if err := CheckLevel("operation.run.started", 1, Level1, false); err != nil {
		t.Fatal(err)
	}
	if err := CheckLevel("inspection.result.accepted", 2, Level1, false); !errors.Is(err, ErrLevel) {
		t.Fatalf("«годен» уровнем 1: %v", err)
	}
	if err := CheckLevel("operation.run.started", 1, Level1, true); !errors.Is(err, ErrLevel) {
		t.Fatalf("критическое действие уровнем 1: %v", err)
	}
	if err := CheckLevel("decision.nonconformity.dispositioned", 2, Level2, true); err != nil {
		t.Fatal(err)
	}
	if err := CheckLevel("decision.nonconformity.dispositioned", 2, Level3, true); !errors.Is(err, ErrLevel) {
		t.Fatal("уровень 3 не для решений")
	}
}

func cmd(item, verdict string, level int) BatchItem {
	b, _ := CanonicalOf(map[string]any{"event_type": "inspection.result.recorded", "item_id": item,
		"command": map[string]any{"signature_level": level}, "data": map[string]any{"item_id": item, "outcome": verdict, "step_key": "weld.zt3"}})
	return BatchItem{Payload: b}
}

// AD-13: пачка уровня 2 — «подписать: N изделий, операция X, годен».
func TestBatch(t *testing.T) {
	s, err := Summarize([]BatchItem{cmd("E:1", "accept", 2), cmd("E:2", "accept", 2), cmd("E:3", "accept", 2)})
	if err != nil || s.Count != 3 || !slices.Equal(s.Items, []string{"E:1", "E:2", "E:3"}) {
		t.Fatalf("%+v %v", s, err)
	}
	if _, err := Summarize([]BatchItem{cmd("E:1", "accept", 2), cmd("E:2", "reject", 2)}); !errors.Is(err, ErrBatch) {
		t.Fatal("разные решения в одной пачке")
	}
	if _, err := Summarize([]BatchItem{cmd("E:1", "accept", 1)}); !errors.Is(err, ErrBatch) {
		t.Fatal("пачка уровня 1")
	}
	if _, err := Summarize([]BatchItem{cmd("E:1", "accept", 2), cmd("E:1", "accept", 2)}); !errors.Is(err, ErrBatch) {
		t.Fatal("изделие дважды")
	}
}

// AD-32: смена профиля вперёд — можно; понижение — отвергается; старые
// подписи — по профилю на момент подписи.
func TestProfileBook(t *testing.T) {
	b := NewProfileBook(nil)
	if b.Required(ClassEvent, 10) != ProfileGost || b.Required(ClassKeyAct, 10) != ProfileHybrid {
		t.Fatal("исходные профили")
	}
	up := ProfileChange{Profile: ProfileHybrid, Classes: []string{ClassEvent}, Seq: 100}
	if err := b.CheckChange(up); err != nil {
		t.Fatal(err)
	}
	b.Apply(up)
	if b.Required(ClassEvent, 99) != ProfileGost || b.Required(ClassEvent, 100) != ProfileHybrid {
		t.Fatal("профиль на момент подписи")
	}
	if err := b.CheckChange(ProfileChange{Profile: ProfileGost, Classes: []string{ClassEvent}, Seq: 200}); !errors.Is(err, ErrDowngrade) {
		t.Fatalf("понижение профиля: %v", err)
	}
	if err := b.CheckChange(ProfileChange{Profile: ProfilePQ, Classes: []string{ClassDocumentSignature}, Seq: 200}); !errors.Is(err, ErrDowngrade) {
		t.Fatalf("gost → pq теряет ГОСТ: %v", err)
	}
	if err := b.CheckChange(ProfileChange{Profile: ProfileHybrid, Classes: []string{ClassEvent}, Seq: 200, EffectiveFromSeq: 150}); !errors.Is(err, ErrProfile) {
		t.Fatalf("задним числом: %v", err)
	}
}

var t0 = time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)

func pub(n int, fill byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(rune(fill)), n)))
}

func reg(ref, kind, subj, profile string, seq int64, classes ...string) Registration {
	size := GostPublicKeySize
	if profile == ProfilePQ {
		size = MLDSAPublicKeySize
	}
	p := pub(size, ref[0])
	raw, _ := base64.StdEncoding.DecodeString(p)
	return Registration{KeyRef: ref, SubjectKind: kind, SubjectID: subj, ProfileID: profile, Algorithm: AlgorithmOf(profile),
		PublicKeyB64: p, Fingerprint: Digest(raw), PayloadClasses: classes, ValidFrom: t0, Provenance: NaturalProvenance(kind),
		Seq: seq, CommittedAt: t0.Add(time.Duration(seq) * time.Minute)}
}

func world() (*Registry, *ProfileBook) {
	r := NewRegistry()
	r.ApplyRegistration(reg("qc-01@1", SubjectPerson, "INS-01", ProfileGost, 1, ClassEvent, ClassPaperAttestation))
	r.ApplyRegistration(reg("qc-01-pq@1", SubjectPerson, "INS-01", ProfilePQ, 2, ClassEvent))
	r.ApplyRegistration(reg("dev-ws2@1", SubjectDevice, "ws-2", ProfileGost, 3, ClassEvent))
	return r, NewProfileBook(nil)
}

func judge(r *Registry, b *ProfileBook, profile string, signers, present []string, res map[string]Crypto, seq int64) Verdict {
	j := Judgement{Class: ClassEvent, Declared: Integrity{FormatVersion: 1, CryptoProfile: profile, Signers: signers}, Present: present,
		Seq: seq, At: t0.Add(time.Duration(seq) * time.Minute)}
	for _, k := range present {
		j.Crypto = append(j.Crypto, CryptoCheck{KeyRef: k, Result: res[k]})
	}
	return Judge(r, b, j)
}

// FR-68, FR-76, AD-9, AD-32: цело / изменённый пакет / понижение профиля /
// недоступный ключ — «не проверяемо», а не «валидно» / отзыв / компрометация.
func TestJudge(t *testing.T) {
	r, b := world()
	ok := map[string]Crypto{"qc-01@1": CryptoOK, "qc-01-pq@1": CryptoOK}
	if v := judge(r, b, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 10); v.Status != StatusValid {
		t.Fatalf("цело: %+v", v)
	}
	if v := judge(r, b, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, map[string]Crypto{"qc-01@1": CryptoBad}, 10); v.Status != StatusRejected || v.Reason != ReasonTampered {
		t.Fatalf("изменённый пакет: %+v", v)
	}
	if v := judge(r, b, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, map[string]Crypto{"qc-01@1": CryptoUnavailable}, 10); v.Status != StatusUnverifiable || v.Reason != ReasonUnavailable {
		t.Fatalf("недоступный ключ: %+v", v)
	}
	if v := judge(r, b, ProfileGost, []string{"nobody@1"}, []string{"nobody@1"}, map[string]Crypto{"nobody@1": CryptoOK}, 10); v.Status != StatusRejected || v.Reason != ReasonUnknownKey {
		t.Fatalf("незарегистрированный ключ: %+v", v)
	}
	if v := judge(r, b, ProfileGost, []string{"qc-01@1"}, nil, nil, 10); v.Status != StatusUnsigned {
		t.Fatalf("без подписей: %+v", v)
	}
	// Гибрид: обе подписи — цело; удалили ML-DSA — понижение профиля.
	both := []string{"qc-01@1", "qc-01-pq@1"}
	if v := judge(r, b, ProfileHybrid, both, both, ok, 10); v.Status != StatusValid {
		t.Fatalf("гибрид: %+v", v)
	}
	if v := judge(r, b, ProfileHybrid, both, []string{"qc-01@1"}, ok, 10); v.Status != StatusRejected || v.Reason != ReasonDowngrade {
		t.Fatalf("гибрид без второй подписи: %+v", v)
	}
	if v := judge(r, b, ProfileHybrid, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 10); v.Status != StatusRejected || v.Reason != ReasonDowngrade {
		t.Fatalf("гибрид одной подписью: %+v", v)
	}
	// Смена профиля событий на hybrid с seq 50: после смены gost — понижение, до — цело.
	b.Apply(ProfileChange{Profile: ProfileHybrid, Classes: []string{ClassEvent}, Seq: 50})
	if v := judge(r, b, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 49); v.Status != StatusValid {
		t.Fatalf("до смены профиля: %+v", v)
	}
	if v := judge(r, b, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 60); v.Status != StatusRejected || v.Reason != ReasonDowngrade {
		t.Fatalf("после смены профиля gost: %+v", v)
	}
	// Класс пакета не из акта ключа.
	j := Judgement{Class: ClassKeyAct, Declared: Integrity{CryptoProfile: ProfileHybrid, Signers: both}, Present: both,
		Crypto: []CryptoCheck{{"qc-01@1", CryptoOK}, {"qc-01-pq@1", CryptoOK}}, Seq: 10}
	if v := Judge(r, NewProfileBook(nil), j); v.Status != StatusRejected || v.Reason != ReasonClassForbidden {
		t.Fatalf("класс пакета: %+v", v)
	}
	// Отзыв с компрометацией с seq 20: подпись 25 — под сомнением, после отзыва (30) — отвергнуто.
	x := t0.Add(20 * time.Minute)
	r.ApplyRevocation(Revocation{KeyRef: "qc-01@1", CompromisedSince: &x, CompromisedSinceSeq: 20, ReasonText: "утерян токен", Seq: 30})
	b2 := NewProfileBook(nil)
	if v := judge(r, b2, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 15); v.Status != StatusValid {
		t.Fatalf("до компрометации: %+v", v)
	}
	if v := judge(r, b2, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 25); v.Status != StatusDoubtful {
		t.Fatalf("после «скомпрометирован с»: %+v", v)
	}
	if v := judge(r, b2, ProfileGost, []string{"qc-01@1"}, []string{"qc-01@1"}, ok, 31); v.Status != StatusRejected || v.Reason != ReasonRevoked {
		t.Fatalf("после отзыва: %+v", v)
	}
}

// AD-11: защитная реакция — решения после X до отзыва.
func TestProtectiveReaction(t *testing.T) {
	r, _ := world()
	x := t0.Add(20 * time.Minute)
	r.ApplyRevocation(Revocation{KeyRef: "qc-01@1", CompromisedSince: &x, CompromisedSinceSeq: 20, Seq: 30})
	k, _ := r.Key("qc-01@1")
	ds := []SignedDecision{
		{EventID: "a", ItemID: "E:1", Seq: 15, KeyRefs: []string{"qc-01@1"}},
		{EventID: "b", ItemID: "E:2", Seq: 22, KeyRefs: []string{"qc-01@1"}},
		{EventID: "c", ItemID: "E:3", Seq: 24, KeyRefs: []string{"other@1"}},
		{EventID: "d", ItemID: "E:4", Seq: 29, KeyRefs: []string{"qc-01@1"}},
		{EventID: "e", ItemID: "E:5", Seq: 31, KeyRefs: []string{"qc-01@1"}},
	}
	got := ProtectiveReaction(k, ds)
	if len(got) != 2 || got[0].EventID != "b" || got[1].EventID != "d" || !got[0].Contain || !got[0].Resign {
		t.Fatalf("%+v", got)
	}
	if s := CompromisedSinceSeq(x, []SeqTime{{19, t0.Add(19 * time.Minute)}, {20, x}, {21, x.Add(time.Minute)}}); s != 20 {
		t.Fatalf("X как seq: %d", s)
	}
}

func actFor(ref, subj, domain string) RegistrationAct {
	g := reg(ref, SubjectPerson, subj, ProfileGost, 0, ClassEvent)
	g.Provenance, g.SubjectConfirmation = "", ConfirmPaper
	return RegistrationAct{Registration: g, SubjectDomain: domain, Registrar: "ADM-01", ProofValid: true,
		ReceiptOriginalNo: "ОТК-2026-0001", ReceiptAttestedBy: "HQC-01"}
}

// AD-11, FR-70, FR-79: доказательство владения, подтверждение субъекта,
// вторая подпись независимой стороны, «выдача себе», повторный ключ, ротация.
func TestRegistrationRules(t *testing.T) {
	r, _ := world()
	a := actFor("qc-02@1", "INS-02", DomainQC)
	if _, _, err := ValidateRegistration(r, a, 10, t0); !errors.Is(err, ErrKeyAct) {
		t.Fatal("без второй подписи")
	}
	a.Approvals = []Approval{{PersonID: "PM-01", AuthorityID: AuthSecondProduction, Valid: true, Provenance: ProvPersonal}}
	if _, _, err := ValidateRegistration(r, a, 10, t0); !errors.Is(err, ErrKeyAct) {
		t.Fatal("ключ контролёра подписывает руководитель производства — не засчитывается")
	}
	a.Approvals = []Approval{{PersonID: "HQC-01", AuthorityID: AuthSecondQC, Valid: true, Provenance: ProvPersonal}}
	prov, alerts, err := ValidateRegistration(r, a, 10, t0)
	if err != nil || prov != ProvPersonal || len(alerts) != 0 {
		t.Fatalf("%s %v %v", prov, alerts, err)
	}
	// Выдача себе: субъект подписывает свою вторую подпись.
	self := actFor("hqc@1", "HQC-01", DomainQC)
	self.Approvals = []Approval{{PersonID: "HQC-01", AuthorityID: AuthSecondQC, Valid: true}}
	if _, _, err := ValidateRegistration(r, self, 10, t0); !errors.Is(err, ErrKeyAct) {
		t.Fatal("выдача себе")
	}
	// Без доказательства владения.
	nopop := a
	nopop.ProofValid = false
	if _, _, err := ValidateRegistration(r, nopop, 10, t0); !errors.Is(err, ErrKeyAct) {
		t.Fatal("без доказательства владения")
	}
	// Отпечаток не совпадает.
	bad := a
	bad.Fingerprint = Digest([]byte("x"))
	if _, _, err := ValidateRegistration(r, bad, 10, t0); !errors.Is(err, ErrKeyAct) {
		t.Fatal("отпечаток")
	}
	// Повторный ключ человека того же профиля — тревога.
	dup := actFor("qc-01-b@1", "INS-01", DomainQC)
	dup.Approvals = a.Approvals
	_, alerts, err = ValidateRegistration(r, dup, 10, t0)
	if err != nil || len(alerts) != 1 || alerts[0].Alert != "duplicate_key" {
		t.Fatalf("повторный ключ: %v %v", alerts, err)
	}
	// Ротация: подписью действующего ключа того же субъекта.
	rot := actFor("qc-01@2", "INS-01", DomainQC)
	rot.Rotates, rot.SubjectConfirmation, rot.Approvals = "qc-01@1", ConfirmRotation, a.Approvals
	if _, _, err := ValidateRegistration(r, rot, 10, t0); !errors.Is(err, ErrKeyAct) {
		t.Fatal("ротация без подписи действующим ключом")
	}
	rot.RotationValid = true
	if _, alerts, err := ValidateRegistration(r, rot, 10, t0); err != nil || len(alerts) != 0 {
		t.Fatalf("ротация: %v %v", alerts, err)
	}
	g := rot.Registration
	g.Seq = 11
	r.ApplyRegistration(g)
	if k, _ := r.Key("qc-01@1"); k.ActiveAt(12, t0) || !k.ActiveAt(10, t0) {
		t.Fatal("старый ключ после ротации")
	}
	// Класс доверия наследуется: зарегистрирован подписью scenario — scenario.
	if p := InheritProvenance(SubjectPerson, []string{ProvPersonal, ProvScenario}); p != ProvScenario {
		t.Fatal(p)
	}
	// Д-67: ключ, зарегистрированный генезисом, — класс по назначению ключа.
	if p := InheritProvenance(SubjectDevice, []string{ProvGenesis}); p != ProvDevice {
		t.Fatal(p)
	}
	if p := InheritProvenance(SubjectEngine, []string{ProvGenesis}); p != ProvServerAttested {
		t.Fatal(p)
	}
	if p := InheritProvenance(SubjectDemoPersona, []string{ProvGenesis}); p != ProvScenario {
		t.Fatal(p)
	}
	// Ключ устройства по акту ввода — вторая подпись руководителя производства.
	dev := reg("dev-ws3@1", SubjectDevice, "ws-3", ProfileGost, 0, ClassEvent)
	dev.SubjectConfirmation = ConfirmPaper
	da := RegistrationAct{Registration: dev, Registrar: "ADM-01", ProofValid: true,
		ReceiptOriginalNo: "АВ-12", ReceiptAttestedBy: "FOR-WC",
		Approvals: []Approval{{PersonID: "PM-01", AuthorityID: AuthSecondProduction, Valid: true, Provenance: ProvPersonal}}}
	if p, _, err := ValidateRegistration(r, da, 10, t0); err != nil || p != ProvDevice {
		t.Fatalf("акт ввода устройства: %s %v", p, err)
	}
}

// FR-139, AD-43: чужой QR, заверение самим подписантом, запрет бумаги.
func TestPaper(t *testing.T) {
	dg := Digest([]byte("документ"))
	e := PaperExpect{DocumentID: "DOC-7", DocDigest: dg, Stage: 1, Signer: "INS-01", PaperAllowed: true, AttesterAuthority: AuthPaperAttestation}
	a := PaperAttestation{DocumentID: "DOC-7", DocDigest: dg, Stage: 1, SignerPersonID: "INS-01", AttestedBy: "FOR-WC",
		ScanAddress: Digest([]byte("скан")), PaperOriginalNo: "ОТК-2026-0042", SignatureLevel: 2}
	f := PaperFacts{ScanQR: QRText("DOC-7", dg), ScanAddress: a.ScanAddress, AttesterHasAuthority: true}
	if err := CheckPaper(e, a, f); err != nil {
		t.Fatal(err)
	}
	foreign := f
	foreign.ScanQR = QRText("DOC-8", Digest([]byte("другой")))
	if err := CheckPaper(e, a, foreign); !errors.Is(err, ErrQR) {
		t.Fatalf("чужой QR: %v", err)
	}
	stale := f
	stale.ScanQR = QRText("DOC-7", Digest([]byte("прежняя версия")))
	if err := CheckPaper(e, a, stale); !errors.Is(err, ErrQR) {
		t.Fatalf("QR прежней версии: %v", err)
	}
	self := a
	self.AttestedBy = "INS-01"
	if err := CheckPaper(e, self, f); !errors.Is(err, ErrAttesterSigner) {
		t.Fatalf("заверитель = подписант: %v", err)
	}
	forb := e
	forb.PaperAllowed = false
	if err := CheckPaper(forb, a, f); !errors.Is(err, ErrPaperForbidden) {
		t.Fatalf("бумага запрещена: %v", err)
	}
	noNo := a
	noNo.PaperOriginalNo = " "
	if err := CheckPaper(e, noNo, f); !errors.Is(err, ErrPaperIncomplete) {
		t.Fatalf("без учётного номера: %v", err)
	}
	lvl := a
	lvl.SignatureLevel = 1
	if err := CheckPaper(e, lvl, f); !errors.Is(err, ErrLevel) {
		t.Fatalf("заверение уровнем 1: %v", err)
	}
}

// FR-139: решение, подписанное на бумаге, закрывает этап так же, как цифровое.
func TestPaperClosesStageLikeDigital(t *testing.T) {
	dg := Digest([]byte("решение"))
	rule := StageRule{Stage: 1, AuthorityID: "qc_acceptance", Need: 1, Level: 2, PaperAllowed: true}
	digital := CountedSignature{Stage: 1, DocDigest: dg, Method: MethodTokenAgent, SignerPersonID: "INS-01", AuthorityOK: true, Level: 2, Verdict: StatusValid}
	paper := CountedSignature{Stage: 1, DocDigest: dg, Method: MethodPaper, SignerPersonID: "INS-01", AuthorityOK: true, Level: 2,
		Verdict: StatusValid, AttestedBy: "FOR-WC", PaperOriginal: "ОТК-2026-0042"}
	if !StageClosed(rule, dg, []CountedSignature{digital}) || !StageClosed(rule, dg, []CountedSignature{paper}) {
		t.Fatal("бумага и цифра закрывают этап одинаково")
	}
	self := paper
	self.AttestedBy = "INS-01"
	if StageClosed(rule, dg, []CountedSignature{self}) {
		t.Fatal("заверение самим подписантом не засчитывается")
	}
	noPaper := rule
	noPaper.PaperAllowed = false
	if StageClosed(noPaper, dg, []CountedSignature{paper}) {
		t.Fatal("бумага на этапе запрещена")
	}
	if StageClosed(rule, Digest([]byte("новая версия")), []CountedSignature{paper}) {
		t.Fatal("подпись прежней версии не засчитывается")
	}
	two := StageRule{Stage: 1, Need: 2, Level: 2, PaperAllowed: true}
	other := digital
	other.SignerPersonID = "INS-02"
	if StageClosed(two, dg, []CountedSignature{digital, paper}) || !StageClosed(two, dg, []CountedSignature{paper, other}) {
		t.Fatal("k из n: разные подписанты")
	}
}

// AD-12: рапорт агента совпадает с журналом сервера; пропуск — расхождение.
func TestShiftReport(t *testing.T) {
	var ls []ShiftLeaf
	for i := range 5 {
		ls = append(ls, ShiftLeaf{Digest: Hash([]byte{byte(i)}), EventType: []string{"operation.run.started", "operation.run.finished"}[i%2], Seq: int64(10 + i)})
	}
	agent := BuildShiftReport(ShiftReport{PersonID: "O17", ShiftID: "S1"}, ls)
	if agent.LeafCount != 5 || agent.CountsByType["operation.run.started"] != 3 {
		t.Fatalf("%+v", agent)
	}
	rev := slices.Clone(ls)
	slices.Reverse(rev)
	if _, err := CompareShift(agent, rev); err != nil {
		t.Fatalf("тот же набор в другом порядке прихода: %v", err)
	}
	if d, err := CompareShift(agent, ls[:4]); !errors.Is(err, ErrShiftMismatch) || len(d.Types) != 1 {
		t.Fatalf("сервер не видит подписи: %+v %v", d, err)
	}
}
