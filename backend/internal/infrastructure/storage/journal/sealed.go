package journal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// Cipher — шифрование блока записи при хранении (AD-23, FR-75): конвертная
// схема — DEK на запись, обёртка DEK ключом KEK в journal.dek_wraps, AEAD
// блока с дополнительными данными chain ‖ seq ‖ commit. Реализация —
// infrastructure/security/atrest (KEK из отдельного тома). Без Cipher
// (WithCipher не задан) блок хранится открыто (dek_id = plain).
type Cipher interface {
	// AEAD — алгоритм блока (sealed_block.aead): aes_256_gcm | kuznyechik_mgm.
	AEAD() string
	// KEKID — идентификатор действующего KEK (обёртки ищутся по нему).
	KEKID() string
	// NewDEK — новый DEK, его идентификатор и обёртка действующим KEK.
	NewDEK() (dekID string, dek, wrapped []byte, err error)
	// Unwrap — DEK из обёртки действующим KEK.
	Unwrap(dekID string, wrapped []byte) ([]byte, error)
	// Seal — AEAD(dek, plain, aad): нонс и шифротекст с тегом.
	Seal(dek, plain, aad []byte) (nonce, ct []byte, err error)
	// Open — расшифрование и проверка тега.
	Open(dek, nonce, ct, aad []byte) ([]byte, error)
}

// WithCipher — шифрование блока записи при хранении (AD-23).
func WithCipher(c Cipher) Option { return func(s *Store) { s.cipher = c } }

// dekCache — расшифрованные DEK процесса: чтение страницы журнала подгружает
// обёртки одним запросом, Open берёт DEK отсюда. Предел — затем сброс.
type dekCache struct {
	mu sync.Mutex
	m  map[string][]byte
}

const dekCacheMax = 1 << 18

func (c *dekCache) get(id string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d, ok := c.m[id]
	return d, ok
}

func (c *dekCache) put(id string, dek []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || len(c.m) >= dekCacheMax {
		c.m = map[string][]byte{}
	}
	c.m[id] = dek
}

// sealBlock шифрует блок записи с уже вычисленными seq и commit.
func (s *Store) sealBlock(e jc.JournalEntry, salt, envelope []byte) (dekID string, nonce, ct, wrapped []byte, err error) {
	commit, err := dj.ParseDigest(e.Commit)
	if err != nil {
		return "", nil, nil, nil, err
	}
	pb, err := dj.PlainBlock(salt, envelope)
	if err != nil {
		return "", nil, nil, nil, err
	}
	dekID, dek, wrapped, err := s.cipher.NewDEK()
	if err != nil {
		return "", nil, nil, nil, err
	}
	nonce, ct, err = s.cipher.Seal(dek, pb, dj.AAD(string(e.Chain), int64(e.Seq), commit))
	if err != nil {
		return "", nil, nil, nil, err
	}
	s.deks.put(dekID, dek)
	return dekID, nonce, ct, wrapped, nil
}

// rawRow — сырые столбцы записи для decode.
type rawRow struct {
	header         string
	link           []byte
	salt, envelope []byte
	dekID, aead    *string
	nonce          []byte
}

const rawColumns = "header, link, salt, envelope, dek_id, aead, nonce"

func (r *rawRow) dest() []any {
	return []any{&r.header, &r.link, &r.salt, &r.envelope, &r.dekID, &r.aead, &r.nonce}
}

// decode собирает запись из заголовка, звена и блока sealed: открытый блок —
// plain_block в base64, зашифрованный — sealed_block (AD-23).
func decode(r rawRow) (jc.JournalEntry, error) {
	var e jc.JournalEntry
	if err := json.Unmarshal([]byte(r.header), &e); err != nil {
		return e, fmt.Errorf("заголовок записи: %w", err)
	}
	l, err := dj.DigestFromBytes(r.link)
	if err != nil {
		return e, err
	}
	e.Link = l.String()
	if r.dekID == nil {
		if e.Sealed, err = dj.PlainSealed(r.salt, r.envelope); err != nil {
			return e, err
		}
		return e, nil
	}
	aead := ""
	if r.aead != nil {
		aead = *r.aead
	}
	e.Sealed = jc.SealedBlock{Aead: jc.SealedBlockAead(aead), DekID: *r.dekID,
		NonceB64: base64.StdEncoding.EncodeToString(r.nonce), CiphertextB64: base64.StdEncoding.EncodeToString(r.envelope)}
	return e, nil
}

