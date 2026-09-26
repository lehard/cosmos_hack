package signing

import (
	"errors"
	"fmt"
	"strings"
)

// Бумага — второй равноправный путь подписи (FR-139, AD-43, PRD §11.16 п. 3):
// всегда подпись двух людей. Подписант ставит подпись ручкой на распечатке с
// QR (`ant:doc:‹id›:‹отпечаток›`); скан загружает и заверяет своей цифровой
// подписью уровня 2 второй человек — заверитель, заданный свойством шага BPMN
// (или этапом attester маршрута шаблона). Заверитель ≠ подписант. В записи —
// подписант, заверитель, адрес скана и учётный номер бумажного оригинала в
// архиве ОТК. Скан с чужим QR не принимается.

// QRPrefix — префикс QR документа (соглашения спайна, contracts/constants.yaml).
const QRPrefix = "ant:doc:"

// QRText — текст QR печатной рамки: ant:doc:‹id›:‹отпечаток› (AD-12).
func QRText(docID, digest string) string { return QRPrefix + docID + ":" + digest }

// ParseQR — id документа и отпечаток из текста QR. Отпечаток — последние два
// сегмента `streebog256:‹hex›`, id документа — всё между ними и префиксом.
func ParseQR(s string) (string, string, error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(s), QRPrefix)
	i := strings.LastIndex(rest, ":"+HashPrefix)
	if !ok || i <= 0 {
		return "", "", fmt.Errorf("%w: QR %q не документ ant", ErrQR, s)
	}
	id, dg := rest[:i], rest[i+1:]
	if _, err := ParseDigest(dg); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrQR, err)
	}
	return id, dg, nil
}

// Ошибки бумажного пути (коды contracts/errors.yaml).
var (
	ErrQR              = errors.New("signing.qr_mismatch")
	ErrAttesterSigner  = errors.New("signing.attester_is_signer")
	ErrPaperForbidden  = errors.New("signing.paper_forbidden")
	ErrPaperIncomplete = errors.New("signing: заверение бумаги неполно")
)

// PaperExpect — чего ждёт этап маршрута документа (или команда-решение).
type PaperExpect struct {
	DocumentID string
	DocDigest  string
	Stage      int
	// Signer — ожидаемый подписант этапа (кто подписывает ручкой).
	Signer string
	// PaperAllowed — этап допускает бумагу с заверением (маршрут шаблона, AD-43).
	PaperAllowed bool
	// AttesterAuthority — полномочие заверителя; пусто — заверитель не задан:
	// бумага на этапе запрещена.
	AttesterAuthority string
}

// PaperAttestation — заверение бумажной подписи (класс пакета paper-attestation,
// подписывает заверитель уровнем 2).
type PaperAttestation struct {
	FormatVersion   int      `json:"format_version"`
	CryptoProfile   string   `json:"crypto_profile"`
	Signers         []string `json:"signers"`
	DocumentID      string   `json:"document_id"`
	Version         int      `json:"version,omitempty"`
	DocDigest       string   `json:"doc_digest"`
	Stage           int      `json:"stage"`
	SignerPersonID  string   `json:"signer_person_id"`
	AttestedBy      string   `json:"attested_by"`
	ScanAddress     string   `json:"scan_address"`
	PaperOriginalNo string   `json:"paper_original_no"`
	SignatureLevel  int      `json:"signature_level"`
	SeenCheckpoint  int64    `json:"seen_checkpoint,omitempty"`
	ClientSignedAt  string   `json:"client_signed_at,omitempty"`
	WorkplaceID     string   `json:"workplace_id,omitempty"`
}

// PaperFacts — что сервер установил сам, не веря заверителю на слово:
// текст QR, прочитанный со скана, адрес скана H(байты), полномочие
// заверителя на позиции заверения.
type PaperFacts struct {
	ScanQR               string
	ScanAddress          string
	AttesterHasAuthority bool
}

