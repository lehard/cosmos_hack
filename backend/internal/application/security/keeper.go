package security

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	cc "ant/internal/contracts/crypto"
	"ant/internal/contracts/procs"
	dj "ant/internal/domain/journal"
)

// Протокол хранителя (contracts/internal/keeper.openapi.yaml, AD-8, AD-46):
// ant раз в N секунд передаёт головы обеих цепочек и звенья от прежней
// принятой головы; хранитель принимает голову, только если звенья сходятся,
// и выдаёт контрольную точку — пакет DSSE профиля hybrid. Верификатор сдаёт
// хранителю подписанный отчёт; ant забирает последний.

// HeadsSubmission — тело POST /v1/heads.
type HeadsSubmission struct {
	Heads           []Head `json:"heads"`
	Links           []Link `json:"links"`
	CommittedAtFrom string `json:"committed_at_from"`
	CommittedAtTo   string `json:"committed_at_to"`
}

// Head — голова цепочки.
type Head struct {
	Chain string `json:"chain"`
	Seq   int64  `json:"seq"`
	Link  string `json:"link"`
}

// Link — звено записи: commit, H(открытых полей), link (AD-44) — хранитель
// проверяет продолжение цепочки, не видя содержимого записей.
type Link struct {
	Chain          string `json:"chain"`
	Seq            int64  `json:"seq"`
	Commit         string `json:"commit"`
	OpenFieldsHash string `json:"open_fields_hash"`
	Link           string `json:"link"`
}

// Checkpoint — контрольная точка хранителя: пакет DSSE, его содержимое и отпечаток.
type Checkpoint struct {
	Envelope []byte
	Payload  cc.KeeperCheckpoint
	// Digest — отпечаток точки: streebog256 канонического payload.
	Digest string
}

// Head — голова цепочки chain в точке (seq 0 — цепочка пуста).
func (c Checkpoint) Head(chain string) (int64, string) {
	for _, h := range c.Payload.Heads {
		if string(h.Chain) == chain {
			return int64(h.Seq), h.Link
		}
	}
	return 0, dj.ZeroLink.String()
}

// Report — отчёт верификатора у хранителя: пакет DSSE, содержимое, отпечаток.
type Report struct {
	Envelope []byte
	Payload  procs.VerifierReportV1
	Digest   string
}

// KeeperAlarm — тревога хранителя (security.keeper.alert).
type KeeperAlarm struct {
	No     int64  `json:"no"`
	Alert  string `json:"alert"`
	Chain  string `json:"chain"`
	Seq    int64  `json:"last_seq"`
	Detail string `json:"detail"`
	At     string `json:"at"`
}

// KeeperStatus — состояние хранителя (GET /v1/status).
type KeeperStatus struct {
	LastSubmissionAt string        `json:"last_submission_at,omitempty"`
	IntervalSeconds  int           `json:"interval_seconds"`
	Checkpoints      int64         `json:"checkpoints"`
	Alarms           []KeeperAlarm `json:"alarms"`
}

// ErrNotFound — у хранителя этого ещё нет (404).
var ErrNotFound = errors.New("security: у хранителя нет запрошенного")

// RejectedError — хранитель отверг головы (409): звенья не сходятся или откат.
type RejectedError struct {
	Code   string
	Detail string
}

func (e *RejectedError) Error() string {
	return "хранитель отверг головы: " + e.Code + ": " + e.Detail
}

// Keeper — ведомый порт хранителя (AD-8, AD-46): mTLS-клиент
// keeper.openapi.yaml (infrastructure/integration/security/keeper).
type Keeper interface {
	SubmitHeads(ctx context.Context, s HeadsSubmission) (Checkpoint, error)
	LatestCheckpoint(ctx context.Context) (Checkpoint, error)
	Checkpoints(ctx context.Context, afterNo int64, limit int) ([]Checkpoint, error)
	SubmitReport(ctx context.Context, envelope []byte) error
	LatestReport(ctx context.Context) (Report, error)
	Reports(ctx context.Context, limit int) ([]Report, error)
	Report(ctx context.Context, digest string) (Report, error)
	Status(ctx context.Context) (KeeperStatus, error)
	// Links — звенья цепочки, принятые хранителем, после afterSeq.
	Links(ctx context.Context, chain string, afterSeq int64, limit int) ([]Link, error)
}

// PayloadOf — payload пакета DSSE и его отпечаток (streebog256 канонического payload).
func PayloadOf(envelope []byte) ([]byte, string, error) {
	var d DSSE
	if err := json.Unmarshal(envelope, &d); err != nil {
		return nil, "", fmt.Errorf("DSSE: %w", err)
	}
	p, err := base64.StdEncoding.DecodeString(d.Payload)
	if err != nil {
		return nil, "", fmt.Errorf("payload: %w", err)
	}
	return p, dj.H(p).String(), nil
}

// ParseCheckpoint разбирает пакет контрольной точки (подпись проверяет
// верификатор по trust-anchors; ant показывает «по данным сервера»).
func ParseCheckpoint(envelope []byte) (Checkpoint, error) {
	p, digest, err := PayloadOf(envelope)
	if err != nil {
		return Checkpoint{}, err
	}
	var c cc.KeeperCheckpoint
	if err := json.Unmarshal(p, &c); err != nil {
		return Checkpoint{}, fmt.Errorf("контрольная точка: %w", err)
	}
	return Checkpoint{Envelope: envelope, Payload: c, Digest: digest}, nil
}

// ParseReport разбирает пакет отчёта верификатора.
func ParseReport(envelope []byte) (Report, error) {
	p, digest, err := PayloadOf(envelope)
	if err != nil {
		return Report{}, err
	}
	var r procs.VerifierReportV1
	if err := json.Unmarshal(p, &r); err != nil {
		return Report{}, fmt.Errorf("отчёт верификатора: %w", err)
	}
	return Report{Envelope: envelope, Payload: r, Digest: digest}, nil
}