// preloadDEKs подгружает обёртки DEK записей страницы одним запросом.
func (s *Store) preloadDEKs(ctx context.Context, es []jc.JournalEntry) error {
	if s.cipher == nil {
		return nil
	}
	var ids []string
	for _, e := range es {
		if id := e.Sealed.DekID; id != "" && id != dj.SealedPlain {
			if _, ok := s.deks.get(id); !ok {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, "SELECT dek_id, wrapped FROM journal.dek_wraps WHERE kek_id = $1 AND dek_id = ANY($2)", s.cipher.KEKID(), ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var w []byte
		if err := rows.Scan(&id, &w); err != nil {
			return err
		}
		dek, err := s.cipher.Unwrap(id, w)
		if err != nil {
			return fmt.Errorf("обёртка DEK %s: %w", id, err)
		}
		s.deks.put(id, dek)
	}
	return rows.Err()
}

// dek — DEK записи: из кэша или из journal.dek_wraps.
func (s *Store) dek(ctx context.Context, id string) ([]byte, error) {
	if d, ok := s.deks.get(id); ok {
		return d, nil
	}
	var w []byte
	if err := s.pool.QueryRow(ctx, "SELECT wrapped FROM journal.dek_wraps WHERE kek_id = $1 AND dek_id = $2", s.cipher.KEKID(), id).Scan(&w); err != nil {
		return nil, fmt.Errorf("обёртка DEK %s под KEK %s: %w", id, s.cipher.KEKID(), err)
	}
	d, err := s.cipher.Unwrap(id, w)
	if err != nil {
		return nil, err
	}
	s.deks.put(id, d)
	return d, nil
}

// OpenUnverified — соль и конверт записи без сверки commit: верификатор
// (AD-9) сам сверяет commit и сообщает о расхождении как о нарушении, а
// переигрывает журнал по тому содержимому, которое лежит в базе.
func (s *Store) OpenUnverified(ctx context.Context, e jc.JournalEntry) (salt, envelope []byte, err error) {
	if e.Sealed.DekID == dj.SealedPlain {
		pb, err := base64.StdEncoding.DecodeString(e.Sealed.CiphertextB64)
		if err != nil {
			return nil, nil, fmt.Errorf("sealed: %w", err)
		}
		return dj.ParsePlainBlock(pb)
	}
	if s.cipher == nil {
		return nil, nil, app.ErrSealed
	}
	commit, err := dj.ParseDigest(e.Commit)
	if err != nil {
		return nil, nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(e.Sealed.NonceB64)
	if err != nil {
		return nil, nil, fmt.Errorf("sealed: nonce: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(e.Sealed.CiphertextB64)
	if err != nil {
		return nil, nil, fmt.Errorf("sealed: %w", err)
	}
	dek, err := s.dek(ctx, e.Sealed.DekID)
	if err != nil {
		return nil, nil, err
	}
	pb, err := s.cipher.Open(dek, nonce, ct, dj.AAD(string(e.Chain), int64(e.Seq), commit))
	if err != nil {
		return nil, nil, &dj.Break{Chain: string(e.Chain), Seq: e.Seq, Reason: "блок не расшифровывается (AEAD): " + err.Error()}
	}
	return dj.ParsePlainBlock(pb)
}

// Open — конверт записи с проверкой commit (AD-23: после расшифрования commit
// проверяется всегда).
func (s *Store) Open(ctx context.Context, e jc.JournalEntry) (app.Envelope, error) {
	salt, env, err := s.OpenUnverified(ctx, e)
	if err != nil {
		if errors.Is(err, dj.ErrSealed) {
			return app.Envelope{}, app.ErrSealed
		}
		return app.Envelope{}, err
	}
	if err := dj.VerifyCommit(e, salt, env); err != nil {
		return app.Envelope{}, err
	}
	return app.Envelope{Raw: env, Salt: salt}, nil
}
