package signing

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"
	"uuid"

	ingest "ant/internal/application/ingest"
	"ant/internal/application/platform"
	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// FR-66, FR-68, FR-69, AD-11: подпись решения агентом токена принимается;
// изменённый пакет, чужой ключ, «годен» уровнем 1, недоступный ключ — нет.
func TestCommandSignature(t *testing.T) {
	w := newWorld(t, false)
	ctx := as("INS-01")
	ce, exp := decision("ENT01:F-001", 2, dom.ProfileGost, "ins-01@1")
	a, err := w.svc.CheckCommand(ctx, w.signCmd(ce, "ins-01@1"), exp)
	if err != nil || a.Status != dom.StatusValid || a.Method != dom.MethodTokenAgent || a.Provenance != dom.ProvPersonal || a.SignerPersonID != "INS-01" {
		t.Fatalf("цифровая подпись: %+v %v", a, err)
	}
	// Изменённый пакет: подпись над другим содержимым.
	raw := w.signCmd(ce, "ins-01@1")
	var env dom.Envelope
	_ = json.Unmarshal(raw, &env)
	other := ce
	other.Data = json.RawMessage(`{"inspection_point":"weld.zt3","item_id":"ENT01:F-001","outcome":"reject"}`)
	c, _ := other.Canonical()
	env.Payload = b64(c)
	if _, err := w.svc.CheckCommand(ctx, env.Marshal(), exp); code(err) != string(errcodes.SigningPackageTampered) {
		t.Fatalf("изменённый пакет: %v", err)
	}
	// Подписано верно, но не то, что исполняет команда.
	if _, err := w.svc.CheckCommand(ctx, w.signCmd(other, "ins-01@1"), exp); code(err) != string(errcodes.SigningDocumentChanged) {
		t.Fatalf("подписано другое решение: %v", err)
	}
	// Чужой ключ: мастер подписывает своим ключом решение контролёра → отказ и тревога.
	ce2, exp2 := decision("ENT01:F-002", 2, dom.ProfileGost, "for-wc@1")
	if _, err := w.svc.CheckCommand(ctx, w.signCmd(ce2, "for-wc@1"), exp2); code(err) != string(errcodes.SigningForeignKey) {
		t.Fatalf("чужой ключ: %v", err)
	}
	if !slices.Contains(w.alerts(), "foreign_key") {
		t.Fatal("нет тревоги «чужой ключ»")
	}
	// «Годен» уровнем 1 не подписывается (PRD §11.16).
	ce3, exp3 := decision("ENT01:F-003", 1, dom.ProfileGost, "ins-01@1")
	if _, err := w.svc.CheckCommand(ctx, w.signCmd(ce3, "ins-01@1"), exp3); code(err) != string(errcodes.SigningLevelNotAllowed) {
		t.Fatalf("уровень 1: %v", err)
	}
	// Недоступный ключ — «не проверяемо», а не «валидно» (AD-32, FR-76).
	ce4, exp4 := decision("ENT01:F-004", 2, dom.ProfileGost, "lost@1")
	raw4 := w.signCmd(ce4, "lost@1")
	if _, err := w.svc.CheckCommand(ctx, raw4, exp4); code(err) != string(errcodes.SigningKeyUnavailable) {
		t.Fatalf("недоступный ключ: %v", err)
	}
	v, _, err := w.svc.Judge(context.Background(), raw4)
	if err != nil || v.Status != dom.StatusUnverifiable {
		t.Fatalf("недоступный ключ: %+v %v", v, err)
	}
	// Без подписи: в prod — нет пути подписи.
	if _, err := w.svc.CheckCommand(ctx, nil, exp); code(err) != string(errcodes.SigningNoSignaturePath) {
		t.Fatalf("без подписи в prod: %v", err)
	}
}

// Д-28, Д-30: в профиле demo неподписанное допустимо с пометкой.
func TestDemoUnsigned(t *testing.T) {
	w := newWorld(t, true)
	_, exp := decision("ENT01:F-001", 2, dom.ProfileGost, "ins-01@1")
	a, err := w.svc.CheckCommand(as("INS-01"), nil, exp)
	if err != nil || a.Status != dom.StatusUnsigned || a.Note != app.NotChecked {
		t.Fatalf("%+v %v", a, err)
	}
}