// CheckPaper — AD-43, FR-139: заверение принимается, если бумага допустима на
// этапе и заверитель задан; QR на скане — этого документа и этого отпечатка
// (чужой QR — signing.qr_mismatch); заверение — уровня 2; заверитель ≠
// подписант (signing.attester_is_signer); подписант — ожидаемый подписант
// этапа; скан — тот, что загружен; есть учётный номер оригинала; у
// заверителя — полномочие заверения.
func CheckPaper(e PaperExpect, a PaperAttestation, f PaperFacts) error {
	if !e.PaperAllowed || e.AttesterAuthority == "" {
		return fmt.Errorf("%w: этап %d не допускает бумагу или заверитель не задан", ErrPaperForbidden, e.Stage)
	}
	qid, qdg, err := ParseQR(f.ScanQR)
	if err != nil {
		return err
	}
	if qid != e.DocumentID || qdg != e.DocDigest {
		return fmt.Errorf("%w: скан с QR %s не относится к документу %s (%s)", ErrQR, qdg, e.DocumentID, e.DocDigest)
	}
	if a.DocumentID != e.DocumentID || a.DocDigest != e.DocDigest || a.Stage != e.Stage {
		return fmt.Errorf("%w: заверение подписано над другим документом или этапом", ErrQR)
	}
	if a.SignatureLevel < Level2 {
		return fmt.Errorf("%w: заверение бумаги — подпись уровня 2", ErrLevel)
	}
	if a.AttestedBy == "" || a.AttestedBy == a.SignerPersonID {
		return fmt.Errorf("%w: бумажную подпись заверяет второй человек", ErrAttesterSigner)
	}
	if e.Signer != "" && a.SignerPersonID != e.Signer {
		return fmt.Errorf("%w: на этапе подписывает %s, а указан %s", ErrQR, e.Signer, a.SignerPersonID)
	}
	if a.ScanAddress == "" || a.ScanAddress != f.ScanAddress {
		return fmt.Errorf("%w: адрес скана не совпадает с загруженным", ErrPaperIncomplete)
	}
	if strings.TrimSpace(a.PaperOriginalNo) == "" || len(a.PaperOriginalNo) > 64 {
		return fmt.Errorf("%w: нет учётного номера бумажного оригинала", ErrPaperIncomplete)
	}
	if !f.AttesterHasAuthority {
		return fmt.Errorf("%w: у заверителя нет полномочия %s", ErrPaperForbidden, e.AttesterAuthority)
	}
	return nil
}

// Способы подписи (document.signature.recorded.method).
const (
	MethodTokenAgent = "token_agent"
	MethodPaper      = "paper"
	MethodDevice     = "device"
	MethodDemoSigner = "demo_signer"
)

// StageRule — этап маршрута подписей (AD-13, AD-43): полномочие, сколько
// подписей, уровень, допуск бумаги.
type StageRule struct {
	Stage       int
	AuthorityID string
	// Need — сколько разных подписантов (1, все = len(Signers), k из n).
	Need int
	// Signers — перечень подписантов этапа (для «все»); пусто — любой с полномочием.
	Signers      []string
	Level        int
	PaperAllowed bool
}

// CountedSignature — подпись документа, уже проверенная: криптография
// (Verdict) — для цифровой подписи самого подписанта, для бумаги — подписи
// заверителя; полномочие подписанта — на seq подписи (для бумаги — заверения).
type CountedSignature struct {
	Stage          int
	DocDigest      string
	Method         string
	SignerPersonID string
	AuthorityOK    bool
	Level          int
	Verdict        Status
	AttestedBy     string
	PaperOriginal  string
}

// Counts — подпись засчитывается в этап над отпечатком docDigest. Бумага с
// действительным заверением засчитывается так же, как подпись агентом (FR-139:
// «решение принимается и продвигает процесс так же»); демо-подписант — как
// подпись агентом (класс scenario различает верификатор).
func Counts(r StageRule, docDigest string, s CountedSignature) bool {
	if s.Stage != r.Stage || s.DocDigest != docDigest || !s.AuthorityOK || s.Verdict != StatusValid || s.Level < r.Level {
		return false
	}
	if len(r.Signers) > 0 && !containsStr(r.Signers, s.SignerPersonID) {
		return false
	}
	switch s.Method {
	case MethodTokenAgent, MethodDemoSigner, MethodDevice:
		return true
	case MethodPaper:
		return r.PaperAllowed && s.AttestedBy != "" && s.AttestedBy != s.SignerPersonID && s.PaperOriginal != ""
	}
	return false
}

// StageClosed — этап закрыт: засчитано Need разных подписантов (подписи
// прежней версии документа не засчитываются — другой отпечаток).
func StageClosed(r StageRule, docDigest string, sigs []CountedSignature) bool {
	need := r.Need
	if need <= 0 {
		need = 1
	}
	if len(r.Signers) > 0 && need > len(r.Signers) {
		need = len(r.Signers)
	}
	var who []string
	for _, s := range sigs {
		if Counts(r, docDigest, s) && !containsStr(who, s.SignerPersonID) {
			who = append(who, s.SignerPersonID)
		}
	}
	return len(who) >= need
}

func containsStr(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}
