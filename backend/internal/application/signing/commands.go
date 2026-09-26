package signing

import "ant/internal/application/platform"

// RegisterKey — акт регистрации ключа (key.registration.recorded, AD-11): новый
// ключ подписывает акт (доказательство владения), субъект подтверждает
// (подписью действующего ключа или бумажной распиской), вторая подпись — от
// независимой стороны. Закрытый ключ сервер не получает.
type RegisterKey struct {
	platform.CommandHeader
	KeyRef               string   `json:"key_ref" doc:"key_id@версия."`
	SubjectKind          string   `json:"subject_kind" enum:"person,device,engine,gateway,enterprise_gateway,keeper,verifier,demo_persona,partner_root"`
	SubjectID            string   `json:"subject_id" maxLength:"128"`
	ProfileID            string   `json:"profile_id" enum:"gost,pq"`
	Algorithm            string   `json:"algorithm" enum:"gost3410_2012_256_paramset_a,ml_dsa_65"`
	PublicKeyB64         string   `json:"public_key_b64" contentEncoding:"base64"`
	PayloadClasses       []string `json:"payload_classes" minItems:"1"`
	Rotates              string   `json:"rotates,omitempty" doc:"Заменяемый ключ (ротация)."`
	ProofOfPossessionB64 string   `json:"proof_of_possession_b64" contentEncoding:"base64" doc:"Подпись нового ключа над актом."`
	SubjectConfirmation  string   `json:"subject_confirmation" enum:"rotation_signature,paper_receipt"`
	DocumentID           string   `json:"document_id,omitempty" doc:"Документ акта с маршрутом подписей (AD-13)."`
	ReceiptOriginalNo    string   `json:"receipt_original_no,omitempty" maxLength:"64" doc:"Первичная выдача: учётный номер бумажной расписки субъекта с отпечатком ключа (архив ОТК, AD-11)."`
	ReceiptAttestedBy    string   `json:"receipt_attested_by,omitempty" maxLength:"64" doc:"Первичная выдача: кто заверил расписку (≠ субъект, AD-43)."`
	ValidFrom            string   `json:"valid_from" format:"date-time"`
	ValidUntil           string   `json:"valid_until,omitempty" format:"date-time"`
	KeyStorage           string   `json:"key_storage,omitempty" enum:"hardware_token,software_browser" doc:"Класс хранения ключа человека (AD-11, AD-14, Д-72): физический ключ или ключ в браузере под PIN."`
	StorageVariant       string   `json:"storage_variant,omitempty" enum:"extension,page" doc:"Где лежит ключ в браузере: расширение или хранилище страницы."`
}

// RevokeKey — акт отзыва ключа (key.revocation.recorded, AD-11): «скомпрометирован
// с X» порождает защитную реакцию — сдерживание «подпись под сомнением».
type RevokeKey struct {
	platform.CommandHeader
	CompromisedSince string        `json:"compromised_since,omitempty" format:"date-time" doc:"Скомпрометирован с (может быть раньше даты отзыва)."`
	Reason           SigningReason `json:"reason"`
	DocumentID       string        `json:"document_id,omitempty"`
}

// SigningReason — основание: код и текст.
type SigningReason struct {
	Code string `json:"code,omitempty" maxLength:"64"`
	Text string `json:"text" minLength:"1" maxLength:"2000"`
}

// RegisterProfile — смена обязательного криптопрофиля для классов пакетов
// (key.profile.registered, AD-32, FR-76): только повышение — понижение
// отвергается (signing.profile_downgrade); старые подписи проверяются по
// профилю на момент подписи.
type RegisterProfile struct {
	platform.CommandHeader
	ProfileID        string        `json:"profile_id" enum:"gost,pq,hybrid"`
	ObjectClasses    []string      `json:"object_classes" minItems:"1" doc:"Классы пакетов (contracts/crypto/payload-classes.yaml)."`
	EffectiveFromSeq int64         `json:"effective_from_seq,omitempty" minimum:"0" doc:"С какой позиции журнала обязателен; 0 — с позиции записи."`
	Reason           SigningReason `json:"reason"`
}