// FR-139, AD-43: бумага. Лист решения печатается с QR, подписант ставит
// подпись ручкой, скан заверяет второй человек подписью уровня 2. Бумага
// закрывает точку так же, как цифровая подпись; чужой QR и заверение самим
// подписантом не принимаются.
func TestPaperPath(t *testing.T) {
	w := newWorld(t, false)
	_, exp := decision("ENT01:F-010", 2, dom.ProfileGost)
	docID, digest, sheet, err := app.Sheet(exp)
	if err != nil || docID != exp.CommandID || len(sheet) == 0 {
		t.Fatal(err)
	}
	scan := []byte("QR:" + dom.QRText(docID, digest))
	addr := dom.Digest(scan)
	w.scans[addr] = scan
	attest := func(attester, signer, scanAddr string) []byte {
		pa := dom.PaperAttestation{FormatVersion: 1, CryptoProfile: dom.ProfileGost, Signers: []string{app.SignerKey(attester)},
			DocumentID: docID, DocDigest: digest, Stage: 1, SignerPersonID: signer, AttestedBy: attester, ScanAddress: scanAddr,
			PaperOriginalNo: "ОТК-2026-0042", SignatureLevel: 2}
		c, _ := dom.CanonicalOf(pa)
		return w.sign(dom.ClassPaperAttestation, c, app.SignerKey(attester))
	}
	exp.Actor = "FOR-WC"
	a, err := w.svc.CheckCommand(as("FOR-WC"), attest("FOR-WC", "INS-01", addr), exp)
	if err != nil || a.Method != dom.MethodPaper || a.Provenance != dom.ProvPaper || a.SignerPersonID != "INS-01" ||
		a.AttestedBy != "FOR-WC" || a.PaperOriginalNo != "ОТК-2026-0042" || a.ScanAddress != addr {
		t.Fatalf("бумага: %+v %v", a, err)
	}
	// Бумага закрывает точку так же, как цифровая подпись: одна и та же
	// функция засчитывания этапа (FR-139).
	ce, dexp := decision("ENT01:F-010", 2, dom.ProfileGost, "ins-01@1")
	ce.CommandID, ce.Data, dexp.CommandID, dexp.Data = exp.CommandID, exp.Data, exp.CommandID, exp.Data
	d, err := w.svc.CheckCommand(as("INS-01"), w.signCmd(ce, "ins-01@1"), dexp)
	if err != nil {
		t.Fatal(err)
	}
	rule := dom.StageRule{Stage: 1, Need: 1, Level: 2, PaperAllowed: true}
	counted := func(x app.Accepted) dom.CountedSignature {
		return dom.CountedSignature{Stage: 1, DocDigest: digest, Method: x.Method, SignerPersonID: x.SignerPersonID, AuthorityOK: true,
			Level: 2, Verdict: x.Status, AttestedBy: x.AttestedBy, PaperOriginal: x.PaperOriginalNo}
	}
	if !dom.StageClosed(rule, digest, []dom.CountedSignature{counted(a)}) || !dom.StageClosed(rule, digest, []dom.CountedSignature{counted(d)}) {
		t.Fatal("бумага и цифра закрывают точку одинаково")
	}
	// Скан с чужим QR.
	foreign := []byte("QR:" + dom.QRText(uuid.NewV7().String(), dom.Digest([]byte("другое решение"))))
	faddr := dom.Digest(foreign)
	w.scans[faddr] = foreign
	if _, err := w.svc.CheckCommand(as("FOR-WC"), attest("FOR-WC", "INS-01", faddr), exp); code(err) != string(errcodes.SigningQrMismatch) {
		t.Fatalf("чужой QR: %v", err)
	}
	// Заверение самим подписантом.
	self := exp
	self.Actor = "INS-01"
	if _, err := w.svc.CheckCommand(as("INS-01"), attest("INS-01", "INS-01", addr), self); code(err) != string(errcodes.SigningAttesterIsSigner) {
		t.Fatalf("заверитель = подписант: %v", err)
	}
	// Бумага на этапе запрещена.
	forb := exp
	forb.PaperAllowed = false
	if _, err := w.svc.CheckCommand(as("FOR-WC"), attest("FOR-WC", "INS-01", addr), forb); code(err) != string(errcodes.SigningPaperForbidden) {
		t.Fatalf("бумага запрещена: %v", err)
	}
	// Заверитель без полномочия заверения.
	ins2 := exp
	ins2.Actor = "INS-02"
	if _, err := w.svc.CheckCommand(as("INS-02"), attest("INS-02", "INS-01", addr), ins2); code(err) != string(errcodes.SigningPaperForbidden) {
		t.Fatalf("без полномочия заверения: %v", err)
	}
	// Заверение чужим ключом (сеанс мастера, ключ начальника ОТК).
	if _, err := w.svc.CheckCommand(as("FOR-WC"), attest("HQC-01", "INS-01", addr), exp); code(err) != string(errcodes.SigningForeignKey) {
		t.Fatalf("заверение чужим ключом: %v", err)
	}
}

