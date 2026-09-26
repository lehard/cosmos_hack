package materials

import "time"

// MaterialUpload — метаданные загружаемого материала (скан, кадр, вложение);
// байты идут телом запроса. Адрес — H(байты) `streebog256:…` (AD-23).
type MaterialUpload struct {
	MediaType      string
	Kind           string
	ItemID         string
	CapturedAt     *time.Time
	IsIllustration bool
	ProvenanceNote string
	Bytes          []byte
}

// MaterialInfo — материал по адресу содержимого (AD-23, FR-102, FR-139):
// в журнале — только адрес и метаданные (material.object.stored).
type MaterialInfo struct {
	MaterialAddress string     `json:"material_address" doc:"Адрес содержимого streebog256:…; повтор загрузки тех же байтов — тот же адрес."`
	MediaType       string     `json:"media_type"`
	SizeBytes       int64      `json:"size_bytes" minimum:"0"`
	Kind            string     `json:"kind" enum:"photo,video,illustration,protocol,log_excerpt,scan,quarantine_payload,other"`
	CapturedAt      *time.Time `json:"captured_at,omitempty"`
	ItemID          string     `json:"item_id,omitempty"`
	IsIllustration  bool       `json:"is_illustration" doc:"Иллюстрация, а не доказательство (NFR-UI-4)."`
	ProvenanceNote  string     `json:"provenance_note,omitempty"`
	StoredEventID   string     `json:"stored_event_id,omitempty" doc:"Запись material.object.stored."`
}

// MaterialContent — байты материала; адрес проверяется после чтения (AD-23).
type MaterialContent struct {
	MediaType string
	Bytes     []byte
}
