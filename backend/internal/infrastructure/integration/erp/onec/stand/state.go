package stand

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"ant/internal/infrastructure/integration/erp/onec"
)

// Message — сообщение журнала обмена stand-а (регистр входящих сообщений
// расширения: ключ идемпотентности → квитанция).
type Message struct {
	MessageID   string          `json:"message_id"`
	Resource    string          `json:"resource"`
	Kind        string          `json:"kind"`
	BusinessKey string          `json:"business_key"`
	Version     int             `json:"message_version"`
	Subject     string          `json:"subject"`
	Body        json.RawMessage `json:"body"`
	// Status — accepted | rejected.
	Status     string        `json:"status"`
	Receipt    *onec.Receipt `json:"receipt,omitempty"`
	Error      *onec.Error   `json:"error,omitempty"`
	HTTPStatus int           `json:"http_status"`
	ReceivedAt time.Time     `json:"received_at"`
	LastAt     time.Time     `json:"last_at"`
	Deliveries int           `json:"deliveries"`
	Lost       int           `json:"lost_responses,omitempty"`
	Faults     []string      `json:"faults,omitempty"`
}

// Document — документ 1С, созданный по сообщению.
type Document struct {
	RefKey        string    `json:"ref_key"`
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Number        string    `json:"number"`
	Date          time.Time `json:"date"`
	MessageID     string    `json:"message_id"`
	Posted        bool      `json:"posted"`
	Storno        bool      `json:"storno,omitempty"`
	StornoBy      string    `json:"storno_by,omitempty"`
	Subject       string    `json:"subject"`
	Lot           bool      `json:"lot,omitempty"`
	WarehouseFrom string    `json:"warehouse_from,omitempty"`
	WarehouseTo   string    `json:"warehouse_to,omitempty"`
	Quality       string    `json:"quality,omitempty"`
	Quantity      int       `json:"quantity,omitempty"`
	Comment       string    `json:"comment,omitempty"`
}

// Snapshot — состояние stand-а: журнал обмена, документы, созданные на
// странице этапы производства.
type Snapshot struct {
	Messages  []Message  `json:"messages"`
	Documents []Document `json:"documents"`
	Stages    []Row      `json:"stages,omitempty"`
}

// Row — строка сущности OData (реквизиты 1С).
type Row = map[string]any

// Store — хранение состояния stand-а (своя схема stand_onec, AD-18).
type Store interface {
	Load(ctx context.Context) (Snapshot, error)
	SaveMessage(ctx context.Context, m Message) error
	SaveDocument(ctx context.Context, d Document) error
	SaveStage(ctx context.Context, id string, r Row) error
}

// Memory — состояние в памяти процесса (тесты, запуск без БД).
type Memory struct {
	mu   sync.Mutex
	snap Snapshot
}

// Load — снимок.
func (m *Memory) Load(context.Context) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snap, nil
}

// SaveMessage — сохранить сообщение (по message_id).
func (m *Memory) SaveMessage(_ context.Context, x Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.snap.Messages {
		if m.snap.Messages[i].MessageID == x.MessageID {
			m.snap.Messages[i] = x
			return nil
		}
	}
	m.snap.Messages = append(m.snap.Messages, x)
	return nil
}

// SaveDocument — сохранить документ (по ref_key).
func (m *Memory) SaveDocument(_ context.Context, x Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.snap.Documents {
		if m.snap.Documents[i].RefKey == x.RefKey {
			m.snap.Documents[i] = x
			return nil
		}
	}
	m.snap.Documents = append(m.snap.Documents, x)
	return nil
}

// SaveStage — сохранить этап производства.
func (m *Memory) SaveStage(_ context.Context, id string, r Row) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.snap.Stages {
		if m.snap.Stages[i]["Ref_Key"] == id {
			m.snap.Stages[i] = r
			return nil
		}
	}
	m.snap.Stages = append(m.snap.Stages, r)
	return nil
}