// actCmd — событие-команда акта ключа, подписанное refs (hybrid: gost + pq).
func (w *world) actCmd(t catalog.Type, id string, data any, refs ...string) []byte {
	return w.actCmdExtra(t, id, data, refs, nil)
}

// actCmdExtra — акт с подписями сверх перечня обязательных (подтверждение ротации).
func (w *world) actCmdExtra(t catalog.Type, id string, data any, refs, extra []string) []byte {
	raw, _ := json.Marshal(data)
	ce := dom.CommandEvent{EventType: string(t), CommandID: id, SourceID: "ADM-01", Data: raw, Level: 2,
		ClientSignedAt: w.clock.At().Format(app.TimeLayout), Profile: dom.ProfileHybrid, Signers: refs}
	c, _ := ce.Canonical()
	return w.sign(dom.ClassKeyAct, c, append(slices.Clone(refs), extra...)...)
}

func header(id string, sig []byte) platform.CommandHeader {
	h := platform.CommandHeader{CommandID: id}
	if sig != nil {
		_ = json.Unmarshal(sig, &h.Signature)
	}
	return h
}

// AD-11, FR-70, FR-79: выпуск ключа контролёра (доказательство владения,
// расписка, вторая подпись начальника ОТК), повторный ключ — тревога,
// ротация, отзыв со «скомпрометирован с X» и защитная реакция.
func TestKeyLifecycle(t *testing.T) {
	w := newWorld(t, false)
	ctx := as("ADM-01")
	nk, _ := profiles.Generate("ins-02-token@1", dom.ProfileGost)
	reg := app.RegisterKey{KeyRef: nk.Ref, SubjectKind: dom.SubjectPerson, SubjectID: "INS-02", ProfileID: dom.ProfileGost,
		Algorithm: dom.AlgGost, PublicKeyB64: nk.PublicB64(), PayloadClasses: []string{dom.ClassEvent, dom.ClassKeyAct},
		SubjectConfirmation: dom.ConfirmPaper, ValidFrom: t0.Format(time.RFC3339), ReceiptOriginalNo: "Р-17", ReceiptAttestedBy: "HQC-01"}
	data := app.RegistrationData{KeyRef: reg.KeyRef, SubjectKind: reg.SubjectKind, SubjectID: reg.SubjectID, ProfileID: reg.ProfileID,
		Algorithm: reg.Algorithm, PublicKeyB64: reg.PublicKeyB64, Fingerprint: nk.Fingerprint(), PayloadClasses: reg.PayloadClasses,
		SubjectConfirmation: reg.SubjectConfirmation, ValidFrom: reg.ValidFrom}
	pop := func(k *profiles.PrivateKey, d app.RegistrationData) string {
		c, _ := app.PoPContent(d)
		s, _ := k.SignPAE(dom.PayloadType(dom.ClassKeyAct, 1), dom.PAE(dom.PayloadType(dom.ClassKeyAct, 1), c))
		return b64(s)
	}
	reg.ProofOfPossessionB64 = pop(nk, data)
	data.ProofOfPossessionB64 = reg.ProofOfPossessionB64
	both := []string{"adm-01@1", "adm-01-pq@1", "hqc-01@1", "hqc-01-pq@1"}
	// Без второй подписи.
	id := uuid.NewV7().String()
	reg.CommandHeader = header(id, w.actCmd(catalog.KeyRegistrationRecorded, id, data, "adm-01@1", "adm-01-pq@1"))
	if _, err := w.svc.RegisterKey(ctx, reg); code(err) != string(errcodes.ApiValidationFailed) {
		t.Fatalf("без второй подписи: %v", err)
	}
	// Гибрид с одной подписью ГОСТ — понижение профиля (акты — hybrid).
	id = uuid.NewV7().String()
	reg.CommandHeader = header(id, w.actCmd(catalog.KeyRegistrationRecorded, id, data, "adm-01@1", "hqc-01@1"))
	if _, err := w.svc.RegisterKey(ctx, reg); code(err) != string(errcodes.SigningProfileDowngrade) {
		t.Fatalf("акт одной подписью ГОСТ: %v", err)
	}
	// Подделанное доказательство владения.
	bad := reg
	other, _ := profiles.Generate("x@1", dom.ProfileGost)
	bad.ProofOfPossessionB64 = pop(other, data)
	bd := data
	bd.ProofOfPossessionB64 = bad.ProofOfPossessionB64
	id = uuid.NewV7().String()
	bad.CommandHeader = header(id, w.actCmd(catalog.KeyRegistrationRecorded, id, bd, both...))
	if _, err := w.svc.RegisterKey(ctx, bad); code(err) != string(errcodes.SigningPackageTampered) {
		t.Fatalf("чужое доказательство владения: %v", err)
	}
	// Всё по правилам: акт записан, ключ в реестре.
	id = uuid.NewV7().String()
	reg.CommandHeader = header(id, w.actCmd(catalog.KeyRegistrationRecorded, id, data, both...))
	r, err := w.svc.RegisterKey(ctx, reg)
	if err != nil || r.Seq == 0 {
		t.Fatalf("регистрация: %v", err)
	}
	kd, err := w.svc.Key(ctx, nk.Ref, platform.Moment{})
	if err != nil || kd.Key.Status != "active" || kd.Key.Fingerprint != nk.Fingerprint() || len(kd.Signatures) != 4 {
		t.Fatalf("ключ в реестре: %+v %v", kd, err)
	}
	// Повторный ключ того же профиля — тревога duplicate_key.
	if !slices.Contains(w.alerts(), "duplicate_key") {
		t.Fatalf("нет тревоги «повторный ключ»: %v", w.alerts())
	}
	// Ротация: новый ключ подписан действующим ключом субъекта.
	w.keys = profiles.NewKeyring(append(w.all(), nk)...)
	rk, _ := profiles.Generate("ins-02-token@2", dom.ProfileGost)
	rd := data
	rd.KeyRef, rd.PublicKeyB64, rd.Fingerprint, rd.Rotates, rd.SubjectConfirmation = rk.Ref, rk.PublicB64(), rk.Fingerprint(), nk.Ref, dom.ConfirmRotation
	rd.ProofOfPossessionB64 = ""
	rd.ProofOfPossessionB64 = pop(rk, rd)
	rot := reg
	rot.KeyRef, rot.PublicKeyB64, rot.Rotates, rot.SubjectConfirmation, rot.ProofOfPossessionB64 = rk.Ref, rk.PublicB64(), nk.Ref, dom.ConfirmRotation, rd.ProofOfPossessionB64
	rot.ReceiptOriginalNo, rot.ReceiptAttestedBy = "", ""
	id = uuid.NewV7().String()
	rot.CommandHeader = header(id, w.actCmdExtra(catalog.KeyRegistrationRecorded, id, rd, both, []string{nk.Ref}))
	if _, err := w.svc.RegisterKey(ctx, rot); err != nil {
		t.Fatalf("ротация: %v", err)
	}
	old, _ := w.svc.Key(ctx, nk.Ref, platform.Moment{})
	if old.Key.Status != "expired" {
		t.Fatalf("старый ключ после ротации: %s", old.Key.Status)
	}
	// Решения контролёра ключом @2, потом отзыв «скомпрометирован с X».
	w.keys = profiles.NewKeyring(append(w.all(), rk)...)
	w.clock.Advance(time.Hour)
	ce, _ := decision("ENT01:F-020", 2, dom.ProfileGost, rk.Ref)
	w.appendDecision(catalog.Type(ce.EventType), "ENT01:F-020", w.signCmd(ce, rk.Ref))
	x := w.clock.At().Add(30 * time.Minute)
	w.clock.Advance(time.Hour)
	ce2, exp2 := decision("ENT01:F-021", 2, dom.ProfileGost, rk.Ref)
	exp2.Actor = "INS-02"
	ce2.SourceID = "INS-02"
	w.appendDecision(catalog.Type(ce2.EventType), "ENT01:F-021", w.signCmd(ce2, rk.Ref))
	w.clock.Advance(time.Minute)
	id = uuid.NewV7().String()
	rv := app.RevokeKey{CompromisedSince: x.Format(time.RFC3339), Reason: app.SigningReason{Code: "token_lost", Text: "токен утерян"}}
	rvd := app.RevocationData{KeyRef: rk.Ref, CompromisedSince: x.Format(app.TimeLayout)}
	rvd.Reason.Code, rvd.Reason.Text = "token_lost", "токен утерян"
	seqX, _ := w.j.Head(context.Background())
	rvd.CompromisedSinceSeq = seqX.MainSeq // решение F-021 — последняя запись до отзыва и после X
	rv.CommandHeader = header(id, w.actCmd(catalog.KeyRevocationRecorded, id, rvd, "adm-01@1", "adm-01-pq@1"))
	if _, err := w.svc.RevokeKey(ctx, rk.Ref, rv); err != nil {
		t.Fatalf("отзыв: %v", err)
	}
	ds, err := w.svc.Doubts(ctx, rk.Ref)
	if err != nil || len(ds) != 1 || ds[0].ItemID != "ENT01:F-021" || !ds[0].Contain || !ds[0].Resign {
		t.Fatalf("защитная реакция: %+v %v", ds, err)
	}
	// После отзыва подпись этим ключом не принимается (FR-79).
	ce3, exp3 := decision("ENT01:F-022", 2, dom.ProfileGost, rk.Ref)
	exp3.Actor = "INS-02"
	if _, err := w.svc.CheckCommand(as("INS-02"), w.signCmd(ce3, rk.Ref), exp3); code(err) != string(errcodes.SigningKeyRevoked) {
		t.Fatalf("после отзыва: %v", err)
	}
}

