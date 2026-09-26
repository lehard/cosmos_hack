package signing

import (
	"context"
	"encoding/base64"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
)

// PaperQR — QR печатной рамки документа (FR-139, AD-12, Д-30): текст
// ant:doc:‹id›:‹отпечаток› и картинка SVG. Печатная рамка в отрисовку
// документа не входит — отпечаток от неё не зависит.
func (s *Service) PaperQR(_ context.Context, text string) (PaperQRView, error) {
	if s.d.QRWriter == nil {
		return PaperQRView{}, platform.NotImplemented("signing.paper.qr")
	}
	id, dg, err := dom.ParseQR(text)
	if err != nil {
		return PaperQRView{}, fail(errcodes.ApiValidationFailed, err.Error(), "field", "text", "reason", "ожидается ant:doc:‹id›:‹отпечаток›")
	}
	svg, err := s.d.QRWriter.SVG(text)
	if err != nil {
		return PaperQRView{}, err
	}
	return PaperQRView{Text: text, DocumentID: id, DocDigest: dg, SVG: svg}, nil
}

// ReadScan — QR со скана (FR-139, AD-43): что напечатано на распечатке и
// тот ли это документ. Окончательную проверку при заверении делает CheckCommand.
func (s *Service) ReadScan(_ context.Context, in ScanRead) (ScanView, error) {
	if s.d.QR == nil {
		return ScanView{}, platform.NotImplemented("signing.paper.scan")
	}
	img, err := base64.StdEncoding.DecodeString(in.ImageB64)
	if err != nil {
		return ScanView{}, fail(errcodes.ApiValidationFailed, "скан не base64", "field", "image_b64", "reason", err.Error())
	}
	text, err := s.d.QR.ReadQR(img)
	if err != nil {
		return ScanView{}, fail(errcodes.SigningQrMismatch, err.Error(), "qr_digest", "—", "doc_digest", in.DocDigest)
	}
	id, dg, err := dom.ParseQR(text)
	if err != nil {
		return ScanView{}, fail(errcodes.SigningQrMismatch, err.Error(), "qr_digest", text, "doc_digest", in.DocDigest)
	}
	v := ScanView{Text: text, DocumentID: id, DocDigest: dg, ScanAddress: dom.Digest(img),
		Matches: (in.DocumentID == "" || in.DocumentID == id) && (in.DocDigest == "" || in.DocDigest == dg)}
	if !v.Matches {
		return v, fail(errcodes.SigningQrMismatch, "скан с QR "+dg+" не относится к документу "+in.DocumentID, "qr_digest", dg, "doc_digest", in.DocDigest)
	}
	return v, nil
}
