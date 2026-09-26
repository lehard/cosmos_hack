package signing

import "time"

// Формы ответов модуля signing (AD-10, AD-11, AD-32, AD-33; FR-67…FR-70, FR-76, FR-79).
// Доверие ключу — от акта регистрации в журнале; закрытых ключей людей и
// устройств на сервере нет.

// KeyView — ключ в реестре: субъект, профиль, отпечаток, акты регистрации и отзыва.
type KeyView struct {
	KeyRef              string     `json:"key_ref" doc:"key_id@версия."`
	SubjectKind         string     `json:"subject_kind" enum:"person,device,engine,gateway,enterprise_gateway,keeper,verifier,demo_persona,partner_root"`
	SubjectID           string     `json:"subject_id"`
	ProfileID           string     `json:"profile_id" enum:"gost,pq"`
	Algorithm           string     `json:"algorithm" enum:"gost3410_2012_256_paramset_a,ml_dsa_65"`
	Fingerprint         string     `json:"fingerprint" doc:"Отпечаток открытого ключа; в сводке акта — «сверьте отпечаток» (AD-11)."`
	PayloadClasses      []string   `json:"payload_classes" doc:"Допустимые классы пакетов (AD-10)."`
	ProvenanceClass     string     `json:"provenance_class" enum:"personal,device,server_attested,scenario,genesis,partner" doc:"Класс доверия ключа (наследуется от регистрирующих подписей, AD-11)."`
	Status              string     `json:"status" enum:"active,revoked,expired,unavailable" doc:"unavailable — ключ недоступен: «не проверяемо», никогда не «валидно» (AD-32)."`
	ValidFrom           time.Time  `json:"valid_from"`
	ValidUntil          *time.Time `json:"valid_until,omitempty"`
	Rotates             string     `json:"rotates,omitempty" doc:"Ключ, который заменяет (ротация)."`
	RegistrationEventID string     `json:"registration_event_id" doc:"Акт регистрации key.registration.recorded."`
	RegistrationDocID   string     `json:"registration_document_id,omitempty" doc:"Документ акта с маршрутом подписей (AD-13)."`
	RevocationEventID   string     `json:"revocation_event_id,omitempty"`
	CompromisedSince    *time.Time `json:"compromised_since,omitempty" doc:"«Скомпрометирован с» — подписи позже под сомнением (AD-11)."`
}

// KeyList — реестр ключей.
type KeyList struct {
	Items      []KeyView `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

// KeyActSignature — подпись под актом регистрации или отзыва.
type KeyActSignature struct {
	SignerPersonID string    `json:"signer_person_id"`
	Role           string    `json:"role" doc:"Роль в акте: субъект, администратор безопасности, вторая подпись (AD-11)."`
	ProfileID      string    `json:"profile_id" enum:"gost,pq,hybrid"`
	SignedAt       time.Time `json:"signed_at"`
}

// KeyDetails — ключ с историей актов (FR-79).
type KeyDetails struct {
	Key        KeyView           `json:"key"`
	Signatures []KeyActSignature `json:"signatures" doc:"Подписи акта регистрации (владение, подтверждение субъекта, вторая подпись)."`
	History    []KeyView         `json:"history" doc:"Прежние версии ключа субъекта по ротации."`
	BasisSeq   int64             `json:"basis_seq"`
}

// CryptoProfile — криптопрофиль (contracts/crypto/profiles.yaml, AD-32).
type CryptoProfile struct {
	ProfileID          string   `json:"profile_id" enum:"gost,pq,hybrid"`
	Title              string   `json:"title"`
	Algorithm          string   `json:"algorithm,omitempty"`
	Components         []string `json:"components,omitempty" doc:"Составные профили гибрида."`
	SignaturesRequired int      `json:"signatures_required" minimum:"1"`
	Production         bool     `json:"production" doc:"Промышленный — через сертифицированное СКЗИ (MVP — не СКЗИ)."`
	ObjectClasses      []string `json:"object_classes" doc:"Объекты, для которых профиль обязателен."`
	EffectiveFromSeq   *int64   `json:"effective_from_seq,omitempty" doc:"С какой записи действует (key.profile.registered)."`
}

// CryptoProfileList — криптопрофили.
type CryptoProfileList struct {
	Items []CryptoProfile `json:"items"`
}

// PaperQRView — QR печатной рамки (AD-12, Д-30): текст и картинка SVG.
type PaperQRView struct {
	Text       string `json:"text" doc:"ant:doc:‹id›:‹отпечаток›."`
	DocumentID string `json:"document_id"`
	DocDigest  string `json:"doc_digest"`
	SVG        string `json:"svg" doc:"QR векторной картинкой для печатной рамки (без внешних ресурсов)."`
}

// ScanRead — скан подписанной распечатки для чтения QR.
type ScanRead struct {
	ImageB64 string `json:"image_b64" contentEncoding:"base64" maxLength:"16777216" doc:"Скан PNG или JPEG в base64."`
	// DocumentID, DocDigest — ожидаемый документ: QR сверяется с ним (чужой QR — signing.qr_mismatch).
	DocumentID string `json:"document_id,omitempty" maxLength:"128"`
	DocDigest  string `json:"doc_digest,omitempty" maxLength:"80"`
}

// ScanView — прочитанный QR скана.
type ScanView struct {
	Text        string `json:"text"`
	DocumentID  string `json:"document_id"`
	DocDigest   string `json:"doc_digest"`
	ScanAddress string `json:"scan_address" doc:"H(байты скана) — адрес в хранилище материалов (AD-23)."`
	Matches     bool   `json:"matches" doc:"QR совпал с ожидаемым документом."`
}