// AD-32, FR-76: смена профиля gost → hybrid для событий; старые подписи — по
// профилю на момент подписи; после смены gost — понижение профиля; попытка
// понизить профиль обратно отвергается.
func TestProfileSwitch(t *testing.T) {
	w := newWorld(t, true)
	ctx := as("AUD-01")
	ceOld, _ := decision("ENT01:F-030", 2, dom.ProfileGost, "ins-01@1")
	before := w.signCmd(ceOld, "ins-01@1")
	seqBefore := w.appendDecision(catalog.Type(ceOld.EventType), "ENT01:F-030", before)
	if _, err := w.svc.RegisterProfile(ctx, app.RegisterProfile{ProfileID: dom.ProfileHybrid, ObjectClasses: []string{dom.ClassEvent},
		Reason: app.SigningReason{Text: "переход на гибрид (О9)"}}); err != nil {
		t.Fatalf("смена профиля: %v", err)
	}
	if v, _, _ := w.svc.JudgeAt(context.Background(), before, seqBefore, t0); v.Status != dom.StatusValid {
		t.Fatalf("старая подпись по профилю на момент подписи: %+v", v)
	}
	ce, exp := decision("ENT01:F-031", 2, dom.ProfileGost, "ins-01@1")
	if _, err := w.svc.CheckCommand(as("INS-01"), w.signCmd(ce, "ins-01@1"), exp); code(err) != string(errcodes.SigningProfileDowngrade) {
		t.Fatalf("gost после смены: %v", err)
	}
	ch, exph := decision("ENT01:F-032", 2, dom.ProfileHybrid, "ins-01@1", "ins-01-pq@1")
	if _, err := w.svc.CheckCommand(as("INS-01"), w.signCmd(ch, "ins-01@1", "ins-01-pq@1"), exph); err != nil {
		t.Fatalf("hybrid после смены: %v", err)
	}
	// Удалили подпись ML-DSA из гибридного пакета — понижение профиля.
	var env dom.Envelope
	_ = json.Unmarshal(w.signCmd(ch, "ins-01@1", "ins-01-pq@1"), &env)
	env.Signatures = env.Signatures[:1]
	if _, err := w.svc.CheckCommand(as("INS-01"), env.Marshal(), exph); code(err) != string(errcodes.SigningProfileDowngrade) {
		t.Fatalf("гибрид без второй подписи: %v", err)
	}
	if _, err := w.svc.RegisterProfile(ctx, app.RegisterProfile{ProfileID: dom.ProfileGost, ObjectClasses: []string{dom.ClassEvent},
		Reason: app.SigningReason{Text: "назад"}}); code(err) != string(errcodes.SigningProfileDowngrade) {
		t.Fatalf("понижение профиля: %v", err)
	}
	pl, err := w.svc.Profiles(ctx, platform.Moment{})
	if err != nil || !slices.Contains(pl.Items[2].ObjectClasses, dom.ClassEvent) {
		t.Fatalf("профили: %+v %v", pl, err)
	}
}

