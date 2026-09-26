package federation

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/federation"
)

// Общее для live и fixtures (эпик 41): окно выписки из итога проверки и отказ
// приёма изменённой выписки.

// ViewOf — окно выписки (federation.extract.read) из итога самостоятельной
// проверки получателем (FR-132, AD-19): содержимое, подписи с итогом по
// каждой, контрольная точка, пакет для скачивания.
func ViewOf(v dom.Verification, raw []byte, direction, partner, address string, at time.Time, subject *platform.DrillRef) PassportExtractView {
	x := v.Extract
	e := PassportExtract{ExtractDigest: v.Digest, Direction: direction, PartnerCode: partner, OriginStatus: v.Origin,
		HeatNo: x.Origin.HeatNo, MaterialAddress: address, At: at.UTC(), Label: LabelOf(x), GlobalID: x.Subject.GlobalID}
	if direction == "outgoing" {
		e.OriginStatus = dom.OriginNotApplicable
	}
	if subject == nil {
		subject = SubjectOf(x, direction)
	}
	e.Subject = subject
	var content map[string]any
	_ = json.Unmarshal(v.Payload, &content)
	view := PassportExtractView{Extract: e, Content: content, Signatures: []PartnerSignature{}, Envelope: string(raw), OriginReason: v.Reason}
	for _, s := range v.Signatures {
		view.Signatures = append(view.Signatures, PartnerSignature{KeyRef: s.KeyRef, SignerRole: s.Role, Verification: s.Verification, Name: s.Name, Human: s.Human})
	}
	if c := x.Checkpoint; c != nil && c.HeadHash != "" {
		view.Checkpoint = c.Keeper + " · seq " + strconv.FormatInt(c.HeadSeq, 10) + " · " + c.HeadHash
	}
	return view
}

// SubjectOf — предмет выписки у нас: у входящей — наша партия (recipient_ref),
// у исходящей — наше изделие или партия по глобальному ID.
func SubjectOf(x dom.Extract, direction string) *platform.DrillRef {
	kind := platform.EntityKind(x.Subject.Kind)
	if kind == "" {
		kind = platform.EntityLot
	}
	switch {
	case direction == "incoming" && x.Subject.RecipientRef != "":
		return &platform.DrillRef{Entity: kind, ID: x.Subject.RecipientRef}
	case direction == "outgoing" && kind == platform.EntityItem:
		return &platform.DrillRef{Entity: kind, ID: x.Subject.GlobalID}
	case direction == "outgoing":
		if _, local, ok := dom.SplitGlobalID(x.Subject.GlobalID); ok {
			return &platform.DrillRef{Entity: kind, ID: local}
		}
	}
	return nil
}

// LabelOf — подпись предмета выписки для человека.
func LabelOf(x dom.Extract) string {
	l := x.Subject.Label
	if l == "" {
		l = x.Subject.GlobalID
	}
	if x.Origin.Material != "" {
		l += " · " + x.Origin.Material
	}
	return l
}

// RejectTampered — отказ приёма (FR-132): изменённая выписка — 422
// federation.extract_tampered; пакет не выписка — 422 api.validation.
func RejectTampered(err error) error {
	switch {
	case errors.Is(err, dom.ErrTampered):
		e := platform.Fail(errcodes.FederationExtractTampered)
		e.Detail = "Выписка паспорта изменена после подписи — не принята: " + err.Error()
		return e
	case errors.Is(err, dom.ErrFormat):
		e := platform.Fail(errcodes.FederationExtractTampered)
		e.Detail = "Пакет не выписка паспорта по формату passport-extract v1 — не принят: " + err.Error()
		return e
	}
	return err
}
