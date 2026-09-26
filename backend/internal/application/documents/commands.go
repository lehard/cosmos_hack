package documents

import "ant/internal/application/platform"

// RequestDecision — «Запросить решение» (FR-146, FR-136): документ с маршрутом по
// шаблону для объекта; обязательные подписи вычисляет access.RequiredApprovals.
type RequestDecision struct {
	platform.CommandHeader
	Template       string            `json:"template" minLength:"1" maxLength:"128" doc:"Шаблон@версия из нормативного слоя."`
	Subject        platform.DrillRef `json:"subject" doc:"Объект решения."`
	SourceEventIDs []string          `json:"source_event_ids,omitempty" doc:"Записи-основания."`
	Decision       string            `json:"decision,omitempty" maxLength:"128" doc:"Решение, которое оформляется (например, disposition=scrap)."`
	Comment        string            `json:"comment,omitempty" maxLength:"2000"`
}

// SignDocument — подпись документа уровня 2 через агент токена (FR-66, AD-13,
// AD-14): пакет DSSE — в signature; агент сам посчитал отпечаток и сводку.
type SignDocument struct {
	platform.CommandHeader
	Version   int    `json:"version" minimum:"1" doc:"Подписываемая версия."`
	DocDigest string `json:"doc_digest" doc:"Отпечаток, который увидел подписант; расхождение — signing.document_changed."`
	Stage     int    `json:"stage" minimum:"1" doc:"Этап маршрута."`
}

// AttestPaper — заверение бумажной подписи (AD-43, FR-139): скан загружен в
// хранилище материалов, заверитель подписывает уровнем 2; заверитель ≠ подписант.
type AttestPaper struct {
	platform.CommandHeader
	Version         int    `json:"version" minimum:"1"`
	DocDigest       string `json:"doc_digest" doc:"Отпечаток из QR распечатки."`
	Stage           int    `json:"stage" minimum:"1"`
	SignerPersonID  string `json:"signer_person_id" doc:"Кто подписал ручкой (ожидаемый подписант этапа из QR)."`
	ScanAddress     string `json:"scan_address" doc:"Адрес скана в хранилище материалов (materials.material.upload)."`
	PaperOriginalNo string `json:"paper_original_no" minLength:"1" maxLength:"64" doc:"Учётный номер бумажного оригинала в архиве ОТК."`
}

// SetPaperStatus — статус бумажного экземпляра: напечатан / подписан / уничтожен (AD-12).
type SetPaperStatus struct {
	platform.CommandHeader
	Version     int    `json:"version" minimum:"1"`
	PaperStatus string `json:"paper_status" enum:"printed,signed,destroyed"`
	CopyNo      string `json:"copy_no,omitempty" maxLength:"32"`
}

// AnnulVersion — аннулирование версии документа новой записью (AD-12).
type AnnulVersion struct {
	platform.CommandHeader
	Version int            `json:"version" minimum:"1"`
	Reason  DocumentReason `json:"reason"`
}

// DocumentReason — основание: код и текст.
type DocumentReason struct {
	Code string `json:"code,omitempty" maxLength:"64"`
	Text string `json:"text" minLength:"1" maxLength:"2000"`
}