// FR-66 (уровень 3), AD-12: сменный рапорт совпадает с журналом сервера;
// подпись, которой нет у сервера, — расхождение и тревога.
func TestShiftReport(t *testing.T) {
	w := newWorld(t, false)
	var leaves []dom.ShiftLeaf
	for i, item := range []string{"ENT01:F-040", "ENT01:F-041", "ENT01:F-042"} {
		ce, _ := decision(item, 2, dom.ProfileGost, "ins-01@1")
		raw := w.signCmd(ce, "ins-01@1")
		seq := w.appendDecision(catalog.Type(ce.EventType), item, raw)
		env, p, _ := dom.ParseEnvelope(raw)
		leaves = append(leaves, dom.ShiftLeaf{Digest: dom.SignatureDigest(env.PayloadType, p, env.Signatures[0]), EventType: ce.EventType, Seq: seq})
		_ = i
		w.clock.Advance(time.Minute)
	}
	report := func(ls []dom.ShiftLeaf) []byte {
		r := dom.BuildShiftReport(dom.ShiftReport{FormatVersion: 1, CryptoProfile: dom.ProfileGost, Signers: []string{"ins-01@1"},
			PersonID: "INS-01", ShiftID: "2026-09-21/1", WindowFrom: t0.Format(app.TimeLayout), WindowTo: t0.Add(8 * time.Hour).Format(app.TimeLayout)}, ls)
		c, _ := dom.CanonicalOf(r)
		return w.sign(dom.ClassShiftReport, c, "ins-01@1")
	}
	_, chk, err := w.svc.ShiftReport(as("INS-01"), app.SubmitShiftReport{CommandHeader: header(uuid.NewV7().String(), report(leaves))})
	if err != nil || !chk.Match || chk.ServerLeaf != 3 {
		t.Fatalf("рапорт совпадает: %+v %v", chk, err)
	}
	extra := append(slices.Clone(leaves), dom.ShiftLeaf{Digest: dom.Hash([]byte("подпись мимо сервера")), EventType: "inspection.result.recorded", Seq: 99})
	_, chk, err = w.svc.ShiftReport(as("INS-01"), app.SubmitShiftReport{CommandHeader: header(uuid.NewV7().String(), report(extra))})
	if err != nil || chk.Match || !slices.Contains(w.alerts(), "agent_journal_mismatch") {
		t.Fatalf("расхождение: %+v %v %v", chk, err, w.alerts())
	}
}

