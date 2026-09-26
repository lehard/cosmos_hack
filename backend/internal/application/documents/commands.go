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

// RequestVersion — «Запросить решение» / оформить версию документа
// (documents.version.request, FR-146, FR-136, AD-12): объект и действие (шаблон
// выбирает сервер) или шаблон явно. Для изделия — запись в его вход (версию
// оформляет свёртка), для объекта вне изделия — запрос и версия сразу.
type RequestVersion struct {
	platform.CommandHeader
	SubjectRef     string   `json:"subject_ref" minLength:"3" maxLength:"160" doc:"Объект: item:‹id›, nonconformity:‹id›, process_version:‹id›…"`
	Action         string   `json:"action,omitempty" maxLength:"128" doc:"Недоступное действие (x-ant-action), для которого запрашивается решение; по нему сервер выбирает шаблон."`
	TemplateRef    string   `json:"template_ref,omitempty" maxLength:"128" doc:"Шаблон ‹id›[@‹версия›] явно (traveler — новая версия сопроводительной карты)."`
	DocumentID     string   `json:"document_id,omitempty" maxLength:"128" doc:"Документ, новую версию которого запрашивают; пусто — новый документ."`
	Decision       string   `json:"decision,omitempty" maxLength:"128" doc:"Решение, которое оформляется."`
	Comment        string   `json:"comment,omitempty" maxLength:"2000"`
	SourceEventIDs []string `json:"source_event_ids,omitempty" maxItems:"50" doc:"Записи-основания."`
}

// RecordSignature — подпись этапа маршрута (documents.signature.record,
// FR-66, AD-13, AD-43): агентом токена (подпись над отпечатком) или в демо
// без агента (Д-30).
type RecordSignature struct {
	platform.CommandHeader
	Version      int    `json:"version" minimum:"1" doc:"Подписываемая версия."`
	Stage        int    `json:"stage" minimum:"1" doc:"Этап маршрута."`
	DocDigest    string `json:"doc_digest,omitempty" doc:"Отпечаток, который увидел подписант; расхождение — signing.document_changed. Пусто — отпечаток версии."`
	KeyRef       string `json:"key_ref,omitempty" maxLength:"128" doc:"Ключ подписанта key_id@версия (агент токена)."`
	SignatureB64 string `json:"signature_b64,omitempty" maxLength:"16384" doc:"Подпись агента над отпечатком (base64); проверяет signing (эпик 27). Пусто — демо без агента токена."`
}

// DeclineSignature — не согласовать версию, вернуть с замечанием
// (documents.signature.decline, FR-136).
type DeclineSignature struct {
	platform.CommandHeader
	Version   int    `json:"version" minimum:"1"`
	Stage     int    `json:"stage" minimum:"1"`
	DocDigest string `json:"doc_digest,omitempty"`
	Comment   string `json:"comment" minLength:"1" maxLength:"2000" doc:"Замечание — обязательно."`
}

// PrintPaper — напечатать бумажный экземпляр с QR (documents.paper.print,
// FR-139): у сопроводительной карты печать фиксирует новую версию, если
// содержимое изменилось с прошлой.
type PrintPaper struct {
	platform.CommandHeader
	Version int    `json:"version" minimum:"0" doc:"Версия; 0 — текущая (у карты — текущее содержимое)."`
	CopyNo  string `json:"copy_no,omitempty" maxLength:"32" doc:"Номер экземпляра."`
}
