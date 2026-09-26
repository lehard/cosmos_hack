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
	ValidFrom            string   `json:"valid_from" format:"date-time"`
	ValidUntil           string   `json:"valid_until,omitempty" format:"date-time"`
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