// FR-26, FR-70: адаптер реестра для приёма — устройство по акту ввода,
// отзыв прекращает приём, недоступный ключ — не «валидно».
func TestIngestKeys(t *testing.T) {
	w := newWorld(t, false)
	k := IngestKeys{Registry: w.svc.Registry(), Sources: []string{"ant-ingest"}}
	ctx := context.Background()
	if err := k.Source(ctx, "ws-2"); err != nil {
		t.Fatal(err)
	}
	if err := k.Source(ctx, "ws-9"); err != ingest.ErrUnknownSource {
		t.Fatalf("незарегистрированный источник: %v", err)
	}
	info, err := k.Key(ctx, "dev-ws2@1")
	if err != nil || info.SourceID != "ws-2" || info.Provenance != dom.ProvDevice || info.Revoked {
		t.Fatalf("%+v %v", info, err)
	}
	if _, err := k.Key(ctx, "nobody@1"); err != ingest.ErrUnknownKey {
		t.Fatalf("неизвестный ключ: %v", err)
	}
	if _, err := k.Key(ctx, "lost@1"); err == nil {
		t.Fatal("недоступный ключ принят")
	}
	// Отзыв ключа устройства (демо: защитное действие без подписи допускается).
	wd := newWorld(t, true)
	if _, err := wd.svc.RevokeKey(as("ADM-01"), "dev-ws2@1", app.RevokeKey{Reason: app.SigningReason{Text: "станок выведен"}}); err != nil {
		t.Fatal(err)
	}
	info, err = IngestKeys{Registry: wd.svc.Registry()}.Key(ctx, "dev-ws2@1")
	if err != nil || !info.Revoked {
		t.Fatalf("отозванный ключ устройства: %+v %v", info, err)
	}
	// Подпись устройства проверяется портом Verifier приёма.
	c := []byte(`{"a":1}`)
	raw := w.sign(dom.ClassEvent, c, "dev-ws2@1")
	res, err := profiles.Verifier{Keys: w.svc.Registry()}.Verify(ctx, raw)
	if err != nil || len(res.Signers) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
}
